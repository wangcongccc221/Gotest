package database

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type faultInfoAPIModel struct {
	FID        int    `json:"FID"`
	FType      int    `json:"FType"`
	FCode      string `json:"FCode"`
	FName      string `json:"FName"`
	FMessage   string `json:"FMessage"`
	FPartIdx   int    `json:"FPartIdx"`
	FStatus    int    `json:"FStatus"`
	FBeginDate string `json:"FBeginDate"`
	FEndDate   string `json:"FEndDate"`
}

type faultInfoDBRow struct {
	FID        int    `gorm:"column:FID"`
	FType      int    `gorm:"column:FType"`
	FCode      string `gorm:"column:FCode"`
	FName      string `gorm:"column:FName"`
	FMessage   string `gorm:"column:FMessage"`
	FPartIdx   int    `gorm:"column:FPartIdx"`
	FStatus    int    `gorm:"column:FStatus"`
	FBeginDate string `gorm:"column:FBeginDate"`
	FEndDate   string `gorm:"column:FEndDate"`
}

func handleFaultInfoGetByDetailCondition(ctx *gin.Context) {
	var request faultInfoAPIModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	rows, err := QueryFaultInfos(request)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, rows)
}

func handleFaultInfoGetList(ctx *gin.Context) {
	var request faultInfoAPIModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	rows, err := QueryUnfinishedFaultInfos(request.FType)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, rows)
}

func handleFaultInfoSaveList(ctx *gin.Context) {
	var rows []faultInfoAPIModel
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	if err := SaveFaultInfos(rows); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func QueryFaultInfos(request faultInfoAPIModel) ([]faultInfoAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return nil, err
	}

	query := db.Model(&TbFaultInfo{}).Select(
		"FID",
		"FType",
		"FCode",
		"FName",
		"FMessage",
		"FPartIdx",
		"FStatus",
		"FBeginDate",
		"FEndDate",
	)
	if request.FStatus >= 0 {
		query = query.Where("FStatus = ?", request.FStatus)
	}
	if begin, ok := parseFaultInfoDateTime(request.FBeginDate); ok {
		query = query.Where("FBeginDate >= ?", begin)
	}
	if end, ok := parseFaultInfoDateTime(request.FEndDate); ok {
		query = query.Where("FBeginDate <= ?", end)
	}
	if keyword := strings.TrimSpace(request.FMessage); keyword != "" {
		query = query.Where("FMessage LIKE ?", "%"+keyword+"%")
	}

	var rows []faultInfoDBRow
	if err := query.Order("FBeginDate desc, FID desc").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return faultInfoRowsToAPI(rows), nil
}

func QueryUnfinishedFaultInfos(faultType int) ([]faultInfoAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return nil, err
	}

	var rows []faultInfoDBRow
	err = db.Model(&TbFaultInfo{}).
		Select("FID", "FType", "FCode", "FName", "FMessage", "FPartIdx", "FStatus", "FBeginDate", "FEndDate").
		Where("FType = ? AND FStatus = ?", faultType, 0).
		Order("FBeginDate desc, FID desc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return faultInfoRowsToAPI(rows), nil
}

func SaveFaultInfos(rows []faultInfoAPIModel) error {
	if len(rows) == 0 {
		return nil
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := saveFaultInfoRow(tx, row); err != nil {
				return err
			}
		}
		return nil
	})
}

func saveFaultInfoRow(tx *gorm.DB, row faultInfoAPIModel) error {
	row.FCode = strings.TrimSpace(row.FCode)
	row.FName = strings.TrimSpace(row.FName)
	row.FMessage = strings.TrimSpace(row.FMessage)
	if row.FCode == "" {
		return errors.New("FCode is required")
	}
	if row.FName == "" {
		return errors.New("FName is required")
	}
	if row.FMessage == "" {
		return errors.New("FMessage is required")
	}

	faultType := row.FType
	partIdx := row.FPartIdx
	status := row.FStatus
	beginDate, hasBegin := parseFaultInfoDateTime(row.FBeginDate)
	endDate, hasEnd := parseFaultInfoDateTime(row.FEndDate)
	if !hasBegin {
		beginDate = databaseNow()
	}

	updates := map[string]any{
		"FType":    &faultType,
		"FCode":    row.FCode,
		"FName":    row.FName,
		"FMessage": row.FMessage,
		"FPartIdx": &partIdx,
		"FStatus":  &status,
	}
	if hasBegin {
		updates["FBeginDate"] = &beginDate
	}
	if hasEnd {
		updates["FEndDate"] = &endDate
	} else if status == 0 {
		updates["FEndDate"] = nil
	}

	if row.FID > 0 {
		result := tx.Model(&TbFaultInfo{}).Where("FID = ?", row.FID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	item := TbFaultInfo{
		FType:      &faultType,
		FCode:      row.FCode,
		FName:      row.FName,
		FMessage:   row.FMessage,
		FPartIdx:   &partIdx,
		FStatus:    &status,
		FBeginDate: &beginDate,
	}
	if hasEnd {
		item.FEndDate = &endDate
	}
	return tx.Create(&item).Error
}

func faultInfoRowsToAPI(rows []faultInfoDBRow) []faultInfoAPIModel {
	result := make([]faultInfoAPIModel, 0, len(rows))
	for _, row := range rows {
		result = append(result, faultInfoToAPIModel(row))
	}
	return result
}

func faultInfoToAPIModel(row faultInfoDBRow) faultInfoAPIModel {
	return faultInfoAPIModel{
		FID:        row.FID,
		FType:      row.FType,
		FCode:      row.FCode,
		FName:      row.FName,
		FMessage:   row.FMessage,
		FPartIdx:   row.FPartIdx,
		FStatus:    row.FStatus,
		FBeginDate: normalizeFaultInfoDateTimeText(row.FBeginDate),
		FEndDate:   normalizeFaultInfoDateTimeText(row.FEndDate),
	}
}

func normalizeFaultInfoDateTimeText(value string) string {
	if parsed, ok := parseFaultInfoDateTime(value); ok {
		return databaseLocalTime(parsed).Format("2006-01-02 15:04:05")
	}
	return strings.TrimSpace(value)
}

func parseFaultInfoDateTime(value string) (time.Time, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, text, databaseChinaLocation)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
