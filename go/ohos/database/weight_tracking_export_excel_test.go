package database

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func TestBuildWeightTrackingExportExcelMatches48AndKeepsLast1000Rows(t *testing.T) {
	rows := make([]weightTrackingExportRow, 0, 1002)
	for index := 1; index <= 1002; index++ {
		rows = append(rows, weightTrackingExportRow{
			VehicleID:     fmt.Sprintf("%d", index),
			FruitWeight:   fmt.Sprintf("%d.10", index),
			VehicleWeight: fmt.Sprintf("%d.20", index),
			AD0:           fmt.Sprintf("%d", index+1000),
			AD1:           fmt.Sprintf("%d", index+2000),
		})
	}

	fixedTime := time.Date(2026, time.July, 17, 16, 20, 30, 0, time.Local)
	logo := decodeWeightTrackingExportTestLogo(t)
	result, err := buildWeightTrackingExportExcel(rows, logo, fixedTime)
	if err != nil {
		t.Fatalf("buildWeightTrackingExportExcel: %v", err)
	}
	if result.FileName != "20260717162030.xlsx" {
		t.Fatalf("FileName = %q, want %q", result.FileName, "20260717162030.xlsx")
	}
	if result.RecordCount != 1000 {
		t.Fatalf("RecordCount = %d, want 1000", result.RecordCount)
	}

	book, err := excelize.OpenReader(bytes.NewReader(result.Content))
	if err != nil {
		t.Fatalf("open generated workbook: %v", err)
	}
	t.Cleanup(func() { _ = book.Close() })
	if sheets := book.GetSheetList(); len(sheets) != 1 || sheets[0] != "Sheet1" {
		t.Fatalf("sheets = %v, want [Sheet1]", sheets)
	}

	pictures, err := book.GetPictures(weightTrackingExportSheetName, "B1")
	if err != nil {
		t.Fatalf("GetPictures(B1): %v", err)
	}
	if len(pictures) != 1 || !bytes.Equal(pictures[0].File, logo) {
		t.Fatalf("pictures at B1 = %d, want the exported logo", len(pictures))
	}
	assertWeightTrackingExportCell(t, book, "A3", "车号")
	assertWeightTrackingExportCell(t, book, "B3", "果重G")
	assertWeightTrackingExportCell(t, book, "C3", "车重G")
	assertWeightTrackingExportCell(t, book, "D3", "AD0")
	assertWeightTrackingExportCell(t, book, "E3", "AD1")
	assertWeightTrackingExportCell(t, book, "A4", "3")
	assertWeightTrackingExportCell(t, book, "B4", "3.10")
	assertWeightTrackingExportCell(t, book, "A1003", "1002")
}

func decodeWeightTrackingExportTestLogo(t *testing.T) []byte {
	t.Helper()
	logoImage := image.NewRGBA(image.Rect(0, 0, 2, 1))
	logoImage.Set(0, 0, color.RGBA{R: 0, G: 146, B: 137, A: 255})
	logoImage.Set(1, 0, color.RGBA{R: 227, G: 30, B: 36, A: 255})
	var content bytes.Buffer
	if err := png.Encode(&content, logoImage); err != nil {
		t.Fatalf("encode test logo: %v", err)
	}
	return content.Bytes()
}

func assertWeightTrackingExportCell(t *testing.T, book *excelize.File, cell string, want string) {
	t.Helper()
	got, err := book.GetCellValue(weightTrackingExportSheetName, cell)
	if err != nil {
		t.Fatalf("GetCellValue(%s): %v", cell, err)
	}
	if got != want {
		t.Fatalf("cell %s = %q, want %q", cell, got, want)
	}
}

func TestWeightTrackingExportExcelEndpointReturnsWorkbook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerPrintTemplateRoutes(router)
	requestBody, err := json.Marshal(map[string]interface{}{
		"LogoBase64": base64.StdEncoding.EncodeToString(decodeWeightTrackingExportTestLogo(t)),
		"Rows": []map[string]string{{
			"VehicleID": "88", "FruitWeight": "123.45", "VehicleWeight": "45.67", "AD0": "100", "AD1": "200",
		}},
	})
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/Api/PrintTemplate/ExportWeightTrackingExcel", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var envelope fruitInfoAPIEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.ReturnCode != 1 {
		t.Fatalf("ReturnCode = %d, message = %q", envelope.ReturnCode, envelope.ReturnMessage)
	}
	var payload struct {
		FileName      string `json:"FileName"`
		ContentBase64 string `json:"ContentBase64"`
		RecordCount   int    `json:"RecordCount"`
	}
	if err := json.Unmarshal([]byte(envelope.Data), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.RecordCount != 1 {
		t.Fatalf("RecordCount = %d, want 1", payload.RecordCount)
	}
	content, err := base64.StdEncoding.DecodeString(payload.ContentBase64)
	if err != nil {
		t.Fatalf("decode workbook: %v", err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	t.Cleanup(func() { _ = book.Close() })
	assertWeightTrackingExportCell(t, book, "A4", "88")
}
