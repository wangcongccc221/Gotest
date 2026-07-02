package database

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type sysFruitInfoAPIModel struct {
	FID         int `json:"FID"`
	CustomerID  int `json:"CustomerID"`
	SystemID    int `json:"SystemID"`
	BatchNumber int `json:"BatchNumber"`
	BatchWeight int `json:"BatchWeight"`
}

type SysFruitInfoQueryRequest struct {
	FID        *int    `json:"FID,omitempty"`
	CustomerID *int    `json:"CustomerID,omitempty"`
	SystemID   *int    `json:"SystemID,omitempty"`
	PageIndex  *int    `json:"PageIndex,omitempty"`
	PageSize   *int    `json:"PageSize,omitempty"`
	SortOrder  *string `json:"SortOrder,omitempty"`
}

type SysFruitInfoQueryResult struct {
	Items      []TbSysFruitInfo `json:"Items"`
	TotalCount int64            `json:"TotalCount"`
	PageIndex  int              `json:"PageIndex"`
	PageSize   int              `json:"PageSize"`
}

func QuerySysFruitInfo(request SysFruitInfoQueryRequest) (SysFruitInfoQueryResult, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return SysFruitInfoQueryResult{}, err
	}

	pageIndex, pageSize := normalizeFruitInfoPage(request.PageIndex, request.PageSize)
	query := db.Model(&TbSysFruitInfo{})
	if id := intPtrValue(request.FID); id > 0 {
		query = query.Where("FID = ?", id)
	}
	if customerID := intPtrValue(request.CustomerID); customerID > 0 {
		query = query.Where("CustomerID = ?", customerID)
	}
	if systemID := intPtrValue(request.SystemID); systemID > 0 {
		query = query.Where("SystemID = ?", systemID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return SysFruitInfoQueryResult{}, err
	}

	var rows []TbSysFruitInfo
	err = query.
		Order(sysFruitInfoOrderClause(request.SortOrder)).
		Offset((pageIndex - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	if err != nil {
		return SysFruitInfoQueryResult{}, err
	}

	return SysFruitInfoQueryResult{
		Items:      rows,
		TotalCount: total,
		PageIndex:  pageIndex,
		PageSize:   pageSize,
	}, nil
}

func sysFruitInfoOrderClause(order *string) string {
	if strings.EqualFold(strings.TrimSpace(stringPtrValue(order)), "asc") {
		return "CustomerID asc, SystemID asc, FID asc"
	}
	return "CustomerID desc, SystemID asc, FID desc"
}

func registerSysFruitInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/SysFruitInfo")
	group.POST("/GetSysFruitInfo", handleSysFruitInfoGetList)
	group.POST("/SaveSysFruitInfos", handleSysFruitInfoSaveList)
	group.POST("/DeleteSysFruitInfo", handleSysFruitInfoDelete)
}

func handleSysFruitInfoGetList(ctx *gin.Context) {
	var request SysFruitInfoQueryRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	result, err := QuerySysFruitInfo(request)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func handleSysFruitInfoSaveList(ctx *gin.Context) {
	var rows []sysFruitInfoAPIModel
	if err := ctx.ShouldBindJSON(&rows); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	if err := SaveSysFruitInfos(rows); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func handleSysFruitInfoDelete(ctx *gin.Context) {
	var request sysFruitInfoAPIModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	if request.CustomerID <= 0 {
		fruitInfoAPIFail(ctx, "CustomerID is required")
		return
	}

	if err := DeleteSysFruitInfosByCustomerID(request.CustomerID); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func SaveSysFruitInfos(rows []sysFruitInfoAPIModel) error {
	if len(rows) == 0 {
		return nil
	}
	customerID := rows[0].CustomerID
	if customerID <= 0 {
		return errors.New("CustomerID is required")
	}
	for _, row := range rows {
		if row.CustomerID != customerID {
			return errors.New("all rows must share the same CustomerID")
		}
		if row.SystemID <= 0 {
			return errors.New("SystemID must be > 0")
		}
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("CustomerID = ?", customerID).Delete(&TbSysFruitInfo{}).Error; err != nil {
			return err
		}
		items := make([]TbSysFruitInfo, 0, len(rows))
		for _, row := range rows {
			items = append(items, TbSysFruitInfo{
				CustomerID:  row.CustomerID,
				SystemID:    row.SystemID,
				BatchNumber: row.BatchNumber,
				BatchWeight: row.BatchWeight,
			})
		}
		return tx.Create(&items).Error
	})
}

func DeleteSysFruitInfosByCustomerID(customerID int) error {
	if customerID <= 0 {
		return errors.New("CustomerID is required")
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		return err
	}
	return txDeleteSysFruitInfosByCustomerID(db, customerID)
}

func txDeleteSysFruitInfosByCustomerID(db *gorm.DB, customerID int) error {
	return db.Where("CustomerID = ?", customerID).Delete(&TbSysFruitInfo{}).Error
}
