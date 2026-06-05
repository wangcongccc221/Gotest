package database

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type fruitInfoAPIEnvelope struct {
	ReturnCode    int    `json:"returnCode"`
	ReturnMessage string `json:"returnMessage"`
	Data          string `json:"data"`
}

type fruitInfoPageResponse struct {
	Items      []fruitInfoAPIModel `json:"Items"`
	TotalCount int64               `json:"TotalCount"`
	PageIndex  int                 `json:"PageIndex"`
	PageSize   int                 `json:"PageSize"`
}

type fruitInfoAPIRequest struct {
	CustomerID      *int    `json:"CustomerID"`
	MajorCustomerID *int    `json:"MajorCustomerID"`
	SysID           *int    `json:"SysID"`
	CompletedState  *string `json:"CompletedState"`
	CustomerName    *string `json:"CustomerName"`
	FarmName        *string `json:"FarmName"`
	FruitName       *string `json:"FruitName"`
	StartTime       *string `json:"StartTime"`
	EndTime         *string `json:"EndTime"`
	FVisible        *int    `json:"FVisible"`
	PageIndex       *int    `json:"PageIndex"`
	PageSize        *int    `json:"PageSize"`
	SortColumn      *string `json:"SortColumn"`
	SortOrder       *string `json:"SortOrder"`
}

type updateFruitCustomerInfoRequest struct {
	CustomerID   int     `json:"CustomerID"`
	CustomerName *string `json:"CustomerName"`
	FarmName     *string `json:"FarmName"`
	FruitName    *string `json:"FruitName"`
	SortBaseName *string `json:"SortBaseName"`
	ProgramName  *string `json:"ProgramName"`
	FBatchNo     *string `json:"FBatchNo"`
}

type fruitInfoAPIModel struct {
	CustomerID           int                  `json:"CustomerID"`
	SysID                int                  `json:"SysID"`
	MajorCustomerID      int                  `json:"MajorCustomerID"`
	FBatchNo             string               `json:"FBatchNo"`
	CustomerName         string               `json:"CustomerName"`
	FarmName             string               `json:"FarmName"`
	FruitName            string               `json:"FruitName"`
	SortBaseName         string               `json:"SortBaseName"`
	StartTime            string               `json:"StartTime"`
	EndTime              string               `json:"EndTime"`
	StartedState         string               `json:"StartedState"`
	CompletedState       string               `json:"CompletedState"`
	BatchWeight          float64              `json:"BatchWeight"`
	BatchNumber          uint64               `json:"BatchNumber"`
	QualityGradeSum      int                  `json:"QualityGradeSum"`
	WeightOrSizeGradeSum int                  `json:"WeightOrSizeGradeSum"`
	ExportSum            int                  `json:"ExportSum"`
	ColorGradeName       string               `json:"ColorGradeName"`
	ShapeGradeName       string               `json:"ShapeGradeName"`
	FlawGradeName        string               `json:"FlawGradeName"`
	HardGradeName        string               `json:"HardGradeName"`
	DensityGradeName     string               `json:"DensityGradeName"`
	SugarDegreeGradeName string               `json:"SugarDegreeGradeName"`
	ProgramName          string               `json:"ProgramName"`
	FVisible             int                  `json:"FVisible"`
	PageSize             int                  `json:"PageSize"`
	PageIndex            int                  `json:"PageIndex"`
	IsMerge              bool                 `json:"IsMerge"`
	SortColumn           string               `json:"SortColumn"`
	SortOrder            string               `json:"SortOrder"`
	ListTbGradeInfo      []gradeInfoAPIModel  `json:"List_tb_GradeInfo"`
	ListTbExportInfo     []exportInfoAPIModel `json:"List_tb_ExportInfo"`
}

type gradeInfoAPIModel struct {
	CustomerID         int     `json:"CustomerID"`
	GradeID            int     `json:"GradeID"`
	BoxNumber          float64 `json:"BoxNumber"`
	FruitNumber        uint64  `json:"FruitNumber"`
	FruitWeight        float64 `json:"FruitWeight"`
	QualityName        string  `json:"QualityName"`
	WeightOrSizeName   string  `json:"WeightOrSizeName"`
	WeightOrSizeLimit  float64 `json:"WeightOrSizeLimit"`
	SelectWeightOrSize string  `json:"SelectWeightOrSize"`
	TraitWeightOrSize  string  `json:"TraitWeightOrSize"`
	TraitColor         string  `json:"TraitColor"`
	TraitShape         string  `json:"TraitShape"`
	TraitFlaw          string  `json:"TraitFlaw"`
	TraitHard          string  `json:"TraitHard"`
	TraitDensity       string  `json:"TraitDensity"`
	TraitSugarDegree   string  `json:"TraitSugarDegree"`
	FPrice             float64 `json:"FPrice"`
	NSizeMax           float64 `json:"nSizeMax"`
	NSizeMin           float64 `json:"nSizeMin"`
}

type exportInfoAPIModel struct {
	CustomerID  int     `json:"CustomerID"`
	ExportID    int     `json:"ExportID"`
	FruitNumber uint64  `json:"FruitNumber"`
	FruitWeight float64 `json:"FruitWeight"`
	ExitName    string  `json:"ExitName"`
}

func registerFruitInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/FruitInfo")
	group.POST("/GetPageFruitInfo", handleFruitInfoGetPage)
	group.POST("/GetFruitInfo", handleFruitInfoGetOne)
	group.POST("/GetAllFruitInfo", handleFruitInfoGetAll)
	group.POST("/GetFruitByListCustomerID", handleFruitInfoGetByIDList)
	group.POST("/DeleteFruitInfo", handleFruitInfoDelete)
	group.POST("/UpdateFruitCustomerInfo", handleFruitInfoUpdateCustomer)
}

func handleFruitInfoGetPage(ctx *gin.Context) {
	var request fruitInfoAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	pageIndex, pageSize := normalizeFruitInfoPage(request.PageIndex, request.PageSize)
	query := applyFruitInfoFilters(db.Model(&TbFruitInfo{}), request)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	var rows []TbFruitInfo
	err = query.
		Order(fruitInfoOrderClause(request.SortColumn, request.SortOrder)).
		Offset((pageIndex - 1) * pageSize).
		Limit(pageSize).
		Find(&rows).Error
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	items := make([]fruitInfoAPIModel, 0, len(rows))
	for _, row := range rows {
		items = append(items, fruitInfoToAPIModel(row, nil, nil))
	}

	fruitInfoAPIOK(ctx, fruitInfoPageResponse{
		Items:      items,
		TotalCount: total,
		PageIndex:  pageIndex,
		PageSize:   pageSize,
	})
}

func handleFruitInfoGetOne(ctx *gin.Context) {
	var request fruitInfoAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	customerID := intPtrValue(request.CustomerID)
	if customerID <= 0 {
		fruitInfoAPIFail(ctx, "CustomerID is required")
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	item, found, err := getFruitInfoDetailByCustomerID(db, customerID)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	if !found {
		fruitInfoAPIOKRaw(ctx, "")
		return
	}
	fruitInfoAPIOK(ctx, item)
}

func handleFruitInfoGetAll(ctx *gin.Context) {
	var request fruitInfoAPIRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	majorCustomerID := intPtrValue(request.MajorCustomerID)
	if majorCustomerID <= 0 {
		fruitInfoAPIOK(ctx, []fruitInfoAPIModel{})
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	var rows []TbFruitInfo
	err = db.Where("MajorCustomerID = ? OR CustomerID = ?", majorCustomerID, majorCustomerID).
		Order("SysID asc, CustomerID asc").
		Find(&rows).Error
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	items, err := fruitInfoDetailsFromRows(db, rows)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, items)
}

func handleFruitInfoGetByIDList(ctx *gin.Context) {
	var requests []fruitInfoAPIRequest
	if err := ctx.ShouldBindJSON(&requests); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	items := make([]fruitInfoAPIModel, 0, len(requests))
	seen := make(map[int]struct{}, len(requests))
	for _, request := range requests {
		customerID := intPtrValue(request.CustomerID)
		if customerID <= 0 {
			continue
		}
		if _, ok := seen[customerID]; ok {
			continue
		}
		seen[customerID] = struct{}{}
		item, found, err := getFruitInfoDetailByCustomerID(db, customerID)
		if err != nil {
			fruitInfoAPIFail(ctx, err.Error())
			return
		}
		if found {
			items = append(items, item)
		}
	}
	fruitInfoAPIOK(ctx, items)
}

func handleFruitInfoDelete(ctx *gin.Context) {
	var customerIDs []int
	if err := ctx.ShouldBindJSON(&customerIDs); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	ids := make([]int, 0, len(customerIDs))
	for _, id := range customerIDs {
		if id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		fruitInfoAPIOKRaw(ctx, "")
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	if err := db.Model(&TbFruitInfo{}).Where("CustomerID IN ?", ids).Update("FVisible", 0).Error; err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func handleFruitInfoUpdateCustomer(ctx *gin.Context) {
	var request updateFruitCustomerInfoRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	if request.CustomerID <= 0 {
		fruitInfoAPIFail(ctx, "CustomerID is required")
		return
	}

	updates := make(map[string]any)
	if request.CustomerName != nil {
		updates["CustomerName"] = strings.TrimSpace(*request.CustomerName)
	}
	if request.FarmName != nil {
		updates["FarmName"] = strings.TrimSpace(*request.FarmName)
	}
	if request.FruitName != nil {
		updates["FruitName"] = strings.TrimSpace(*request.FruitName)
	}
	if request.SortBaseName != nil {
		updates["ProgramName"] = strings.TrimSpace(*request.SortBaseName)
	}
	if request.ProgramName != nil {
		updates["ProgramName"] = strings.TrimSpace(*request.ProgramName)
	}
	if request.FBatchNo != nil {
		updates["FBatchNo"] = strings.TrimSpace(*request.FBatchNo)
	}
	if len(updates) == 0 {
		fruitInfoAPIOKRaw(ctx, "")
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	if err := db.Model(&TbFruitInfo{}).Where("CustomerID = ?", request.CustomerID).Updates(updates).Error; err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, "")
}

func applyFruitInfoFilters(query *gorm.DB, request fruitInfoAPIRequest) *gorm.DB {
	if request.FVisible != nil {
		if *request.FVisible == 0 {
			query = query.Where("FVisible = ?", 0)
		} else {
			query = query.Where("(FVisible IS NULL OR FVisible <> ?)", 0)
		}
	}
	if request.CustomerID != nil && *request.CustomerID > 0 {
		query = query.Where("CustomerID = ?", *request.CustomerID)
	}
	if request.MajorCustomerID != nil && *request.MajorCustomerID > 0 {
		query = query.Where("MajorCustomerID = ?", *request.MajorCustomerID)
	}
	if request.SysID != nil {
		query = query.Where("SysID = ?", *request.SysID)
	}
	if value := trimStringPtr(request.CompletedState); value != "" {
		query = query.Where("CompletedState = ?", value)
	}
	if value := trimStringPtr(request.CustomerName); value != "" {
		query = query.Where("CustomerName LIKE ?", "%"+value+"%")
	}
	if value := trimStringPtr(request.FarmName); value != "" {
		query = query.Where("FarmName LIKE ?", "%"+value+"%")
	}
	if value := trimStringPtr(request.FruitName); value != "" {
		query = query.Where("FruitName LIKE ?", "%"+value+"%")
	}
	if value := trimStringPtr(request.StartTime); value != "" {
		query = query.Where("StartTime >= ?", value)
	}
	if value := trimStringPtr(request.EndTime); value != "" {
		query = query.Where("EndTime <= ?", value)
	}
	return query
}

func getFruitInfoDetailByCustomerID(db *gorm.DB, customerID int) (fruitInfoAPIModel, bool, error) {
	var fruit TbFruitInfo
	err := db.Where("CustomerID = ?", customerID).First(&fruit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fruitInfoAPIModel{}, false, nil
	}
	if err != nil {
		return fruitInfoAPIModel{}, false, err
	}

	var grades []TbGradeInfo
	if err := db.Where("CustomerID = ?", customerID).Order("GradeID asc").Find(&grades).Error; err != nil {
		return fruitInfoAPIModel{}, false, err
	}
	var exports []TbExportInfo
	if err := db.Where("CustomerID = ?", customerID).Order("ExportID asc").Find(&exports).Error; err != nil {
		return fruitInfoAPIModel{}, false, err
	}
	return fruitInfoToAPIModel(fruit, grades, exports), true, nil
}

func fruitInfoDetailsFromRows(db *gorm.DB, rows []TbFruitInfo) ([]fruitInfoAPIModel, error) {
	items := make([]fruitInfoAPIModel, 0, len(rows))
	for _, row := range rows {
		item, found, err := getFruitInfoDetailByCustomerID(db, row.CustomerID)
		if err != nil {
			return nil, err
		}
		if found {
			items = append(items, item)
		}
	}
	return items, nil
}

func fruitInfoToAPIModel(fruit TbFruitInfo, grades []TbGradeInfo, exports []TbExportInfo) fruitInfoAPIModel {
	item := fruitInfoAPIModel{
		CustomerID:           fruit.CustomerID,
		SysID:                intPtrValue(fruit.SysID),
		MajorCustomerID:      intPtrValue(fruit.MajorCustomerID),
		FBatchNo:             fruit.FBatchNo,
		CustomerName:         fruit.CustomerName,
		FarmName:             fruit.FarmName,
		FruitName:            fruit.FruitName,
		SortBaseName:         fruit.ProgramName,
		StartTime:            fruit.StartTime,
		EndTime:              fruit.EndTime,
		StartedState:         fruit.StartedState,
		CompletedState:       fruit.CompletedState,
		BatchWeight:          float64PtrValue(fruit.BatchWeight),
		BatchNumber:          uint64PtrValue(fruit.BatchNumber),
		QualityGradeSum:      intPtrValue(fruit.QualityGradeSum),
		WeightOrSizeGradeSum: intPtrValue(fruit.WeightOrSizeGradeSum),
		ExportSum:            intPtrValue(fruit.ExportSum),
		ColorGradeName:       fruit.ColorGradeName,
		ShapeGradeName:       fruit.ShapeGradeName,
		FlawGradeName:        fruit.FlawGradeName,
		HardGradeName:        fruit.HardGradeName,
		DensityGradeName:     fruit.DensityGradeName,
		SugarDegreeGradeName: fruit.SugarDegreeGradeName,
		ProgramName:          fruit.ProgramName,
		FVisible:             visibleValue(fruit.FVisible),
		ListTbGradeInfo:      make([]gradeInfoAPIModel, 0, len(grades)),
		ListTbExportInfo:     make([]exportInfoAPIModel, 0, len(exports)),
	}
	for _, grade := range grades {
		item.ListTbGradeInfo = append(item.ListTbGradeInfo, gradeInfoToAPIModel(grade))
	}
	for _, export := range exports {
		item.ListTbExportInfo = append(item.ListTbExportInfo, exportInfoToAPIModel(export))
	}
	return item
}

func gradeInfoToAPIModel(grade TbGradeInfo) gradeInfoAPIModel {
	return gradeInfoAPIModel{
		CustomerID:         grade.CustomerID,
		GradeID:            grade.GradeID,
		BoxNumber:          float64PtrValue(grade.BoxNumber),
		FruitNumber:        uint64PtrValue(grade.FruitNumber),
		FruitWeight:        float64PtrValue(grade.FruitWeight),
		QualityName:        grade.QualityName,
		WeightOrSizeName:   grade.WeightOrSizeName,
		WeightOrSizeLimit:  float64PtrValue(grade.WeightOrSizeLimit),
		SelectWeightOrSize: grade.SelectWeightOrSize,
		TraitWeightOrSize:  grade.TraitWeightOrSize,
		TraitColor:         grade.TraitColor,
		TraitShape:         grade.TraitShape,
		TraitFlaw:          grade.TraitFlaw,
		TraitHard:          grade.TraitHard,
		TraitDensity:       grade.TraitDensity,
		TraitSugarDegree:   grade.TraitSugarDegree,
		FPrice:             float64PtrValue(grade.FPrice),
		NSizeMax:           grade.NSizeMax,
		NSizeMin:           grade.NSizeMin,
	}
}

func exportInfoToAPIModel(export TbExportInfo) exportInfoAPIModel {
	return exportInfoAPIModel{
		CustomerID:  export.CustomerID,
		ExportID:    export.ExportID,
		FruitNumber: uint64PtrValue(export.FruitNumber),
		FruitWeight: float64PtrValue(export.FruitWeight),
		ExitName:    export.ExitName,
	}
}

func normalizeFruitInfoPage(pageIndex *int, pageSize *int) (int, int) {
	index := intPtrValue(pageIndex)
	size := intPtrValue(pageSize)
	if index <= 0 {
		index = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 1000 {
		size = 1000
	}
	return index, size
}

func fruitInfoOrderClause(column *string, order *string) string {
	columnName := "CustomerID"
	switch strings.TrimSpace(stringPtrValue(column)) {
	case "StartTime":
		columnName = "StartTime"
	case "EndTime":
		columnName = "EndTime"
	case "BatchWeight":
		columnName = "BatchWeight"
	case "BatchNumber":
		columnName = "BatchNumber"
	case "CustomerName":
		columnName = "CustomerName"
	case "FarmName":
		columnName = "FarmName"
	case "FruitName":
		columnName = "FruitName"
	}

	direction := "desc"
	if strings.EqualFold(strings.TrimSpace(stringPtrValue(order)), "asc") {
		direction = "asc"
	}
	return columnName + " " + direction
}

func fruitInfoAPIOK(ctx *gin.Context, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOKRaw(ctx, string(bytes))
}

func fruitInfoAPIOKRaw(ctx *gin.Context, data string) {
	ctx.JSON(http.StatusOK, fruitInfoAPIEnvelope{
		ReturnCode:    1,
		ReturnMessage: "",
		Data:          data,
	})
}

func fruitInfoAPIFail(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, fruitInfoAPIEnvelope{
		ReturnCode:    0,
		ReturnMessage: message,
		Data:          "",
	})
}

func trimStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func float64PtrValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func uint64PtrValue(value *uint64) uint64 {
	if value == nil {
		return 0
	}
	return *value
}

func visibleValue(value *int) int {
	if value == nil {
		return 1
	}
	return *value
}
