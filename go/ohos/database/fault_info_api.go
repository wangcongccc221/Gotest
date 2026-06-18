package database

import (
	"errors"
	"fmt"
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
		faultInfoAPILog("GetListFaultInfoByDetailCondition failed: invalid request: %v", err)
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	faultInfoAPILog("GetListFaultInfoByDetailCondition begin: status=%d, start=%q, end=%q, keyword=%q",
		request.FStatus, request.FBeginDate, request.FEndDate, request.FMessage)
	rows, err := QueryFaultInfos(request)
	if err != nil {
		faultInfoAPILog("GetListFaultInfoByDetailCondition failed: status=%d, start=%q, end=%q, keyword=%q, error=%v",
			request.FStatus, request.FBeginDate, request.FEndDate, request.FMessage, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	faultInfoAPILog("GetListFaultInfoByDetailCondition success: rows=%d", len(rows))
	fruitInfoAPIOK(ctx, rows)
}

func handleFaultInfoGetList(ctx *gin.Context) {
	var request faultInfoAPIModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		faultInfoAPILog("GetListFaultInfo failed: invalid request: %v", err)
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	faultInfoAPILog("GetListFaultInfo begin: type=%d, status=%d", request.FType, request.FStatus)
	rows, err := QueryUnfinishedFaultInfos(request.FType)
	if err != nil {
		faultInfoAPILog("GetListFaultInfo failed: type=%d, status=%d, error=%v", request.FType, request.FStatus, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	faultInfoAPILog("GetListFaultInfo success: type=%d, status=%d, rows=%d", request.FType, request.FStatus, len(rows))
	fruitInfoAPIOK(ctx, rows)
}

func handleFaultInfoSaveList(ctx *gin.Context) {
	var rows []faultInfoAPIModel
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		faultInfoAPILog("SaveFaultInfos failed: invalid request: %v", err)
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	faultType := faultInfoRowsType(rows)
	faultInfoAPILog("SaveFaultInfos begin: type=%d, rows=%d", faultType, len(rows))
	if err := SaveFaultInfos(rows); err != nil {
		faultInfoAPILog("SaveFaultInfos failed: type=%d, rows=%d, error=%v", faultType, len(rows), err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	faultInfoAPILog("SaveFaultInfos success: type=%d, rows=%d", faultType, len(rows))
	fruitInfoAPIOKRaw(ctx, "")
}

func faultInfoRowsType(rows []faultInfoAPIModel) int {
	if len(rows) == 0 {
		return -1
	}
	return rows[0].FType
}

func faultInfoAPILog(format string, args ...any) {
	fmt.Printf("%s Web API faultInfo %s\n", databaseNow().Format("15:04:05.000"), fmt.Sprintf(format, args...))
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
