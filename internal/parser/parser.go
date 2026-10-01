package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"log-analyzer/internal/model"
)

var patterns = []*regexp.Regexp{
	regexp.MustCompile(`^\{.*\}$`),
	regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})\s+\[(\w+)\]\s+(.+)$`),
	regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}).*\[([^\]]+)\].*"([^"]*)".*(\d{3})`),
}

// Compiled once, reused across calls.
var (
	reDigits = regexp.MustCompile(`\d+`)
	reUUID   = regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)
	reEmail  = regexp.MustCompile(`\b\w+@\w+\.\w+\b`)
)

func ParseFile(filename string) ([]model.LogEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var entries []model.LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		entry := model.LogEntry{Raw: line}

		if line[0] == '{' {
			var jsonEntry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &jsonEntry); err == nil {
				entry = parseJSONLog(jsonEntry, line)
				entries = append(entries, entry)
				continue
			}
		}

		for _, pattern := range patterns[1:] {
			if matches := pattern.FindStringSubmatch(line); matches != nil {
				entry = parseStructuredLog(matches, line)
				break
			}
		}

		if entry.Timestamp.IsZero() {
			entry = model.LogEntry{
				Timestamp: time.Now(),
				Level:     InferLogLevel(line),
				Message:   line,
				Raw:       line,
			}
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

func parseJSONLog(data map[string]interface{}, raw string) model.LogEntry {
	entry := model.LogEntry{Raw: raw}

	if ts, ok := data["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			entry.Timestamp = t
		}
	}
	if level, ok := data["level"].(string); ok {
		entry.Level = level
	}
	if msg, ok := data["message"].(string); ok {
		entry.Message = msg
	}
	if src, ok := data["source"].(string); ok {
		entry.Source = src
	}

	return entry
}

func parseStructuredLog(matches []string, raw string) model.LogEntry {
	entry := model.LogEntry{Raw: raw}

	if len(matches) >= 4 {
		if t, err := time.Parse("2006-01-02 15:04:05", matches[1]); err == nil {
			entry.Timestamp = t
		}
		entry.Level = matches[2]
		entry.Message = matches[3]
	}

	return entry
}

func InferLogLevel(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "error") || strings.Contains(lower, "fatal"):
		return "ERROR"
	case strings.Contains(lower, "warn"):
		return "WARN"
	case strings.Contains(lower, "debug"):
		return "DEBUG"
	default:
		return "INFO"
	}
}

// NormalizeErrorMessage replaces variable parts of an error message
// (numbers, UUIDs, emails) with placeholders so identical errors
// with different parameters collapse into the same pattern key.
func NormalizeErrorMessage(msg string) string {
	normalized := reUUID.ReplaceAllString(msg, "UUID")
	normalized = reEmail.ReplaceAllString(normalized, "EMAIL")
	normalized = reDigits.ReplaceAllString(normalized, "XXX")
	return normalized
}
