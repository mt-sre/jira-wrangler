package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

type Options struct {
	JiraToken             string
	JiraURL               string
	ConfigPath            string
	OverrideTemplatesPath string
	SecretsPath           string
}

func (o *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(
		&o.JiraToken,
		"jira-token",
		o.JiraToken,
		"JIRA Personal Access Token",
	)
	flags.StringVar(
		&o.JiraURL,
		"jira-url",
		o.JiraURL,
		"JIRA Server URL",
	)
	flags.StringVar(
		&o.ConfigPath,
		"config-file",
		o.ConfigPath,
		"Config file location",
	)
	flags.StringVar(
		&o.OverrideTemplatesPath,
		"override-templates-path",
		o.OverrideTemplatesPath,
		"Path to override templates",
	)
	flags.StringVar(
		&o.SecretsPath,
		"secrets-path",
		o.SecretsPath,
		"Path to directory containing secrets",
	)
}

func (o *Options) LoadSecrets() error {
	if o.JiraURL == "" {
		url, err := loadFromFile(filepath.Join(o.SecretsPath, "jira-url"))
		if err != nil {
			return fmt.Errorf("loading 'jira-url' from file: %w", err)
		}

		o.JiraURL = url
	}

	if o.JiraToken == "" {
		token, err := loadFromFile(filepath.Join(o.SecretsPath, "jira-token"))
		if err != nil {
			return fmt.Errorf("loading 'jira-token' from file: %w", err)
		}

		o.JiraToken = token
	}

	return nil
}

func loadFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %q: %w", path, err)
	}

	return string(data), nil
}
