package rptx

import (
	"fmt"

	"github.com/xlibnetizen/rptx/utils"
)

func GenerateReport(title string, content []string) string {
	report := fmt.Sprintf("=== %s ===\n", title)
	for _, line := range content {
		report += fmt.Sprintf("- %s\n", line)
	}

	return report
}

func CreatePdf(title string, content []string) ([]byte, error) {
	return utils.GeneratePDF(nil, "Chào mừng")
}
