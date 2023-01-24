package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/onpremise"
	"github.com/spf13/cobra"
	"github.com/thetechnick/jira-wrangler/internal/cli"
	jirainternal "github.com/thetechnick/jira-wrangler/internal/jira"
)

func main() {
	opts := Options{
		ConfigPath: "config.yaml",
	}

	cmd := &cobra.Command{
		Use:  "jira-wrangler",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithCancel(cmd.Root().Context())
			defer cancel()

			tp := jira.BearerAuthTransport{
				Token: opts.JiraToken,
			}
			client, err := jirainternal.NewClient(
				tp.Client(),
				jirainternal.WithBaseURL(opts.JiraURL),
			)
			if err != nil {
				return fmt.Errorf("setting up JIRA client: %w", err)
			}

			cfg, err := cli.LoadConfig(opts.ConfigPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			now := time.Now().UTC()
			fmt.Println(cfg.Title)
			_, week := now.ISOWeek()
			fmt.Printf("Week %d - %s\n", week, now.Format(time.RFC822))
			fmt.Println()

			for _, report := range cfg.Reports {
				fmt.Println(report.Title)
				if err := generateReport(ctx, report, client); err != nil {
					return fmt.Errorf("generating report: %w", err)
				}
				fmt.Println()
			}

			return nil
		},
	}

	opts.AddFlags(cmd.Flags())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

type JiraClient interface {
	SearchIssues(ctx context.Context, jql string) ([]jirainternal.Issue, error)
	GetIssue(ctx context.Context, key string) (jirainternal.Issue, error)
}

func generateReport(ctx context.Context, cfg cli.ReportConfig, client JiraClient) error {
	jql := fmt.Sprintf(
		`project = "SDE" AND labels = %q AND Status in ("To Do","In Progress") ORDER BY priority DESC`,
		cfg.Label,
	)
	issues, err := client.SearchIssues(ctx, jql)
	if err != nil {
		return err
	}

	issuesByColor := groupIssuesByColor(issues)
	for _, issues := range issuesByColor {
		for _, i := range issues {
			issue, err := client.GetIssue(ctx, i.Key)
			if err != nil {
				return err
			}

			printIssue(issue)
		}
	}
	return nil
}

func printIssue(issue jirainternal.Issue) {
	fmt.Printf("- [%s] %s\n", issue.Key, issue.Summary)
	fmt.Printf("  Status:\t %s\n", issue.Status)

	if priority := issue.Priority; len(priority) > 0 {
		fmt.Printf("  Priority:\t %s\n", priority)
	}
	if color := issue.Color; len(color) > 0 {
		fmt.Printf("  Color:\t %s\n", color)
	}
	if targedEnd := issue.TargetEnd; len(targedEnd) > 0 {
		fmt.Printf("  Target end:\t %s\n", targedEnd)
	}
	if comment := issue.StatusComment; len(comment) > 0 {
		fmt.Printf("  Comment:\t %s\n", comment)
	}
}

var (
	colorRank = []jirainternal.Color{
		jirainternal.ColorRed,
		jirainternal.ColorYellow,
		jirainternal.ColorGreen,
	}
)

func groupIssuesByColor(issues []jirainternal.Issue) [][]jirainternal.Issue {
	issuesByColor := make([][]jirainternal.Issue, len(colorRank)+1)
	colorIndex := map[jirainternal.Color]int{}
	for i, color := range colorRank {
		colorIndex[color] = i
	}
	for _, issue := range issues {
		color := issue.Color

		i := len(issuesByColor) - 1
		if colorIndex, ok := colorIndex[color]; ok {
			i = colorIndex
		}
		issuesByColor[i] = append(issuesByColor[i], issue)
	}

	return issuesByColor
}
