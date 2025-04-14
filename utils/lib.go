package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/signintech/gopdf"
	"github.com/xuri/excelize/v2"
)

const (
	BUFFER_SIZE = 1024 * 1024
)

const (
	// Text
	CONTENT_TYPE_PLAIN_TEXT = "text/plain"
	CONTENT_TYPE_CSV        = "text/csv"
	CONTENT_TYPE_HTML       = "text/html"
	CONTENT_TYPE_CSS        = "text/css"
	CONTENT_TYPE_JAVASCRIPT = "text/javascript"
	// Application
	CONTENT_TYPE_JSON         = "application/json"
	CONTENT_TYPE_XML          = "application/xml"
	CONTENT_TYPE_PDF          = "application/pdf"
	CONTENT_TYPE_ZIP          = "application/zip"
	CONTENT_TYPE_GZIP         = "application/gzip"
	CONTENT_TYPE_OCTET_STREAM = "application/octet-stream"
	// Microsoft Office
	CONTENT_TYPE_EXCEL = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	CONTENT_TYPE_WORD  = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	CONTENT_TYPE_PPT   = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	// Images
	CONTENT_TYPE_JPEG = "image/jpeg"
	CONTENT_TYPE_PNG  = "image/png"
	CONTENT_TYPE_GIF  = "image/gif"
	CONTENT_TYPE_SVG  = "image/svg+xml"
	CONTENT_TYPE_WEBP = "image/webp"
	// Video
	CONTENT_TYPE_MP4  = "video/mp4"
	CONTENT_TYPE_WEBM = "video/webm"
	CONTENT_TYPE_AVI  = "video/x-msvideo"
	// Audio
	CONTENT_TYPE_MP3 = "audio/mpeg"
	CONTENT_TYPE_WAV = "audio/wav"
	CONTENT_TYPE_OGG = "audio/ogg"
)

type ExcelConfig struct {
	SheetName string
	Headers   [][]interface{}
	Data      [][]interface{}
	Formats   map[string]string
}

func AddDataToRequest(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), "ExportType", "EXCEL")
	ctx = context.WithValue(ctx, "ReportCode", "ACC816_FINANCIAL_LEDGER_EXCEL")
	return r.WithContext(ctx)
}

func AddDataToBody(r *http.Request, extraData map[string]interface{}) {
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	bodyBytes, err := io.ReadAll(tee)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}

	// Restore the original request body
	r.Body = io.NopCloser(&buf)

	// Convert body to map
	var bodyMap map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			fmt.Println("Error parsing body JSON:", err)
			return
		}
	} else {
		bodyMap = make(map[string]interface{})
	}

	// Add extra data to the map
	for key, value := range extraData {
		bodyMap[key] = value
	}

	// Convert the map back to JSON
	newBodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	// Reset the request body
	r.Body = io.NopCloser(bytes.NewBuffer(newBodyBytes))
	r.ContentLength = int64(len(newBodyBytes))
}

func ShowBody(r *http.Request) {
	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	bodyBytes, err := io.ReadAll(tee)
	if err != nil {
		fmt.Println("Error reading body:", err)
	}

	var bodyMap map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			fmt.Println("Error parsing body JSON:", err)
			return
		}
	} else {
		bodyMap = make(map[string]interface{})
	}

	r.Body = io.NopCloser(&buf)
}

func ParseBody(r *http.Request, req interface{}) error {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("error reading body: %v", err)
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, req); err != nil {
			return fmt.Errorf("error parsing JSON: %v", err)
		}
	}

	return nil
}

func ParseRequest(r *http.Request, req interface{}) error {
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return err
	}
	return nil
}

func Pipe(w http.ResponseWriter, r *http.Request, bytes []byte, fileName, contentType string) {
	if strings.TrimSpace(fileName) == "" {
		fileName = "download-file"
	}

	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(bytes)))

	_, err := w.Write(bytes)
	if err != nil {
		log.Println("Error writing response:", err)
	}

}

func AutoWidthColumn(f *excelize.File, sheetName string) error {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("Read sheet error: %v", err)
	}

	// Variable stores maximum width of each column
	colWidths := make(map[int]int)

	// Browse row by row
	for _, row := range rows {
		for colIndex, cellValue := range row {
			cellValue = strings.TrimSpace(cellValue) // Remove extra whitespace
			length := len(cellValue)

			// Update the maximum width of the current column
			if length > colWidths[colIndex] {
				colWidths[colIndex] = length
			}
		}
	}

	// Update width for each column
	for colIndex, width := range colWidths {
		// Convert column numbers to column names
		colName, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return fmt.Errorf("Column conversion error: %v", err)
		}
		// Set column width (plus a little to avoid text clipping)
		f.SetColWidth(sheetName, colName, colName, float64(width)+4)
	}

	return nil
}

func GeneratePDF(conf *gopdf.Config, text string) ([]byte, error) {
	pdf := gopdf.GoPdf{}

	if conf == nil {
		defaultConf := gopdf.Config{PageSize: *gopdf.PageSizeA4}
		conf = &defaultConf
	}

	if text == "" {
		text = "Welcome"
	}

	pdf.Start(*conf)
	pdf.AddPage()

	pathFontArial, _ := GetPath("utils", "fonts", "Arial.ttf")
	//fmt.Println("Path font:", pathFontArial)

	// Load font
	err := pdf.AddTTFFont("arial", pathFontArial)
	if err != nil {
		log.Println("Error loading font:", err)
		return nil, err
	}

	err = pdf.SetFont("arial", "", 24)
	if err != nil {
		log.Println("Error setting font:", err)
		return nil, err
	}

	// Draw border
	pdf.SetStrokeColor(0, 0, 255)
	pdf.SetLineWidth(2)
	margin := 20.0
	width, height := gopdf.PageSizeA4.W, gopdf.PageSizeA4.H
	pdf.Line(margin, margin, width-margin, margin)               // Top
	pdf.Line(width-margin, margin, width-margin, height-margin)  // Right
	pdf.Line(margin, height-margin, width-margin, height-margin) // Bottom
	pdf.Line(margin, margin, margin, height-margin)              // Left

	// Add text
	textWidth, _ := pdf.MeasureTextWidth(text)
	textX := (width-textWidth)/2 + 10
	textX = float64(30)
	textY := height / 2
	pdf.SetTextColor(128, 128, 128)
	DrawMultilineText(&pdf, text, textX, textY, 28)

	// Export PDF
	var buf bytes.Buffer
	err = pdf.Write(&buf)
	if err != nil {
		log.Println("Error generating PDF:", err)
	}

	return buf.Bytes(), nil
}

func DrawMultilineText(pdf *gopdf.GoPdf, text string, startX, startY float64, lineHeight float64) {
	lines := strings.Split(text, "\n")
	x := startX
	y := startY

	for _, line := range lines {
		pdf.SetXY(x, y)
		pdf.Text(line)
		y += lineHeight
	}
}

func BuildPath(paths ...string) string {
	return filepath.Join(paths...)
}

func GetPath(parts ...string) (string, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Error path: %w", err)
	}
	fullPath := filepath.Join(append([]string{basePath}, parts...)...)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("File not exists: %s", fullPath)
	}
	return fullPath, nil
}

func MergePDFs(data ...[]byte) ([]byte, error) {
	//go get github.com/pdfcpu/pdfcpu@latest
	var tempFiles []string
	// Write temporary file to disk
	for i, d := range data {
		tmpFile := fmt.Sprintf("tmp_%d.pdf", i)
		if err := os.WriteFile(tmpFile, d, 0644); err != nil {
			return nil, fmt.Errorf("write temp file: %w", err)
		}
		tempFiles = append(tempFiles, tmpFile)
		defer os.Remove(tmpFile)
	}

	// Temporary output file for merging
	outputFile := "merged_tmp.pdf"
	defer os.Remove(outputFile)

	conf := model.NewDefaultConfiguration()
	err := api.MergeCreateFile(tempFiles, outputFile, false, conf)
	if err != nil {
		return nil, fmt.Errorf("merge error: %w", err)
	}

	mergedData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("read merged file: %w", err)
	}

	return mergedData, nil
}

func AddPageNumbers(input []byte) ([]byte, error) {
	conf := model.NewDefaultConfiguration()

	tmpInputFile, err := os.CreateTemp("", "input_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp input file: %v", err)
	}
	defer os.Remove(tmpInputFile.Name())
	if _, err := tmpInputFile.Write(input); err != nil {
		return nil, fmt.Errorf("failed to write input PDF to temp file: %v", err)
	}
	tmpInputFile.Close()

	tmpOutputFile, err := os.CreateTemp("", "output_*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp output file: %v", err)
	}
	defer os.Remove(tmpOutputFile.Name())
	tmpOutputFile.Close()

	// Create watermark with dynamic placeholder
	now := time.Now().Format("2006-01-02 15:04")
	_ = now
	text := fmt.Sprint("Trang %%p của %%P")
	desc := "font:Helvetica, points:10, pos:br, off:0 0, rot:0, scale:1 abs"

	wm, err := api.TextWatermark(text, desc, true, false, types.POINTS)
	if err != nil {
		return nil, fmt.Errorf("failed to create watermark: %v", err)
	}

	// Apply watermark to all pages
	err = api.AddWatermarksFile(tmpInputFile.Name(), tmpOutputFile.Name(), nil, wm, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to add watermark: %v", err)
	}

	outputBytes, err := os.ReadFile(tmpOutputFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %v", err)
	}

	return outputBytes, nil
}
