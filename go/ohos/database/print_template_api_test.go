package database

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPrintTemplateConfigSavesAndLoadsFromSysConfig(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "print-template-config.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	config, err := LoadPrintTemplateConfig()
	if err != nil {
		t.Fatalf("LoadPrintTemplateConfig default: %v", err)
	}
	if len(config.TopFields) != 11 || len(config.ContentFields) != 11 {
		t.Fatalf("default field counts = top %d content %d, want 11/11", len(config.TopFields), len(config.ContentFields))
	}

	config.TopFields[0].Checked = false
	config.ContentFields[2].Label = "Pieces"
	config.ContentFields[2].Width = 180
	if err := SavePrintTemplateConfig(config); err != nil {
		t.Fatalf("SavePrintTemplateConfig: %v", err)
	}

	loaded, err := LoadPrintTemplateConfig()
	if err != nil {
		t.Fatalf("LoadPrintTemplateConfig saved: %v", err)
	}
	if loaded.TopFields[0].Checked {
		t.Fatalf("CustomerName checked = true, want false")
	}
	if loaded.ContentFields[2].Label != "Pieces" || loaded.ContentFields[2].Width != 180 {
		t.Fatalf("saved content field = %#v, want label Pieces width 180", loaded.ContentFields[2])
	}

	var row struct {
		FValue string `gorm:"column:FValue"`
	}
	if err := db.Table((&TbSysConfigs{}).TableName()).
		Select("FValue").
		Where("FModuleName = ? AND FType = ?", "RSS", printTemplateConfigType).
		First(&row).Error; err != nil {
		t.Fatalf("read sys config row: %v", err)
	}
	if !strings.Contains(row.FValue, `"Pieces"`) {
		t.Fatalf("FValue = %q, want saved json", row.FValue)
	}
}

func TestPrintTemplatePreviewReportUsesSavedTemplate(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "print-template-preview.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	visible := 1
	batchWeight := 2000000.0
	batchNumber := uint64(10)
	weightGradeSum := 2
	qualityGradeSum := 0
	fruit := TbFruitInfo{
		CustomerID:           101,
		CustomerName:         "Alice",
		FarmName:             "Hidden Farm",
		FruitName:            "Orange",
		StartTime:            "2026-06-18 08:00:00",
		EndTime:              "2026-06-18 09:00:00",
		CompletedState:       "1",
		ProgramName:          "Program A",
		BatchWeight:          &batchWeight,
		BatchNumber:          &batchNumber,
		QualityGradeSum:      &qualityGradeSum,
		WeightOrSizeGradeSum: &weightGradeSum,
		FVisible:             &visible,
	}
	if err := db.Create(&fruit).Error; err != nil {
		t.Fatalf("seed fruit: %v", err)
	}

	box1 := 2.0
	count1 := uint64(6)
	weight1 := 1200000.0
	limit1 := 80.0
	price1 := 3.5
	box2 := 3.0
	count2 := uint64(4)
	weight2 := 800000.0
	limit2 := 70.0
	price2 := 2.5
	grades := []TbGradeInfo{
		{CustomerID: 101, GradeID: 0, WeightOrSizeName: "L", BoxNumber: &box1, FruitNumber: &count1, FruitWeight: &weight1, WeightOrSizeLimit: &limit1, FPrice: &price1},
		{CustomerID: 101, GradeID: 1, WeightOrSizeName: "M", BoxNumber: &box2, FruitNumber: &count2, FruitWeight: &weight2, WeightOrSizeLimit: &limit2, FPrice: &price2},
	}
	if err := db.Create(&grades).Error; err != nil {
		t.Fatalf("seed grades: %v", err)
	}

	config := defaultPrintTemplateConfig()
	for i := range config.TopFields {
		config.TopFields[i].Checked = config.TopFields[i].Key == "CustomerName"
	}
	for i := range config.ContentFields {
		config.ContentFields[i].Checked = config.ContentFields[i].Key == "WeightOrSizeName" || config.ContentFields[i].Key == "GradeCount"
		if config.ContentFields[i].Key == "GradeCount" {
			config.ContentFields[i].Label = "Pieces"
			config.ContentFields[i].Width = 180
		}
	}
	if err := SavePrintTemplateConfig(config); err != nil {
		t.Fatalf("SavePrintTemplateConfig: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)
	body := bytes.NewBufferString(`{"CustomerID":0}`)
	request := httptest.NewRequest(http.MethodPost, "/Api/PrintTemplate/PreviewReport", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var envelope fruitInfoAPIEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.ReturnCode != 1 {
		t.Fatalf("ReturnCode = %d, message=%q", envelope.ReturnCode, envelope.ReturnMessage)
	}
	var report printReportPreviewAPIModel
	if err := json.Unmarshal([]byte(envelope.Data), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.CustomerID != 101 {
		t.Fatalf("CustomerID = %d, want 101", report.CustomerID)
	}
	if !strings.Contains(report.HTML, "Customer") || !strings.Contains(report.HTML, "Alice") {
		t.Fatalf("HTML missing selected customer field: %s", report.HTML)
	}
	if !strings.Contains(report.HTML, `<img class="logo" src="report_logo.png" alt="Reemoon logo">`) {
		t.Fatalf("HTML missing Reemoon report logo: %s", report.HTML)
	}
	if strings.Contains(report.HTML, "logo-space") {
		t.Fatalf("HTML still uses empty logo placeholder: %s", report.HTML)
	}
	if strings.Contains(report.HTML, "Hidden Farm") {
		t.Fatalf("HTML contains unchecked farm field: %s", report.HTML)
	}
	if !strings.Contains(report.HTML, "<th>Pieces</th>") {
		t.Fatalf("HTML missing saved Pieces header: %s", report.HTML)
	}
	if strings.Contains(report.HTML, "Unit price") {
		t.Fatalf("HTML contains unchecked price header: %s", report.HTML)
	}
}
