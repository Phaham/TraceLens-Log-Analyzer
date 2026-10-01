package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"

	"log-analyzer/internal/model"
	"log-analyzer/internal/parser"
)

type LogAnalyzer struct {
	LLM llms.Model
}

func New() (*LogAnalyzer, error) {
	llm, err := openai.New()
	if err != nil {
		return nil, fmt.Errorf("creating LLM: %w", err)
	}
	return &LogAnalyzer{LLM: llm}, nil
}

func (la *LogAnalyzer) Analyze(entries []model.LogEntry) (*model.LogAnalysis, error) {
	if len(entries) == 0 {
		return &model.LogAnalysis{}, nil
	}

	analysis := &model.LogAnalysis{
		TotalEntries: len(entries),
		TimeRange: model.TimeRange{
			Start: entries[0].Timestamp,
			End:   entries[len(entries)-1].Timestamp,
		},
	}

	var errorMessages []string
	for _, entry := range entries {
		switch strings.ToUpper(entry.Level) {
		case "ERROR", "FATAL":
			analysis.ErrorCount++
			errorMessages = append(errorMessages, entry.Message)
		case "WARN", "WARNING":
			analysis.WarningCount++
		}
	}

	analysis.TopErrors = findErrorPatterns(errorMessages)

	if err := la.performAIAnalysis(entries, analysis); err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	return analysis, nil
}

func findErrorPatterns(messages []string) []model.ErrorPattern {
	patternCounts := make(map[string]int)
	patternExamples := make(map[string]string)

	for _, msg := range messages {
		pattern := parser.NormalizeErrorMessage(msg)
		patternCounts[pattern]++
		if patternExamples[pattern] == "" {
			patternExamples[pattern] = msg
		}
	}

	type kv struct {
		Pattern string
		Count   int
	}

	var sorted []kv
	for k, v := range patternCounts {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	var result []model.ErrorPattern
	for i, item := range sorted {
		if i >= 10 {
			break
		}
		result = append(result, model.ErrorPattern{
			Pattern: item.Pattern,
			Count:   item.Count,
			Example: patternExamples[item.Pattern],
		})
	}

	return result
}

func (la *LogAnalyzer) performAIAnalysis(entries []model.LogEntry, analysis *model.LogAnalysis) error {
	sampleSize := 50
	if len(entries) < sampleSize {
		sampleSize = len(entries)
	}
	sample := entries[len(entries)-sampleSize:]

	template := prompts.NewPromptTemplate(`
You are an expert system administrator analyzing application logs. Based on the log data provided, identify:

1. **Anomalies**: Unusual patterns, spikes, or unexpected behaviors
2. **Recommendations**: Specific actions to improve system reliability
3. **Critical Issues**: Problems requiring immediate attention

Log Summary:
- Total Entries: {{.total_entries}}
- Errors: {{.error_count}}
- Warnings: {{.warning_count}}
- Time Range: {{.time_range}}

Top Error Patterns:
{{range .top_errors}}
- {{.Pattern}} ({{.Count}} occurrences)
{{end}}

Recent Log Sample:
{{range .sample}}
{{.timestamp}} [{{.level}}] {{.message}}
{{end}}

Respond in JSON format:
{
  "anomalies": [
    {
      "type": "error_spike|performance|security|other",
      "description": "What was detected",
      "severity": "critical|high|medium|low",
      "examples": ["example log entries"]
    }
  ],
  "recommendations": [
    "Specific actionable recommendations"
  ]
}`, []string{"total_entries", "error_count", "warning_count", "time_range", "top_errors", "sample"})

	sampleData := make([]map[string]string, len(sample))
	for i, entry := range sample {
		sampleData[i] = map[string]string{
			"timestamp": entry.Timestamp.Format(time.RFC3339),
			"level":     entry.Level,
			"message":   entry.Message,
		}
	}

	prompt, err := template.Format(map[string]any{
		"total_entries": analysis.TotalEntries,
		"error_count":   analysis.ErrorCount,
		"warning_count": analysis.WarningCount,
		"time_range":    fmt.Sprintf("%s to %s", analysis.TimeRange.Start.Format(time.RFC3339), analysis.TimeRange.End.Format(time.RFC3339)),
		"top_errors":    analysis.TopErrors,
		"sample":        sampleData,
	})
	if err != nil {
		return fmt.Errorf("formatting prompt: %w", err)
	}

	ctx := context.Background()
	response, err := la.LLM.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, llms.WithJSONMode())
	if err != nil {
		return fmt.Errorf("generating analysis: %w", err)
	}

	var aiResult struct {
		Anomalies       []model.Anomaly `json:"anomalies"`
		Recommendations []string        `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(response.Choices[0].Content), &aiResult); err != nil {
		return fmt.Errorf("parsing AI response: %w", err)
	}

	analysis.Anomalies = aiResult.Anomalies
	analysis.Recommendations = aiResult.Recommendations

	return nil
}
