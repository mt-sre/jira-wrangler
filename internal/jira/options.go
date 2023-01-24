package jira

type WithBaseURL string

func (w WithBaseURL) ConfigureClient(c *ClientConfig) {
	c.BaseURL = string(w)
}
