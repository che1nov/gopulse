package domain

type ReportFormat string

const (
	ReportTerminal ReportFormat = "terminal"
	ReportMarkdown ReportFormat = "markdown"
	ReportJSON     ReportFormat = "json"
)
