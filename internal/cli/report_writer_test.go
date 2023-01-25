package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetechnick/jira-wrangler/internal/jira"
)

func TestReportWriter_WriteReport(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		Report   Report
		Expected string
	}{
		"happy path": {
			Report: Report{
				Title:      "title",
				WeekOfYear: "1",
				Now:        "24 Jan 23 12:42 UTC",
				Groups: []Group{
					{
						Title: "title",
						Issues: []jira.Issue{
							{
								Color:     jira.ColorGreen,
								Key:       "MTSRE-1234",
								Status:    "In Progress",
								TargetEnd: "Now",
								Summary:   "Test",
							},
						},
					},
					{
						Title: "title 2",
						Issues: []jira.Issue{
							{
								Color:     jira.ColorYellow,
								Key:       "MTSRE-1235",
								Status:    "In Progress",
								TargetEnd: "Later",
								Summary:   "Test 2",
							},
							{
								Color:     jira.ColorGreen,
								Key:       "MTSRE-1236",
								Status:    "In Progress",
								TargetEnd: "Way Later",
								Summary:   "Test 3",
							},
						},
					},
				},
			},
			Expected: strings.Join([]string{
				"title",
				"Week 1 - 24 Jan 23 12:42 UTC",
				"",
				"title",
				"- [MTSRE-1234] Test",
				"  Status:\tIn Progress",
				"  Color:\tGreen",
				"  TargetEnd:\tNow",
				"title 2",
				"- [MTSRE-1235] Test 2",
				"  Status:\tIn Progress",
				"  Color:\tYellow",
				"  TargetEnd:\tLater",
				"- [MTSRE-1236] Test 3",
				"  Status:\tIn Progress",
				"  Color:\tGreen",
				"  TargetEnd:\tWay Later",
				"",
			}, "\n"),
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			rw, err := NewTemplatedReportWriter(&buf)
			require.NoError(t, err)

			require.NoError(t, rw.WriteReport(tc.Report))

			assert.Equal(t, tc.Expected, buf.String())
		})
	}
}
