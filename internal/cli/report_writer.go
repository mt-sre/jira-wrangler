package cli

import (
	"embed"
	"fmt"
	"io"
	"os"
	"text/template"
	"time"

	"github.com/thetechnick/jira-wrangler/internal/jira"
)

type ReportWriter interface {
	WriteReport(rpt Report) error
}

func NewReport(title string, groups ...Group) Report {
	now := time.Now().UTC()
	_, week := now.ISOWeek()

	return Report{
		Groups:     groups,
		Now:        now.Format(time.RFC822),
		Title:      title,
		WeekOfYear: fmt.Sprint(week),
	}
}

type Report struct {
	Groups     []Group
	Now        string
	Title      string
	WeekOfYear string
}

type Group struct {
	Title  string
	Issues []jira.Issue
}

//go:embed templates
var tmplFS embed.FS

func NewTemplatedReportWriter(out io.Writer, opts ...TemplatedReportWriterOption) (*TemplatedReportWriter, error) {
	var cfg TemplatedReportWriterConfig

	cfg.Option(opts...)

	templates, err := template.ParseFS(tmplFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing default templates: %w", err)
	}

	if cfg.OverrideTemplatePath != "" {
		templates, err = templates.ParseFS(os.DirFS(cfg.OverrideTemplatePath), "*.tmpl")
		if err != nil {
			return nil, fmt.Errorf("parsing override templates: %w", err)
		}
	}

	return &TemplatedReportWriter{
		out:       out,
		templates: templates,
	}, nil
}

type TemplatedReportWriter struct {
	out       io.Writer
	templates *template.Template
}

func (w *TemplatedReportWriter) WriteReport(rpt Report) error {
	return w.templates.ExecuteTemplate(w.out, "report", rpt)
}

type TemplatedReportWriterConfig struct {
	OverrideTemplatePath string
}

func (c *TemplatedReportWriterConfig) Option(opts ...TemplatedReportWriterOption) {
	for _, opt := range opts {
		opt.ConfigureTemplatedReportWriter(c)
	}
}

type TemplatedReportWriterOption interface {
	ConfigureTemplatedReportWriter(*TemplatedReportWriterConfig)
}
