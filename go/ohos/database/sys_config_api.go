package database

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type sysConfigAPIRequest struct {
	FType       string `json:"FType"`
	FVisible    *int   `json:"FVisible"`
	FModuleName string `json:"FModuleName"`
}

type sysConfigAPIModel struct {
	FID              int    `json:"FID"`
	FType            string `json:"FType"`
	FValue           string `json:"FValue"`
	FModuleName      string `json:"FModuleName"`
	FVisible         int    `json:"FVisible"`
	FEnType          string `json:"FEnType"`
	FValueType       int    `json:"FValueType"`
	FValueTypeDetail string `json:"FValueTypeDetail"`
	FZhType          string `json:"FZhType"`
}

func registerSysConfigRoutes(router *gin.Engine) {
	group := router.Group("/Api/SysConfig")
	group.POST("/GetListSysConfigs", handleSysConfigGetList)
	group.POST("/SaveSysConfigs", handleSysConfigSaveList)
}

func handleSysConfigGetList(ctx *gin.Context) {
	var request sysConfigAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	rows, err := QuerySysConfigs(request)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, rows)
}

func handleSysConfigSaveList(ctx *gin.Context) {
	var rows []sysConfigAPIModel
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	if err := SaveSysConfigs(rows); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func QuerySysConfigs(request sysConfigAPIRequest) ([]sysConfigAPIModel, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return nil, err
	}

	moduleName := normalizeSysConfigModuleName(request.FModuleName)
	query := db.Model(&TbSysConfigs{}).
		Select("FID", "FType", "FValue", "FModuleName", "FVisible", "FEnType", "FValueType", "FValueTypeDetail", "FZhType").
		Where("FModuleName = ?", moduleName)
	if request.FVisible != nil {
		query = query.Where("FVisible = ?", *request.FVisible)
	}
	if value := strings.TrimSpace(request.FType); value != "" {
		likeValue := "%" + value + "%"
		query = query.Where("(FType LIKE ? OR FZhType LIKE ? OR FEnType LIKE ?)", likeValue, likeValue, likeValue)
	}

	var rows []TbSysConfigs
	if err := query.Order("FID asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]sysConfigAPIModel, 0, len(rows))
	for _, row := range rows {
		result = append(result, sysConfigToAPIModel(row))
	}
	return result, nil
}

func SaveSysConfigs(rows []sysConfigAPIModel) error {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := saveSysConfigRow(tx, row); err != nil {
				return err
			}
		}
		return nil
	})
}

func saveSysConfigRow(tx *gorm.DB, row sysConfigAPIModel) error {
	row.FType = strings.TrimSpace(row.FType)
	row.FModuleName = normalizeSysConfigModuleName(row.FModuleName)
	if row.FType == "" {
		return errors.New("FType is required")
	}

	updates := map[string]any{
		"FType":            row.FType,
		"FValue":           row.FValue,
		"FModuleName":      row.FModuleName,
		"FVisible":         row.FVisible,
		"FEnType":          row.FEnType,
		"FValueType":       row.FValueType,
		"FValueTypeDetail": row.FValueTypeDetail,
		"FZhType":          row.FZhType,
	}
	if row.FID > 0 {
		result := tx.Model(&TbSysConfigs{}).Where("FID = ?", row.FID).Updates(updates)
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
	result := tx.Table((&TbSysConfigs{}).TableName()).
		Select("FID").
		Where("FModuleName = ? AND FType = ?", row.FModuleName, row.FType).
		Find(&existing)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return tx.Model(&TbSysConfigs{}).Where("FID = ?", existing.FID).Updates(updates).Error
	}

	now := time.Now()
	item := TbSysConfigs{
		FType:            row.FType,
		FValue:           row.FValue,
		FModuleName:      row.FModuleName,
		FVisible:         row.FVisible,
		FEnType:          row.FEnType,
		FValueType:       row.FValueType,
		FValueTypeDetail: row.FValueTypeDetail,
		FCreateDate:      &now,
		FZhType:          row.FZhType,
	}
	return tx.Create(&item).Error
}

func normalizeSysConfigModuleName(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return "RSS"
	}
	return text
}

func sysConfigToAPIModel(row TbSysConfigs) sysConfigAPIModel {
	return sysConfigAPIModel{
		FID:              row.FID,
		FType:            row.FType,
		FValue:           row.FValue,
		FModuleName:      row.FModuleName,
		FVisible:         row.FVisible,
		FEnType:          row.FEnType,
		FValueType:       row.FValueType,
		FValueTypeDetail: row.FValueTypeDetail,
		FZhType:          row.FZhType,
	}
}
