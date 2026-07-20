package database

import (
	"encoding/base64"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	weightTrackingExportSheetName = "Sheet1"
	weightTrackingExportMaxRows   = 1000
)

type weightTrackingExportRow struct {
	VehicleID     string `json:"VehicleID"`
	FruitWeight   string `json:"FruitWeight"`
	VehicleWeight string `json:"VehicleWeight"`
	AD0           string `json:"AD0"`
	AD1           string `json:"AD1"`
}

type weightTrackingExportExcelResult struct {
	FileName    string
	Content     []byte
	RecordCount int
}

type weightTrackingExportExcelRequest struct {
	LogoBase64 string                    `json:"LogoBase64"`
	Rows       []weightTrackingExportRow `json:"Rows"`
}

type weightTrackingExportExcelAPIModel struct {
	FileName      string `json:"FileName"`
	ContentBase64 string `json:"ContentBase64"`
	RecordCount   int    `json:"RecordCount"`
}

func handleWeightTrackingExportExcel(ctx *gin.Context) {
	var request weightTrackingExportExcelRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	logo, err := base64.StdEncoding.DecodeString(request.LogoBase64)
	if err != nil || len(logo) == 0 {
		fruitInfoAPIFail(ctx, "invalid report logo")
		return
	}
	result, err := buildWeightTrackingExportExcel(request.Rows, logo, time.Now())
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, weightTrackingExportExcelAPIModel{
		FileName:      result.FileName,
		ContentBase64: base64.StdEncoding.EncodeToString(result.Content),
		RecordCount:   result.RecordCount,
	})
}

func buildWeightTrackingExportExcel(rows []weightTrackingExportRow, logo []byte, now time.Time) (weightTrackingExportExcelResult, error) {
	if len(rows) == 0 {
		return weightTrackingExportExcelResult{}, errors.New("no weight tracking data")
	}
	if len(logo) == 0 {
		return weightTrackingExportExcelResult{}, errors.New("no report logo")
	}
	if len(rows) > weightTrackingExportMaxRows {
		rows = rows[len(rows)-weightTrackingExportMaxRows:]
	}

	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	if err := book.SetSheetName(book.GetSheetName(0), weightTrackingExportSheetName); err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	for _, column := range []string{"A", "B", "C", "D", "E"} {
		if err := book.SetColWidth(weightTrackingExportSheetName, column, column, 24); err != nil {
			return weightTrackingExportExcelResult{}, err
		}
	}
	if err := book.MergeCell(weightTrackingExportSheetName, "A1", "E1"); err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	if err := book.SetRowHeight(weightTrackingExportSheetName, 1, 50); err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	if err := book.SetRowHeight(weightTrackingExportSheetName, 2, 10); err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	if err := book.AddPictureFromBytes(weightTrackingExportSheetName, "B1", &excelize.Picture{
		Extension: ".png",
		File:      logo,
		Format: &excelize.GraphicOptions{
			AltText:         "Reemoon logo",
			LockAspectRatio: true,
			ScaleX:          0.5,
			ScaleY:          0.5,
		},
	}); err != nil {
		return weightTrackingExportExcelResult{}, err
	}

	headerStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "SimSun", Size: 12, Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	dataStyle, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "SimSun", Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return weightTrackingExportExcelResult{}, err
	}

	for index, header := range []string{"车号", "果重G", "车重G", "AD0", "AD1"} {
		cell, err := excelize.CoordinatesToCellName(index+1, 3)
		if err != nil {
			return weightTrackingExportExcelResult{}, err
		}
		if err := setWeightTrackingExportCell(book, cell, header, headerStyle); err != nil {
			return weightTrackingExportExcelResult{}, err
		}
	}
	for rowIndex, row := range rows {
		values := []string{row.VehicleID, row.FruitWeight, row.VehicleWeight, row.AD0, row.AD1}
		for columnIndex, value := range values {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+4)
			if err != nil {
				return weightTrackingExportExcelResult{}, err
			}
			if err := setWeightTrackingExportCell(book, cell, value, dataStyle); err != nil {
				return weightTrackingExportExcelResult{}, err
			}
		}
	}

	content, err := book.WriteToBuffer()
	if err != nil {
		return weightTrackingExportExcelResult{}, err
	}
	return weightTrackingExportExcelResult{
		FileName:    now.Format("20060102150405") + ".xlsx",
		Content:     content.Bytes(),
		RecordCount: len(rows),
	}, nil
}

func setWeightTrackingExportCell(book *excelize.File, cell string, value string, styleID int) error {
	if err := book.SetCellStr(weightTrackingExportSheetName, cell, value); err != nil {
		return err
	}
	return book.SetCellStyle(weightTrackingExportSheetName, cell, cell, styleID)
}
