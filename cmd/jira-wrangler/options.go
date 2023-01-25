package main

import (
	"github.com/spf13/pflag"
)

type Options struct {
	JiraToken             string
	JiraURL               string
	ConfigPath            string
	OverrideTemplatesPath string
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
}
