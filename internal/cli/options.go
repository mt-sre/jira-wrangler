package cli

type WithOverrideTemplatePath string

func (w WithOverrideTemplatePath) ConfigureTemplatedReportWriter(c *TemplatedReportWriterConfig) {
	c.OverrideTemplatePath = string(w)
}
