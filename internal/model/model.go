package model

import "time"

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	Raw       string    `json:"raw"`
}

type LogAnalysis struct {
	TotalEntries    int            `json:"total_entries"`
	ErrorCount      int            `json:"error_count"`
	WarningCount    int            `json:"warning_count"`
	TopErrors       []ErrorPattern `json:"top_errors"`
	TimeRange       TimeRange      `json:"time_range"`
	Recommendations []string       `json:"recommendations"`
	Anomalies       []Anomaly      `json:"anomalies"`
}

type ErrorPattern struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
	Example string `json:"example"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Anomaly struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Examples    []string `json:"examples"`
}
