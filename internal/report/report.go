package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"log-analyzer/internal/model"
)

func Print(a *model.LogAnalysis) {
	fmt.Printf("📊 Log Analysis Report\n")
	fmt.Printf("=====================\n\n")

	fmt.Printf("📈 Summary:\n")
	fmt.Printf("  Total Entries: %d\n", a.TotalEntries)
	fmt.Printf("  Errors: %d\n", a.ErrorCount)
	fmt.Printf("  Warnings: %d\n", a.WarningCount)
	fmt.Printf("  Time Range: %s to %s\n\n",
		a.TimeRange.Start.Format(time.DateOnly+" 15:04:05"),
		a.TimeRange.End.Format(time.DateOnly+" 15:04:05"))

	if len(a.TopErrors) > 0 {
		fmt.Printf("🔴 Top Error Patterns:\n")
		for i, pattern := range a.TopErrors {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s (%d occurrences)\n", i+1, pattern.Pattern, pattern.Count)
		}
		fmt.Println()
	}

	if len(a.Anomalies) > 0 {
		fmt.Printf("⚠️  Detected Anomalies:\n")
		for _, anomaly := range a.Anomalies {
			fmt.Printf("  %s - %s (%s)\n", anomaly.Type, anomaly.Description, anomaly.Severity)
		}
		fmt.Println()
	}

	if len(a.Recommendations) > 0 {
		fmt.Printf("💡 Recommendations:\n")
		for i, rec := range a.Recommendations {
			fmt.Printf("  %d. %s\n", i+1, rec)
		}
		fmt.Println()
	}
}

func WriteJSON(a *model.LogAnalysis, filename string) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Printf("📄 Report saved to %s\n", filename)
	return nil
}
