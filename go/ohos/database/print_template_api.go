package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const printTemplateConfigType = "PrintTemplateConfig"

type printTemplateFieldAPIModel struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Checked bool    `json:"checked"`
	Width   float64 `json:"width,omitempty"`
}

type printTemplateConfigAPIModel struct {
	TopFields     []printTemplateFieldAPIModel `json:"topFields"`
	ContentFields []printTemplateFieldAPIModel `json:"contentFields"`
}

type printReportPreviewRequest struct {
	CustomerID int `json:"CustomerID"`
}

type printReportPreviewAPIModel struct {
	CustomerID int                         `json:"CustomerID"`
	FileName   string                      `json:"FileName"`
	Title      string                      `json:"Title"`
	HTML       string                      `json:"Html"`
	Template   printTemplateConfigAPIModel `json:"Template"`
}

type printReportTableRow struct {
	values map[string]string
}

func registerPrintTemplateRoutes(router *gin.Engine) {
	group := router.Group("/Api/PrintTemplate")
	group.POST("/GetPrintTemplate", handlePrintTemplateGet)
	group.POST("/SavePrintTemplate", handlePrintTemplateSave)
	group.POST("/PreviewReport", handlePrintTemplatePreviewReport)
}

func handlePrintTemplateGet(ctx *gin.Context) {
	config, err := LoadPrintTemplateConfig()
	if err != nil {
		printTemplateAPILog("GetPrintTemplate failed: %v", err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	printTemplateAPILog("GetPrintTemplate success: top=%d, content=%d", len(config.TopFields), len(config.ContentFields))
	fruitInfoAPIOK(ctx, config)
}

func handlePrintTemplateSave(ctx *gin.Context) {
	var config printTemplateConfigAPIModel
	if err := ctx.ShouldBindJSON(&config); err != nil {
		printTemplateAPILog("SavePrintTemplate failed: invalid request: %v", err)
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	normalized := normalizePrintTemplateConfig(config)
	if err := SavePrintTemplateConfig(normalized); err != nil {
		printTemplateAPILog("SavePrintTemplate failed: %v", err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	printTemplateAPILog("SavePrintTemplate success: top=%d, content=%d", len(normalized.TopFields), len(normalized.ContentFields))
	fruitInfoAPIOK(ctx, normalized)
}

func handlePrintTemplatePreviewReport(ctx *gin.Context) {
	var request printReportPreviewRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		printTemplateAPILog("PreviewReport failed: invalid request: %v", err)
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	result, err := BuildPrintReportPreview(request.CustomerID)
	if err != nil {
		printTemplateAPILog("PreviewReport failed: customerID=%d, error=%v", request.CustomerID, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	printTemplateAPILog("PreviewReport success: requestedCustomerID=%d, customerID=%d, html=%d bytes", request.CustomerID, result.CustomerID, len(result.HTML))
	fruitInfoAPIOK(ctx, result)
}

func LoadPrintTemplateConfig() (printTemplateConfigAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return printTemplateConfigAPIModel{}, err
	}

	value, err := getConfigValueWithDB(db, printTemplateConfigType)
	if err != nil {
		return printTemplateConfigAPIModel{}, err
	}
	if strings.TrimSpace(value) == "" {
		return defaultPrintTemplateConfig(), nil
	}

	var config printTemplateConfigAPIModel
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return printTemplateConfigAPIModel{}, fmt.Errorf("invalid print template config: %w", err)
	}
	return normalizePrintTemplateConfig(config), nil
}

func SavePrintTemplateConfig(config printTemplateConfigAPIModel) error {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}

	normalized := normalizePrintTemplateConfig(config)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return saveConfigValue(db, printTemplateConfigType, string(bytes))
}

func BuildPrintReportPreview(customerID int) (printReportPreviewAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return printReportPreviewAPIModel{}, err
	}

	var item fruitInfoAPIModel
	if customerID > 0 {
		var found bool
		item, found, err = getFruitInfoDetailByCustomerID(db, customerID)
		if err != nil {
			return printReportPreviewAPIModel{}, err
		}
		if !found {
			return printReportPreviewAPIModel{}, errors.New("report customer not found")
		}
	} else {
		var found bool
		item, found, err = getLatestCompletedFruitInfoDetail(db)
		if err != nil {
			return printReportPreviewAPIModel{}, err
		}
		if !found {
			return printReportPreviewAPIModel{}, errors.New("no completed report data")
		}
	}

	config, err := LoadPrintTemplateConfig()
	if err != nil {
		return printReportPreviewAPIModel{}, err
	}
	htmlText, err := buildPrintReportHTML(item, config)
	if err != nil {
		return printReportPreviewAPIModel{}, err
	}
	return printReportPreviewAPIModel{
		CustomerID: item.CustomerID,
		FileName:   fmt.Sprintf("加工报表_%d.html", item.CustomerID),
		Title:      "加工报表",
		HTML:       htmlText,
		Template:   config,
	}, nil
}

func defaultPrintTemplateConfig() printTemplateConfigAPIModel {
	return printTemplateConfigAPIModel{
		TopFields: []printTemplateFieldAPIModel{
			{Key: "CustomerName", Label: "Customer", Checked: true},
			{Key: "FarmName", Label: "Farm", Checked: true},
			{Key: "FruitName", Label: "Fruit", Checked: true},
			{Key: "BatchNumber", Label: "TotalPieces", Checked: true},
			{Key: "BatchWeight", Label: "TotalWeight", Checked: true},
			{Key: "BoxNumber", Label: "TotalBox", Checked: true},
			{Key: "AverWeight", Label: "AverFruitWeig", Checked: true},
			{Key: "ProgramName", Label: "SortProcedure", Checked: true},
			{Key: "CustomerID", Label: "SerialNum", Checked: true},
			{Key: "StartTime", Label: "StartTime", Checked: true},
			{Key: "EndTime", Label: "EndTime", Checked: true},
		},
		ContentFields: []printTemplateFieldAPIModel{
			{Key: "WeightOrSizeName", Label: "GradeName", Checked: true, Width: 120},
			{Key: "WeightOrSizeLimit", Label: "Weight/Size", Checked: true, Width: 120},
			{Key: "GradeCount", Label: "GradeTotalNum", Checked: true, Width: 120},
			{Key: "GradeCountPercent", Label: "Pieces Percentage", Checked: true, Width: 120},
			{Key: "WeightGradeCount", Label: "GradeTotalWeight", Checked: true, Width: 120},
			{Key: "WeightGradeCountPercent", Label: "Weight Percentage", Checked: true, Width: 120},
			{Key: "BoxGradeCount", Label: "TotalBox", Checked: true, Width: 120},
			{Key: "BoxGradeCountPercent", Label: "LblMainReportCartonsPer", Checked: true, Width: 120},
			{Key: "FPrice", Label: "Unit price(RMB/kg)", Checked: false, Width: 120},
			{Key: "Amount", Label: "Amount", Checked: false, Width: 120},
			{Key: "Notes", Label: "Notes", Checked: false, Width: 120},
		},
	}
}

func normalizePrintTemplateConfig(config printTemplateConfigAPIModel) printTemplateConfigAPIModel {
	defaults := defaultPrintTemplateConfig()
	return printTemplateConfigAPIModel{
		TopFields:     mergePrintTemplateFields(defaults.TopFields, config.TopFields, false),
		ContentFields: mergePrintTemplateFields(defaults.ContentFields, config.ContentFields, true),
	}
}

func mergePrintTemplateFields(defaults []printTemplateFieldAPIModel, input []printTemplateFieldAPIModel, withWidth bool) []printTemplateFieldAPIModel {
	byKey := make(map[string]printTemplateFieldAPIModel, len(input))
	for _, item := range input {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		item.Key = key
		item.Label = strings.TrimSpace(item.Label)
		byKey[key] = item
	}

	result := make([]printTemplateFieldAPIModel, 0, len(defaults))
	for _, item := range defaults {
		if override, ok := byKey[item.Key]; ok {
			item.Checked = override.Checked
			if strings.TrimSpace(override.Label) != "" {
				item.Label = strings.TrimSpace(override.Label)
			}
			if withWidth && override.Width > 0 {
				item.Width = math.Max(20, math.Min(600, override.Width))
			}
		}
		if withWidth && item.Width <= 0 {
			item.Width = 120
		}
		result = append(result, item)
	}
	return result
}

func getLatestCompletedFruitInfoDetail(db *gorm.DB) (fruitInfoAPIModel, bool, error) {
	var fruit TbFruitInfo
	err := db.Where("CompletedState = ?", "1").Order("CustomerID desc").First(&fruit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fruitInfoAPIModel{}, false, nil
	}
	if err != nil {
		return fruitInfoAPIModel{}, false, err
	}
	return getFruitInfoDetailByCustomerID(db, fruit.CustomerID)
}

func buildPrintReportHTML(item fruitInfoAPIModel, config printTemplateConfigAPIModel) (string, error) {
	topFields := checkedPrintTemplateFields(config.TopFields)
	contentFields := checkedPrintTemplateFields(config.ContentFields)
	if len(contentFields) == 0 {
		return "", errors.New("no print content fields selected")
	}

	summaryValues := buildPrintReportSummaryValues(item)
	rows := buildPrintReportRows(item, isPrintPriceEnabled(config.ContentFields))
	totalWidth := printTemplateTotalWidth(contentFields)

	var metaBuilder strings.Builder
	for _, field := range topFields {
		value := summaryValues[field.Key]
		metaBuilder.WriteString(`<div class="meta-item"><span>`)
		metaBuilder.WriteString(html.EscapeString(field.Label))
		metaBuilder.WriteString(`:</span><strong>`)
		metaBuilder.WriteString(html.EscapeString(emptyDash(value)))
		metaBuilder.WriteString(`</strong></div>`)
	}

	var colBuilder strings.Builder
	for _, field := range contentFields {
		percent := 100.0 * printTemplateFieldWidth(field) / totalWidth
		colBuilder.WriteString(fmt.Sprintf(`<col style="width: %.4f%%">`, percent))
	}

	var headBuilder strings.Builder
	headBuilder.WriteString("<tr>")
	for _, field := range contentFields {
		headBuilder.WriteString("<th>")
		headBuilder.WriteString(html.EscapeString(field.Label))
		headBuilder.WriteString("</th>")
	}
	headBuilder.WriteString("</tr>")

	var bodyBuilder strings.Builder
	if len(rows) == 0 {
		bodyBuilder.WriteString(fmt.Sprintf(`<tr><td colspan="%d" class="empty">No data</td></tr>`, len(contentFields)))
	} else {
		for _, row := range rows {
			bodyBuilder.WriteString("<tr>")
			for _, field := range contentFields {
				bodyBuilder.WriteString("<td>")
				bodyBuilder.WriteString(html.EscapeString(emptyDash(row.values[field.Key])))
				bodyBuilder.WriteString("</td>")
			}
			bodyBuilder.WriteString("</tr>")
		}
	}

	printTime := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <title>加工报表</title>
  <style>
    * { box-sizing: border-box; }
    body { margin: 0; min-height: 252.45mm; background: #d8d8d8; color: #000; font-family: SimSun, "宋体", "Microsoft YaHei", Arial, sans-serif; }
    main { position: relative; width: 210mm; min-height: 297mm; margin: 0 auto; padding: 13mm 10.5mm 16mm; background: #fff; transform: scale(0.85); transform-origin: top center; }
    .print-time { position: absolute; top: 7mm; right: 10.5mm; height: 22px; text-align: right; font-size: 15px; line-height: 22px; }
    .logo { display: block; width: 400px; height: 55px; object-fit: contain; margin: 8px auto 15px; }
    h1 { margin: 0 0 10px; font-size: 20px; font-weight: 700; text-align: center; letter-spacing: 0; }
    .double-line { height: 6px; border-top: 2px solid #000; border-bottom: 2px solid #000; margin: 0 0 10px; }
    .meta { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px 0; padding: 0 0 10px; font-size: 15px; line-height: 22px; }
    .meta-item { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding-right: 12px; }
    .meta-item span, .meta-item strong { font-weight: 400; }
    table { width: 100%%; border-collapse: collapse; table-layout: fixed; font-size: 12px; page-break-inside: auto; }
    col { min-width: 20px; }
    thead { display: table-header-group; }
    tr { page-break-inside: avoid; page-break-after: auto; }
    th, td { border: 1px solid #000; padding: 5px 6px; height: 24px; text-align: center; word-break: break-word; }
    th { font-weight: 700; background: #fff; }
    .empty { text-align: center !important; }
    .page-footer { position: absolute; left: 10.5mm; right: 10.5mm; bottom: 8mm; text-align: center; font-size: 11px; }
    @page { size: A4; margin: 0; }
    @media print {
      body { background: #fff; }
      main { width: auto; min-height: 297mm; margin: 0; padding: 13mm 10.5mm 16mm; transform: none; }
      .page-footer { position: fixed; }
    }
  </style>
</head>
<body>
  <main>
    <div class="print-time">%s</div>
    <img class="logo" src="report_logo.png" alt="Reemoon logo">
    <h1>加工报表</h1>
    <div class="double-line"></div>
    <section class="meta">%s</section>
    <div class="double-line"></div>
    <table>
      <colgroup>%s</colgroup>
      <thead>%s</thead>
      <tbody>%s</tbody>
    </table>
    <div class="page-footer">Page 1/1</div>
  </main>
</body>
</html>`, html.EscapeString(printTime), metaBuilder.String(), colBuilder.String(), headBuilder.String(), bodyBuilder.String()), nil
}

func checkedPrintTemplateFields(fields []printTemplateFieldAPIModel) []printTemplateFieldAPIModel {
	result := make([]printTemplateFieldAPIModel, 0, len(fields))
	for _, field := range fields {
		if field.Checked {
			result = append(result, field)
		}
	}
	return result
}

func isPrintPriceEnabled(fields []printTemplateFieldAPIModel) bool {
	for _, field := range fields {
		if field.Key == "FPrice" {
			return field.Checked
		}
	}
	return false
}

func buildPrintReportSummaryValues(item fruitInfoAPIModel) map[string]string {
	totalCount := printReportTotalCount(item)
	totalWeight := printReportTotalWeight(item)
	totalBox := printReportTotalBox(item)

	return map[string]string{
		"CustomerName": item.CustomerName,
		"FarmName":     item.FarmName,
		"FruitName":    item.FruitName,
		"BatchNumber":  fmt.Sprintf("%d", totalCount),
		"BatchWeight":  formatFloat(totalWeight/1000000, 3),
		"BoxNumber":    formatFloat(totalBox, 0),
		"AverWeight":   formatFloat(printReportAverageWeight(totalWeight, totalCount), 3),
		"ProgramName":  firstNonEmpty(item.ProgramName, item.SortBaseName),
		"CustomerID":   fmt.Sprintf("%d", item.CustomerID),
		"StartTime":    item.StartTime,
		"EndTime":      item.EndTime,
	}
}

func buildPrintReportRows(item fruitInfoAPIModel, priceEnabled bool) []printReportTableRow {
	totalCount := printReportTotalCount(item)
	totalWeight := printReportTotalWeight(item)
	totalBox := printReportTotalBox(item)
	rows := make([]printReportTableRow, 0, len(item.ListTbGradeInfo)+1)
	totalAmount := 0.0

	for _, grade := range item.ListTbGradeInfo {
		count := float64(grade.FruitNumber)
		weight := grade.FruitWeight
		box := grade.BoxNumber
		amount := grade.FPrice * weight / 1000.0
		totalAmount += amount
		rows = append(rows, printReportTableRow{values: map[string]string{
			"WeightOrSizeName":        printReportGradeName(grade),
			"WeightOrSizeLimit":       formatFloat(grade.WeightOrSizeLimit, 0),
			"GradeCount":              fmt.Sprintf("%d", grade.FruitNumber),
			"GradeCountPercent":       formatPercent(count, float64(totalCount), 2),
			"WeightGradeCount":        formatFloat(weight/1000.0, 3),
			"WeightGradeCountPercent": formatPercent(weight, totalWeight, 3),
			"BoxGradeCount":           formatFloat(box, 0),
			"BoxGradeCountPercent":    formatPercent(box, totalBox, 3),
			"FPrice":                  formatFloat(grade.FPrice, 2),
			"Amount":                  formatFloat(amount, 2),
			"Notes":                   "",
		}})
	}

	if len(rows) > 0 {
		totalRow := map[string]string{
			"WeightOrSizeName":        "Total",
			"GradeCount":              fmt.Sprintf("%d", totalCount),
			"GradeCountPercent":       formatPercent(float64(totalCount), float64(totalCount), 3),
			"WeightGradeCount":        formatFloat(totalWeight/1000.0, 3),
			"WeightGradeCountPercent": formatPercent(totalWeight, totalWeight, 3),
			"BoxGradeCount":           formatFloat(totalBox, 0),
			"BoxGradeCountPercent":    formatPercent(totalBox, totalBox, 3),
			"Amount":                  formatFloat(totalAmount, 2),
		}
		rows = append(rows, printReportTableRow{values: totalRow})
		if priceEnabled && totalWeight > 0 {
			rows = append(rows, printReportTableRow{values: map[string]string{
				"WeightOrSizeName": "AverAmount",
				"Amount":           formatFloat(totalAmount*1000.0/totalWeight, 2),
			}})
		}
	}
	return rows
}

func printReportGradeName(grade gradeInfoAPIModel) string {
	sizeName := strings.TrimSpace(grade.WeightOrSizeName)
	qualityName := strings.TrimSpace(grade.QualityName)
	if sizeName != "" && qualityName != "" {
		return sizeName + "." + qualityName
	}
	return firstNonEmpty(sizeName, qualityName)
}

func printReportTotalCount(item fruitInfoAPIModel) uint64 {
	if item.BatchNumber > 0 {
		return item.BatchNumber
	}
	var total uint64
	for _, grade := range item.ListTbGradeInfo {
		total += grade.FruitNumber
	}
	return total
}

func printReportTotalWeight(item fruitInfoAPIModel) float64 {
	if item.BatchWeight > 0 {
		return item.BatchWeight
	}
	var total float64
	for _, grade := range item.ListTbGradeInfo {
		total += grade.FruitWeight
	}
	return total
}

func printReportTotalBox(item fruitInfoAPIModel) float64 {
	var total float64
	for _, grade := range item.ListTbGradeInfo {
		total += grade.BoxNumber
	}
	return total
}

func printReportAverageWeight(totalWeight float64, totalCount uint64) float64 {
	if totalCount == 0 {
		return 0
	}
	return totalWeight / float64(totalCount)
}

func printTemplateTotalWidth(fields []printTemplateFieldAPIModel) float64 {
	total := 0.0
	for _, field := range fields {
		total += printTemplateFieldWidth(field)
	}
	if total <= 0 {
		return 1
	}
	return total
}

func printTemplateFieldWidth(field printTemplateFieldAPIModel) float64 {
	if field.Width <= 0 {
		return 120
	}
	return math.Max(20, field.Width)
}

func formatPercent(value float64, total float64, digits int) string {
	if total <= 0 {
		return formatFloat(0, digits) + "%"
	}
	return formatFloat(value*100.0/total, digits) + "%"
}

func formatFloat(value float64, digits int) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		value = 0
	}
	if digits <= 0 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.*f", digits, value)
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func printTemplateAPILog(format string, args ...any) {
}
