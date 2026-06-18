package database

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type baseFaultAPIRequest struct {
	FID   int `json:"FID"`
	FType int `json:"FType"`
}

type baseFaultAPIModel struct {
	FID                    int    `json:"FID"`
	FType                  int    `json:"FType"`
	FCode                  string `json:"FCode"`
	FName                  string `json:"FName"`
	FGroupName             string `json:"FGroupName"`
	FMessage               string `json:"FMessage"`
	FShutdownCondition     int    `json:"FShutdownCondition"`
	FDecelerationCondition int    `json:"FDecelerationCondition"`
	FPopupCondition        int    `json:"FPopupCondition"`
	FPromptCondition       int    `json:"FPromptCondition"`
	FExitID                int    `json:"FExitID"`
	FDuration              int    `json:"FDuration"`
	FStatus                int    `json:"FStatus"`
	FOccurDate             string `json:"FOccurDate"`
}

func registerFaultRoutes(router *gin.Engine) {
	group := router.Group("/Api/Fault")
	group.POST("/GetListBaseFaultInfo", handleBaseFaultGetList)
	group.POST("/SaveBaseFaults", handleBaseFaultSaveList)
	group.POST("/DeleteBaseFaultInfo", handleBaseFaultDelete)
	group.POST("/GetListFaultInfoByDetailCondition", handleFaultInfoGetByDetailCondition)
	group.POST("/GetListFaultInfo", handleFaultInfoGetList)
	group.POST("/SaveFaultInfos", handleFaultInfoSaveList)
}

func handleBaseFaultGetList(ctx *gin.Context) {
	var request baseFaultAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	rows, err := QueryBaseFaults(request.FType)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, rows)
}

func handleBaseFaultSaveList(ctx *gin.Context) {
	var rows []baseFaultAPIModel
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	if err := SaveBaseFaults(rows); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func handleBaseFaultDelete(ctx *gin.Context) {
	var request baseFaultAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	if request.FID <= 0 {
		fruitInfoAPIFail(ctx, "FID is required")
		return
	}

	if err := DeleteBaseFault(request.FID); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func QueryBaseFaults(faultType int) ([]baseFaultAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return nil, err
	}

	query := db.Model(&TbBaseFault{}).
		Select(
			"FID",
			"FType",
			"FCode",
			"FName",
			"FGroupName",
			"FMessage",
			"FShutdownCondition",
			"FDecelerationCondition",
			"FPopupCondition",
			"FPromptCondition",
			"FExitID",
			"FDuration",
		)
	if faultType >= 0 {
		query = query.Where("FType = ?", faultType)
	}
	var rows []TbBaseFault
	if err := query.Order("FID asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]baseFaultAPIModel, 0, len(rows))
	for _, row := range rows {
		result = append(result, baseFaultToAPIModel(row))
	}
	return result, nil
}

func SaveBaseFaults(rows []baseFaultAPIModel) error {
	if len(rows) == 0 {
		return nil
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		idsByType := make(map[int][]int)
		for _, row := range rows {
			if err := saveBaseFaultRow(tx, row); err != nil {
				return err
			}
			if row.FID > 0 {
				idsByType[row.FType] = append(idsByType[row.FType], row.FID)
			}
		}
		for _, row := range rows {
			if row.FID <= 0 {
				var saved TbBaseFault
				err := tx.Select("FID").
					Where("FType = ? AND FCode = ? AND FName = ?", row.FType, strings.TrimSpace(row.FCode), strings.TrimSpace(row.FName)).
					Order("FID desc").
					First(&saved).Error
				if err != nil {
					return err
				}
				idsByType[row.FType] = append(idsByType[row.FType], saved.FID)
			}
		}
		for faultType, ids := range idsByType {
			if len(ids) == 0 {
				continue
			}
			if err := tx.Where("FType = ? AND FID NOT IN ?", faultType, ids).Delete(&TbBaseFault{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteBaseFault(fid int) error {
	if fid <= 0 {
		return errors.New("FID is required")
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}
	result := db.Delete(&TbBaseFault{}, fid)
	return result.Error
}

func saveBaseFaultRow(tx *gorm.DB, row baseFaultAPIModel) error {
	row.FCode = strings.TrimSpace(row.FCode)
	row.FName = strings.TrimSpace(row.FName)
	row.FGroupName = strings.TrimSpace(row.FGroupName)
	row.FMessage = strings.TrimSpace(row.FMessage)
	if row.FCode == "" {
		return errors.New("FCode is required")
	}
	if row.FName == "" {
		return errors.New("FName is required")
	}
	if row.FType == 1 && row.FGroupName == "" {
		row.FGroupName = row.FName
	}
	if row.FGroupName == "" {
		return errors.New("FGroupName is required")
	}

	now := databaseNow()
	faultType := row.FType
	shutdownCondition := row.FShutdownCondition
	decelerationCondition := row.FDecelerationCondition
	popupCondition := row.FPopupCondition
	promptCondition := row.FPromptCondition
	exitID := row.FExitID
	duration := row.FDuration
	updates := map[string]any{
		"FType":                  &faultType,
		"FCode":                  row.FCode,
		"FName":                  row.FName,
		"FGroupName":             row.FGroupName,
		"FMessage":               row.FMessage,
		"FShutdownCondition":     &shutdownCondition,
		"FDecelerationCondition": &decelerationCondition,
		"FPopupCondition":        &popupCondition,
		"FPromptCondition":       &promptCondition,
		"FExitID":                &exitID,
		"FDuration":              &duration,
		"FModifyDate":            &now,
	}

	if row.FID > 0 {
		result := tx.Model(&TbBaseFault{}).Where("FID = ?", row.FID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return nil
		}
	}

	var existing struct {
		FID int `gorm:"column:FID"`
	}
	result := tx.Table((&TbBaseFault{}).TableName()).
		Select("FID").
		Where("FType = ? AND FCode = ? AND FName = ?", row.FType, row.FCode, row.FName).
		Find(&existing)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return tx.Model(&TbBaseFault{}).Where("FID = ?", existing.FID).Updates(updates).Error
	}

	item := TbBaseFault{
		FCreateDate:            &now,
		FModifyDate:            &now,
		FType:                  &faultType,
		FCode:                  row.FCode,
		FName:                  row.FName,
		FGroupName:             row.FGroupName,
		FMessage:               row.FMessage,
		FShutdownCondition:     &shutdownCondition,
		FDecelerationCondition: &decelerationCondition,
		FPopupCondition:        &popupCondition,
		FPromptCondition:       &promptCondition,
		FExitID:                &exitID,
		FDuration:              &duration,
	}
	return tx.Create(&item).Error
}

func baseFaultToAPIModel(row TbBaseFault) baseFaultAPIModel {
	return baseFaultAPIModel{
		FID:                    row.FID,
		FType:                  intPtrValue(row.FType),
		FCode:                  row.FCode,
		FName:                  row.FName,
		FGroupName:             row.FGroupName,
		FMessage:               row.FMessage,
		FShutdownCondition:     intPtrValue(row.FShutdownCondition),
		FDecelerationCondition: intPtrValue(row.FDecelerationCondition),
		FPopupCondition:        intPtrValue(row.FPopupCondition),
		FPromptCondition:       intPtrValue(row.FPromptCondition),
		FExitID:                intPtrValue(row.FExitID),
		FDuration:              intPtrValue(row.FDuration),
		FStatus:                row.FStatus,
		FOccurDate:             formatFaultDateTime(row.FOccurDate),
	}
}

func formatFaultDateTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return databaseLocalTime(*value).Format("2006-01-02 15:04:05")
}
