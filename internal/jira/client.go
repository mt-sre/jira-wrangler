package jira

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	jira "github.com/andygrunwald/go-jira/v2/onpremise"
)

const (
	colorCustomFieldID         = "customfield_12320845"
	targetEndDateCustomFieldID = "customfield_12313942"
	reportCommentPrefix        = "[report]"
)

func NewClient(client *http.Client, opts ...ClientOption) (*Client, error) {
	var cfg ClientConfig

	cfg.Option(opts...)

	c, err := jira.NewClient(cfg.BaseURL, client)
	if err != nil {
		return nil, fmt.Errorf("initializing jira client: %w", err)
	}

	return &Client{
		c: c,
	}, nil
}

type Client struct {
	c *jira.Client
}

func (c *Client) GetIssue(ctx context.Context, key string) (Issue, error) {
	raw, _, err := c.c.Issue.Get(ctx, key, &jira.GetQueryOptions{})
	if err != nil {
		return Issue{}, fmt.Errorf("getting issue: %w", err)
	}

	return issueFromRaw(*raw), nil
}

func (c *Client) SearchIssues(ctx context.Context, jql string) ([]Issue, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	issues, _, err := c.c.Issue.Search(ctx, jql, &jira.SearchOptions{})
	if err != nil {
		return nil, fmt.Errorf("querying JIRA for issues: %w", err)
	}

	res := make([]Issue, 0, len(issues))
	for _, i := range issues {
		res = append(res, issueFromRaw(i))
	}

	return res, nil
}

func issueFromRaw(raw jira.Issue) Issue {
	return Issue{
		Key:           raw.Key,
		Color:         colorFromRaw(raw),
		Priority:      priorityFromRaw(raw),
		Status:        raw.Fields.Status.Name,
		StatusComment: statusCommentFromRaw(raw),
		Summary:       raw.Fields.Summary,
		TargetEnd:     customStringFieldFromRaw(raw, targetEndDateCustomFieldID),
	}
}

type Issue struct {
	Key           string
	Color         Color
	Priority      string
	Status        string
	StatusComment string
	Summary       string
	TargetEnd     string
}

func priorityFromRaw(raw jira.Issue) string {
	if raw.Fields.Priority == nil {
		return ""
	}
	if raw.Fields.Priority.Name == "Undefined" {
		return ""
	}
	return raw.Fields.Priority.Name
}

func colorFromRaw(raw jira.Issue) Color {
	colorField := raw.Fields.Unknowns[colorCustomFieldID]
	var c string
	if colorFieldMap, ok := colorField.(map[string]interface{}); ok {
		c = colorFieldMap["value"].(string)
	}

	return ParseColor(c)
}

func customStringFieldFromRaw(raw jira.Issue, customFieldID string) string {
	field := raw.Fields.Unknowns[customFieldID]
	if field == nil {
		return ""
	}

	return fmt.Sprint(field)
}

func statusCommentFromRaw(issue jira.Issue) string {
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

func ParseColor(raw string) Color {
	switch strings.ToLower(raw) {
	case "red":
		return ColorRed
	case "yellow":
		return ColorYellow
	case "green":
		return ColorGreen
	default:
		return ColorNone
	}
}

type Color string

func (c Color) String() string {
	return string(c)
}

func (c Color) Less(other Color) bool {
	return _colorOrd[c] < _colorOrd[other]
}

const (
	ColorNone   Color = ""
	ColorRed    Color = "Red"
	ColorYellow Color = "Yellow"
	ColorGreen  Color = "Green"
)

var _colorOrd = map[Color]int{
	ColorNone:   0,
	ColorRed:    1,
	ColorYellow: 2,
	ColorGreen:  3,
}

type ClientConfig struct {
	BaseURL string
}

func (c *ClientConfig) Option(opts ...ClientOption) {
	for _, opt := range opts {
		opt.ConfigureClient(c)
	}
}

type ClientOption interface {
	ConfigureClient(*ClientConfig)
}
