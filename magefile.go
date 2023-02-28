//go:build mage
// +build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mt-sre/go-ci/command"
	"github.com/mt-sre/go-ci/container"
	"github.com/mt-sre/go-ci/git"
	"github.com/mt-sre/go-ci/web"
	cp "github.com/otiai10/copy"
)

const projectName = "jira-wrangler"

var _depBin = filepath.Join(_dependencyDir, "bin")

var _dependencyDir = func() string {
	if dir, ok := os.LookupEnv("DEPENDENCY_DIR"); ok {
		return dir
	}

	return filepath.Join(_projectRoot, ".cache", "dependencies")
}()

var _projectRoot = func() string {
	if root, ok := os.LookupEnv("PROJECT_ROOT"); ok {
		return root
	}

	topLevel, err := git.RevParse(context.Background(),
		git.WithRevParseFormat(git.RevParseFormatTopLevel),
	)
	if err != nil {
		panic(err)
	}

	return topLevel
}()

var _version = func() string {
	const zeroVer = "v0.0.0"

	if imageVersion, ok := os.LookupEnv("IMAGE_VERSION"); ok {
		return imageVersion
	}

	latest, err := git.LatestVersion(context.Background())
	if err != nil {
		if errors.Is(err, git.ErrNoTagsFound) {
			return zeroVer
		}

		panic(err)
	}

	return latest
}()

var _shortSHA = func() string {
	short, err := git.RevParse(context.Background(),
		git.WithRevParseFormat(git.RevParseFormatShort),
	)
	if err != nil {
		panic(err)
	}

	return short
}()

var _jiraURL = func() string {
	url := defaultJIRAURL
	if val, ok := os.LookupEnv("JIRA_URL"); ok {
		url = val
	}

	return url
}()

const defaultJIRAURL = "https://issues.redhat.com"

var _jiraToken = func() string {
	return os.Getenv("JIRA_TOKEN")
}()

var Aliases = map[string]interface{}{
	"build":     Build.CLI,
	"check":     All.Check,
	"install":   Build.Install,
	"release":   Release.Full,
	"run-hooks": Hooks.Run,
	"test":      All.Test,
}

type All mg.Namespace

func (All) Check(ctx context.Context) {
	mg.SerialCtxDeps(
		ctx,
		Check.Tidy,
		Check.Verify,
		Check.Lint,
	)
}

func (All) Test(ctx context.Context) {
	mg.CtxDeps(
		ctx,
		Test.Unit,
	)
}

type Check mg.Namespace

func (Check) Tidy(ctx context.Context) error {
	tidy := gocmd(
		command.WithArgs{"mod", "tidy"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := tidy.Run(); err != nil {
		return fmt.Errorf("starting tidy: %w", err)
	}

	if tidy.Success() {
		return nil
	}

	return fmt.Errorf("running tidy: %w", tidy.Error())
}

func (Check) Verify(ctx context.Context) error {
	verify := gocmd(
		command.WithArgs{"mod", "verify"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := verify.Run(); err != nil {
		return fmt.Errorf("starting verification: %w", err)
	}

	if verify.Success() {
		return nil
	}

	return fmt.Errorf("running verification: %w", verify.Error())
}

func (Check) Lint(ctx context.Context) error {
	mg.CtxDeps(ctx, Deps.UpdateGolangCILint)

	lint := golangci(
		command.WithArgs{"run",
			"--timeout=10m",
			"-E", "unused,gofmt,goimports,gosimple,staticcheck",
			"--skip-dirs-use-default",
			"--verbose",
		},
		command.WithContext{Context: ctx},
	)

	if err := lint.Run(); err != nil {
		return fmt.Errorf("starting linter: %w", err)
	}

	fmt.Fprint(os.Stdout, lint.CombinedOutput())

	if lint.Success() {
		return nil
	}

	return fmt.Errorf("running linter: %w", lint.Error())
}

var golangci = command.NewCommandAlias(filepath.Join(_depBin, "golangci-lint"))

type Test mg.Namespace

func (Test) Unit(ctx context.Context) error {
	test := gocmd(
		command.WithCurrentEnv(true),
		command.WithArgs{
			"test", "-v", "-tags=unit",
			"-cover", "-count=1", "-race", "-timeout", "15m", "./...",
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := test.Run(); err != nil {
		return fmt.Errorf("starting unit tests: %w", err)
	}

	if test.Success() {
		return nil
	}

	return fmt.Errorf("running unit tests: %w", test.Error())
}

func (Test) ApplyDev(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		populateImageReference,
		Deps.UpdateKustomize,
		Release.Image,
	)

	temp, err := os.MkdirTemp("", fmt.Sprintf("%s-apply-dev-*", projectName))
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}

	if err := cp.Copy(filepath.Join(_projectRoot, "config"), temp); err != nil {
		return fmt.Errorf("copying 'config' to temp directory: %w", err)
	}

	defer func() { _ = sh.Rm(temp) }()

	devOverlay := filepath.Join(temp, "overlays", "dev")

	setImage := kustomize(
		command.WithArgs{"edit", "set", "image", fmt.Sprintf("quay.io/mt-sre/jira-wrangler=%s", _imageReference)},
		command.WithConsoleOut(mg.Verbose()),
		command.WithWorkingDirectory(devOverlay),
	)

	if err := setImage.Run(); err != nil {
		return fmt.Errorf("starting to set %s image: %w", projectName, err)
	}

	if !setImage.Success() {
		return fmt.Errorf("setting %s image: %w", projectName, setImage.Error())
	}

	addSecret := kustomize(
		command.WithArgs{
			"edit", "add", "secret", projectName,
			"--from-literal", fmt.Sprintf("jira-url=%s", _jiraURL),
			"--from-literal", fmt.Sprintf("jira-token=%s", _jiraToken),
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithWorkingDirectory(devOverlay),
	)

	if err := addSecret.Run(); err != nil {
		return fmt.Errorf("starting to add %s secret: %w", projectName, err)
	}

	if !addSecret.Success() {
		return fmt.Errorf("adding %s secret: %w", projectName, addSecret.Error())
	}

	apply := kubectl(
		command.WithCurrentEnv(true),
		command.WithArgs{"apply", "-k", devOverlay},
		command.WithConsoleOut(mg.Verbose()),
	)

	if err := apply.Run(); err != nil {
		return fmt.Errorf("starting to apply kustomize 'dev' overlay: %w", err)
	}

	if apply.Success() {
		return nil
	}

	return fmt.Errorf("applying kustomize 'dev' overlay: %w", apply.Error())
}

var (
	kustomize = command.NewCommandAlias(filepath.Join(_depBin, "kustomize"))
	kubectl   = command.NewCommandAlias("kubectl")
)

type Build mg.Namespace

// Copies CLI binary to "$GOPATH/bin".
func (Build) Install(ctx context.Context) error {
	install := gocmd(
		command.WithArgs{
			"install", filepath.Join(_projectRoot, "cmd", projectName),
		},
	)

	if err := install.Run(); err != nil {
		return fmt.Errorf("starting to install %s: %w", projectName, err)
	}

	if install.Success() {
		return nil
	}

	return fmt.Errorf("installing %s: %w", projectName, install.Error())
}

func (Build) CLI(ctx context.Context) error {
	build := goreleaser(
		command.WithCurrentEnv(true),
		command.WithArgs{
			"build", "--rm-dist", "--single-target", "--snapshot",
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := build.Run(); err != nil {
		return fmt.Errorf("starting to build %s: %w", projectName, err)
	}

	if build.Success() {
		return nil
	}

	return fmt.Errorf("building %s: %w", projectName, build.Error())
}

func (Build) Clean() error {
	return sh.Rm(filepath.Join(_projectRoot, "dist"))
}

var gocmd = command.NewCommandAlias(mg.GoCmd())

type Release mg.Namespace

// Generates release artifacts and pushes to SCM.
func (Release) Full(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateGoReleaser,
		Release.Clean,
	)

	release := goreleaser(
		command.WithArgs{"release", "--rm-dist"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := release.Run(); err != nil {
		return fmt.Errorf("starting release: %w", err)
	}

	if release.Success() {
		return nil
	}

	return fmt.Errorf("releasing plugin: %w", release.Error())
}

// Generates release artifacts locally.
func (Release) Snapshot(ctx context.Context) error {
	mg.CtxDeps(
		ctx,
		Deps.UpdateGoReleaser,
		Release.Clean,
	)

	release := goreleaser(
		command.WithArgs{"release", "--rm-dist", "--snapshot"},
		command.WithCurrentEnv(true),
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := release.Run(); err != nil {
		return fmt.Errorf("starting release snapshot: %w", err)
	}

	if release.Success() {
		return nil
	}

	return fmt.Errorf("releasing snapshot: %w", release.Error())
}

// Image pushes the jira-wrangler container image to the target repo.
// The target image can be modified by setting the environment variables
// IMAGE_ORG and IMAGE_REPO.
func (Release) Image(ctx context.Context) {
	mg.CtxDeps(ctx, populateImageReference)
	mg.SerialCtxDeps(
		ctx,
		Release.Snapshot,
		mg.F(pushImage, fmt.Sprintf("%s:latest", _imageReference)),
		mg.F(pushImage, fmt.Sprintf("%s:%s", _imageReference, _version)),
		mg.F(pushImage, fmt.Sprintf("%s:%s-%s", _imageReference, _version, _shortSHA)),
	)
}

var errNoContainerRuntime = errors.New("no container runtime")

func pushImage(ctx context.Context, ref string) error {
	runtime, ok := container.Runtime()
	if !ok {
		return errNoContainerRuntime
	}

	push := command.NewCommand(runtime,
		command.WithContext{Context: ctx},
		command.WithConsoleOut(mg.Verbose()),
		command.WithArgs{"push", ref},
	)

	if err := push.Run(); err != nil {
		return fmt.Errorf("starting to push image %q: %w", ref, err)
	}

	if !push.Success() {
		return fmt.Errorf("pushing image %q: %w", ref, push.Error())
	}

	return nil
}

func (Release) Clean() error {
	return sh.Rm(filepath.Join(_projectRoot, "dist"))
}

var goreleaser = command.NewCommandAlias(filepath.Join(_depBin, "goreleaser"))

type Deps mg.Namespace

func (Deps) UpdateGolangCILint(ctx context.Context) {
	mg.CtxDeps(ctx, mg.F(Deps.updateGoDependency, "github.com/golangci/golangci-lint/cmd/golangci-lint"))
}

func (Deps) UpdateGoReleaser(ctx context.Context) {
	mg.CtxDeps(ctx, mg.F(Deps.updateGoDependency, "github.com/goreleaser/goreleaser"))
}

func (Deps) UpdateKustomize(ctx context.Context) {
	mg.CtxDeps(ctx, mg.F(Deps.updateGoDependency, "sigs.k8s.io/kustomize/kustomize/v4"))
}

func (Deps) updateGoDependency(ctx context.Context, src string) error {
	if err := setupDepsBin(); err != nil {
		return fmt.Errorf("creating dependencies bin directory: %w", err)
	}

	toolsDir := filepath.Join(_projectRoot, "tools")

	tidy := gocmd(
		command.WithArgs{"mod", "tidy"},
		command.WithWorkingDirectory(toolsDir),
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := tidy.Run(); err != nil {
		return fmt.Errorf("starting to tidy tools dir: %w", err)
	}

	if !tidy.Success() {
		return fmt.Errorf("tidying tools dir: %w", tidy.Error())
	}

	install := gocmd(
		command.WithArgs{"install", src},
		command.WithWorkingDirectory(toolsDir),
		command.WithCurrentEnv(true),
		command.WithEnv{"GOBIN": _depBin},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := install.Run(); err != nil {
		return fmt.Errorf("starting to install command from source %q: %w", src, err)
	}

	if !install.Success() {
		return fmt.Errorf("installing command from source %q: %w", src, install.Error())
	}

	return nil
}

func (Deps) UpdatePreCommit(ctx context.Context) error {
	if err := setupDepsBin(); err != nil {
		return fmt.Errorf("creating dependencies bin directory: %w", err)
	}

	const urlPrefix = "https://github.com/pre-commit/pre-commit/releases/download"

	// pinning to version 2.17.0 since 2.18.0+ requires python>=3.7
	const version = "2.17.0"

	out := filepath.Join(_depBin, "pre-commit")

	if _, err := os.Stat(out); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspecting output location %q: %w", out, err)
		}

		if err := web.DownloadFile(ctx, urlPrefix+fmt.Sprintf("/v%s/pre-commit-%s.pyz", version, version), out); err != nil {
			return fmt.Errorf("downloading pre-commit: %w", err)
		}
	}

	return os.Chmod(out, 0775)
}

func setupDepsBin() error {
	return os.MkdirAll(_depBin, 0o774)
}

type Hooks mg.Namespace

func (Hooks) Enable(ctx context.Context) error {
	mg.CtxDeps(ctx, Deps.UpdatePreCommit)

	install := precommit(
		command.WithArgs{"install"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := install.Run(); err != nil {
		return fmt.Errorf("starting to enable hooks: %w", err)
	}

	if install.Success() {
		return nil
	}

	return fmt.Errorf("enabling hooks: %w", install.Error())
}

func (Hooks) Disable(ctx context.Context) error {
	mg.CtxDeps(ctx, Deps.UpdatePreCommit)

	uninstall := precommit(
		command.WithArgs{"uninstall"},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("starting to disable hooks: %w", err)
	}

	if uninstall.Success() {
		return nil
	}

	return fmt.Errorf("disabling hooks: %w", uninstall.Error())
}

func (Hooks) Run(ctx context.Context) error {
	mg.CtxDeps(ctx, Deps.UpdatePreCommit)

	run := precommit(
		command.WithArgs{
			"run",
			"--show-diff-on-failure",
			"--from-ref", "origin/main", "--to-ref", "HEAD",
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := run.Run(); err != nil {
		return fmt.Errorf("starting to run hooks: %w", err)
	}

	if run.Success() {
		return nil
	}

	return fmt.Errorf("running hooks: %w", run.Error())
}

func (Hooks) RunAllFiles(ctx context.Context) error {
	mg.CtxDeps(ctx, Deps.UpdatePreCommit)

	runAll := precommit(
		command.WithArgs{
			"run", "--all-files",
		},
		command.WithConsoleOut(mg.Verbose()),
		command.WithContext{Context: ctx},
	)

	if err := runAll.Run(); err != nil {
		return fmt.Errorf("starting to run hooks for all files: %w", err)
	}

	if runAll.Success() {
		return nil
	}

	return fmt.Errorf("running hooks for all files: %w", runAll.Error())
}

var precommit = command.NewCommandAlias(filepath.Join(_depBin, "pre-commit"))

var errUnsetEnvVar = errors.New("unset environment variable")

func populateImageReference(ctx context.Context) error {
	if val, ok := os.LookupEnv("IMAGE_REGISTRY"); ok {
		_imageReference = val
	} else {
		return fmt.Errorf("IMAGE_REGISTRY: %w", errUnsetEnvVar)
	}
	if val, ok := os.LookupEnv("IMAGE_ORG"); ok {
		_imageReference = path.Join(_imageReference, val)
	} else {
		return fmt.Errorf("IMAGE_ORG: %w", errUnsetEnvVar)
	}
	if val, ok := os.LookupEnv("IMAGE_REPO"); ok {
		_imageReference = path.Join(_imageReference, val)
	} else {
		_imageReference = path.Join(_imageReference, projectName)
	}

	return nil
}

var _imageReference string
