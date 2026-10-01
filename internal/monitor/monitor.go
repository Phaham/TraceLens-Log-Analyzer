package monitor

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/prompts"

	"log-analyzer/internal/analyzer"
	"log-analyzer/internal/model"
	"log-analyzer/internal/parser"
)

type Thresholds struct {
	ErrorsPerMinute   int
	CriticalKeywords  []string
	ResponseTimeLimit time.Duration
}

var DefaultThresholds = Thresholds{
	ErrorsPerMinute:   10,
	CriticalKeywords:  []string{"fatal", "out of memory", "database down"},
	ResponseTimeLimit: 5 * time.Second,
}

type LogMonitor struct {
	analyzer   *analyzer.LogAnalyzer
	watcher    *fsnotify.Watcher
	alertChain chains.Chain
	thresholds Thresholds
}

func New(a *analyzer.LogAnalyzer) (*LogMonitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	alertChain := chains.NewLLMChain(a.LLM, prompts.NewPromptTemplate(`
Generate a concise alert message for this log analysis:

{{.analysis}}

Format as: [SEVERITY] Brief description - Action needed
Keep under 140 characters.`, []string{"analysis"}))

	return &LogMonitor{
		analyzer:   a,
		watcher:    watcher,
		alertChain: alertChain,
		thresholds: DefaultThresholds,
	}, nil
}

func (lm *LogMonitor) Start(filename string) error {
	if err := lm.watcher.Add(filename); err != nil {
		return err
	}

	fmt.Printf("🚨 Monitoring %s for critical issues...\n", filename)

	for {
		select {
		case event, ok := <-lm.watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				go lm.checkForAlerts(filename)
			}
		case err, ok := <-lm.watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (lm *LogMonitor) checkForAlerts(filename string) {
	entries, err := parser.ParseFile(filename)
	if err != nil {
		log.Printf("Error parsing file: %v", err)
		return
	}

	recent := getRecentEntries(entries, time.Minute)
	if !lm.shouldAlert(recent) {
		return
	}

	analysis, err := lm.analyzer.Analyze(recent)
	if err != nil {
		log.Printf("Error analyzing logs: %v", err)
		return
	}

	alert, err := chains.Run(context.Background(), lm.alertChain,
		fmt.Sprintf("Analysis: %+v", analysis))
	if err != nil {
		log.Printf("Error generating alert: %v", err)
		return
	}

	fmt.Printf("🚨 ALERT: %s\n", alert)
}

func getRecentEntries(entries []model.LogEntry, duration time.Duration) []model.LogEntry {
	cutoff := time.Now().Add(-duration)
	var recent []model.LogEntry

	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Timestamp.Before(cutoff) {
			break
		}
		recent = append([]model.LogEntry{entries[i]}, recent...)
	}

	return recent
}

func (lm *LogMonitor) shouldAlert(entries []model.LogEntry) bool {
	errorCount := 0
	for _, entry := range entries {
		if entry.Level == "ERROR" || entry.Level == "FATAL" {
			errorCount++
		}

		for _, keyword := range lm.thresholds.CriticalKeywords {
			if strings.Contains(strings.ToLower(entry.Message), keyword) {
				return true
			}
		}
	}

	return errorCount >= lm.thresholds.ErrorsPerMinute
}
