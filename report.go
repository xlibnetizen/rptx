package rptx

import "fmt"

func GenerateReport(title string, content []string) string {
	report := fmt.Sprintf("=== %s ===\n", title)
	for _, line := range content {
		report += fmt.Sprintf("- %s\n", line)
	}
	return report
}
