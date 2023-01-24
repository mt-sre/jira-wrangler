package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/onpremise"
	"sigs.k8s.io/yaml"
)

const (
	colorCustomFieldID         = "customfield_12320845"
	targetEndDateCustomFieldID = "customfield_12313942"
	reportCommentPrefix        = "[report]"
)

var (
	colorRank = []string{"Red", "Yellow", "Green"}
)

type Config struct {
	Title   string         `json:"title"`
	Reports []ReportConfig `json:"reports"`
}

type ReportConfig struct {
	Title string `json:"title"`
	Label string `json:"label"`
}

func main() {
	var (
		jiraToken  string
		jiraURL    string
		configFile string
	)
	flag.StringVar(&jiraToken, "jira-token", "", "JIRA Personal Access Token")
	flag.StringVar(&jiraURL, "jira-url", "", "JIRA Server URL")
	flag.StringVar(&configFile, "config-file", "config.yaml", "Config file location")

	if len(configFile) == 0 {
		configFile = "config.yaml"
	}

	flag.Parse()

	tp := jira.BearerAuthTransport{
		Token: jiraToken,
	}
	client, err := jira.NewClient(jiraURL, tp.Client())
	if err != nil {
		panic(err)
	}

	var config Config
	configFileContent, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(configFileContent, &config); err != nil {
		panic(err)
	}

	now := time.Now().UTC()
	fmt.Println(config.Title)
	_, week := now.ISOWeek()
	fmt.Printf("Week %d - %s\n", week, now.Format(time.RFC822))
	fmt.Println()

	ctx := context.Background()
	for _, report := range config.Reports {
		fmt.Println(report.Title)
		if err := generateReport(ctx, report, client); err != nil {
			panic(err)
		}
		fmt.Println()
	}
}

func generateReport(ctx context.Context, config ReportConfig, client *jira.Client) error {
	jql := fmt.Sprintf(
		`project = "SDE" AND labels = %q AND Status in ("To Do","In Progress") ORDER BY priority DESC`,
		config.Label,
	)
	issues, _, err := client.Issue.Search(ctx, jql, &jira.SearchOptions{})
	if err != nil {
		return err
	}

	issuesByColor := groupIssuesByColor(issues)
	for _, issues := range issuesByColor {
		for _, i := range issues {
			issue, _, err := client.Issue.Get(ctx, i.Key, &jira.GetQueryOptions{})
			if err != nil {
				return err
			}

			printIssue(issue)
		}
	}
	return nil
}

func printIssue(issue *jira.Issue) {
	fmt.Printf("- [%s] %s\n", issue.Key, issue.Fields.Summary)
	fmt.Printf("  Status:\t %s\n", issue.Fields.Status.Name)

	if priority := getPriority(issue); len(priority) > 0 {
		fmt.Printf("  Priority:\t %s\n", priority)
	}
	if color := getColor(issue); len(color) > 0 {
		fmt.Printf("  Color:\t %s\n", color)
	}
	if targedEnd := getCustomField(issue, targetEndDateCustomFieldID); len(targedEnd) > 0 {
		fmt.Printf("  Target end:\t %s\n", targedEnd)
	}
	if comment := statusComment(issue); len(comment) > 0 {
		fmt.Printf("  Comment:\t %s\n", comment)
	}
}

func getColor(issue *jira.Issue) string {
	colorField := issue.Fields.Unknowns[colorCustomFieldID]
	var c string
	if colorFieldMap, ok := colorField.(map[string]interface{}); ok {
		c = colorFieldMap["value"].(string)
	}
	if c == "Not Selected" {
		return ""
	}
	return c
}

func getPriority(issue *jira.Issue) string {
	if issue.Fields.Priority == nil {
		return ""
	}
	if issue.Fields.Priority.Name == "Undefined" {
		return ""
	}
	return issue.Fields.Priority.Name
}

func getCustomField(issue *jira.Issue, customFieldID string) string {
	field := issue.Fields.Unknowns[customFieldID]
	if field == nil {
		return ""
	}
	return fmt.Sprintf("%s", field)
}

func statusComment(issue *jira.Issue) string {
	if issue.Fields.Comments == nil {
		return ""
	}
	var latestReportComment string
	for _, c := range issue.Fields.Comments.Comments {
		trimmedBody := strings.TrimSpace(c.Body)
		if strings.HasPrefix(trimmedBody, reportCommentPrefix) {
			latestReportComment = strings.TrimSpace(trimmedBody[len(reportCommentPrefix):])
		}
	}
	latestReportComment = strings.ReplaceAll(latestReportComment, "\n", "")
	return strings.ReplaceAll(latestReportComment, "\r", "")
}

func groupIssuesByColor(issues []jira.Issue) [][]jira.Issue {
	issuesByColor := make([][]jira.Issue, len(colorRank)+1)
	colorIndex := map[string]int{}
	for i, color := range colorRank {
		colorIndex[color] = i
	}
	for _, issue := range issues {
		color := getColor(&issue)

		i := len(issuesByColor) - 1
		if colorIndex, ok := colorIndex[color]; ok {
			i = colorIndex
		}
		issuesByColor[i] = append(issuesByColor[i], issue)
	}

	return issuesByColor
}
