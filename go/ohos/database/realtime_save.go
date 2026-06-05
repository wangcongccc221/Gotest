package database

import (
	"errors"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"
)

type RealtimeFruitSaveInput struct {
	SavedAt              time.Time
	CustomerName         string
	FarmName             string
	FruitName            string
	ProgramName          string
	BatchNumber          uint64
	BatchWeight          float64
	QualityGradeSum      int
	WeightOrSizeGradeSum int
	ExportSum            int
	ColorGradeName       string
	ShapeGradeName       string
	FlawGradeName        string
	HardGradeName        string
	DensityGradeName     string
	SugarDegreeGradeName string
	Grades               []RealtimeGradeSaveInput
	Exports              []RealtimeExportSaveInput
	Systems              []RealtimeSysFruitSaveInput
	Process              *RealtimeFruitProcessSaveInput
}

type RealtimeGradeSaveInput struct {
	GradeID            int
	BoxNumber          float64
	FruitNumber        uint64
	FruitWeight        float64
	QualityName        string
	WeightOrSizeName   string
	WeightOrSizeLimit  float64
	SelectWeightOrSize string
	TraitWeightOrSize  string
	TraitColor         string
	TraitShape         string
	TraitFlaw          string
	TraitHard          string
	TraitDensity       string
	TraitSugarDegree   string
	FPrice             *float64
}

type RealtimeExportSaveInput struct {
	ExportID    int
	FruitNumber uint64
	FruitWeight float64
	ExitName    string
	SysID       *int
}

type RealtimeSysFruitSaveInput struct {
	SystemID    int
	BatchNumber int
	BatchWeight int
}

type RealtimeFruitProcessSaveInput struct {
	RealWeightCount      float64
	RealWeightCountPer   float64
	SeparationEfficiency float64
	SpeedPercent         float64
	AvgWeight            float64
	RunningDate          string
}

func SaveRealtimeFruitInfo(input RealtimeFruitSaveInput) (int, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return 0, err
	}

	savedAt := input.SavedAt
	if savedAt.IsZero() {
		savedAt = time.Now()
	}

	customerID := 0
	if err := db.Transaction(func(tx *gorm.DB) error {
		programName, err := realtimeSaveProgramName(tx, input.ProgramName)
		if err != nil {
			return err
		}

		fruit, hasFruit, err := realtimeSaveCurrentFruitInfo(tx)
		if err != nil {
			return err
		}
		if hasFruit {
			decreased, err := realtimeSaveSystemCountersDecreased(tx, fruit.CustomerID, input.Systems)
			if err != nil {
				return err
			}
			if decreased {
				fruit = TbFruitInfo{}
				hasFruit = false
			}
		}

		startTime := strings.TrimSpace(fruit.StartTime)
		if !hasFruit || fruit.StartedState != "1" || startTime == "" {
			startTime = savedAt.Format("2006-01-02 15:04:05")
		}

		update := realtimeSaveFruitInfoValues(input, startTime, programName)
		if hasFruit {
			customerID = fruit.CustomerID
			if err := tx.Model(&TbFruitInfo{}).Where("CustomerID = ?", customerID).Updates(update).Error; err != nil {
				return err
			}
		} else {
			batchWeight := input.BatchWeight
			batchNumber := input.BatchNumber
			qualityGradeSum := input.QualityGradeSum
			weightOrSizeGradeSum := input.WeightOrSizeGradeSum
			exportSum := input.ExportSum
			fruit = TbFruitInfo{
				CustomerName:         input.CustomerName,
				FarmName:             input.FarmName,
				FruitName:            input.FruitName,
				StartTime:            startTime,
				EndTime:              "",
				StartedState:         "1",
				CompletedState:       "0",
				BatchWeight:          &batchWeight,
				BatchNumber:          &batchNumber,
				QualityGradeSum:      &qualityGradeSum,
				WeightOrSizeGradeSum: &weightOrSizeGradeSum,
				ExportSum:            &exportSum,
				ColorGradeName:       input.ColorGradeName,
				ShapeGradeName:       input.ShapeGradeName,
				FlawGradeName:        input.FlawGradeName,
				HardGradeName:        input.HardGradeName,
				DensityGradeName:     input.DensityGradeName,
				SugarDegreeGradeName: input.SugarDegreeGradeName,
				ProgramName:          programName,
			}
			if err := tx.Select(realtimeSaveFruitInfoColumns()).Create(&fruit).Error; err != nil {
				return err
			}
			customerID = fruit.CustomerID
		}

		if err := realtimeSaveReplaceGradeInfos(tx, customerID, input.Grades); err != nil {
			return err
		}
		if err := realtimeSaveReplaceExportInfos(tx, customerID, input.Exports); err != nil {
			return err
		}
		if err := realtimeSaveReplaceSysFruitInfos(tx, customerID, input.Systems); err != nil {
			return err
		}
		return realtimeSaveProcessInfo(tx, savedAt, input.Process)
	}); err != nil {
		return 0, err
	}
	return customerID, nil
}

func RealtimeSaveDatabaseForLog() string {
	ormMu.Lock()
	db := activeORM
	dsn := activeORMDSN
	database := activeORMDatabase
	initErr := ormInitErr
	ormMu.Unlock()

	if initErr != nil {
		return "error: " + initErr.Error()
	}
	if db == nil {
		return "not initialized"
	}
	if strings.TrimSpace(database) != "" {
		return database
	}
	if strings.TrimSpace(dsn) != "" {
		return dsn
	}
	return "unknown"
}

func getInitializedFileORMDB() (*gorm.DB, error) {
	ormMu.Lock()
	db := activeORM
	dsn := activeORMDSN
	database := activeORMDatabase
	initErr := ormInitErr
	ormMu.Unlock()

	if initErr != nil {
		return nil, initErr
	}
	if db == nil {
		return nil, errors.New("ORM database is not initialized")
	}
	dsn = strings.TrimSpace(dsn)
	if database == "sqlite-memory" || strings.Contains(dsn, "mode=memory") || strings.EqualFold(dsn, ":memory:") || dsn == "" {
		return nil, errors.New("ORM database is not a file database")
	}
	return db, nil
}

func realtimeSaveCurrentFruitInfo(tx *gorm.DB) (TbFruitInfo, bool, error) {
	var fruit TbFruitInfo
	err := tx.Where("CompletedState = ?", "0").Order("CustomerID desc").First(&fruit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TbFruitInfo{}, false, nil
	}
	if err != nil {
		return TbFruitInfo{}, false, err
	}
	return fruit, true, nil
}

func realtimeSaveSystemCountersDecreased(tx *gorm.DB, customerID int, systems []RealtimeSysFruitSaveInput) (bool, error) {
	var oldSystems []TbSysFruitInfo
	if err := tx.Where("CustomerID = ?", customerID).Find(&oldSystems).Error; err != nil {
		return false, err
	}
	if len(oldSystems) == 0 || len(systems) == 0 {
		return false, nil
	}

	currentBySystemID := make(map[int]RealtimeSysFruitSaveInput, len(systems))
	for _, system := range systems {
		currentBySystemID[system.SystemID] = system
	}
	for _, oldSystem := range oldSystems {
		current, ok := currentBySystemID[oldSystem.SystemID]
		if !ok {
			continue
		}
		if current.BatchNumber < oldSystem.BatchNumber || current.BatchWeight < oldSystem.BatchWeight {
			return true, nil
		}
	}
	return false, nil
}

func realtimeSaveFruitInfoValues(input RealtimeFruitSaveInput, startTime string, programName string) map[string]any {
	return map[string]any{
		"CustomerName":         input.CustomerName,
		"FarmName":             input.FarmName,
		"FruitName":            input.FruitName,
		"StartTime":            startTime,
		"EndTime":              "",
		"StartedState":         "1",
		"CompletedState":       "0",
		"BatchWeight":          input.BatchWeight,
		"BatchNumber":          input.BatchNumber,
		"QualityGradeSum":      input.QualityGradeSum,
		"WeightOrSizeGradeSum": input.WeightOrSizeGradeSum,
		"ExportSum":            input.ExportSum,
		"ColorGradeName":       input.ColorGradeName,
		"ShapeGradeName":       input.ShapeGradeName,
		"FlawGradeName":        input.FlawGradeName,
		"HardGradeName":        input.HardGradeName,
		"DensityGradeName":     input.DensityGradeName,
		"SugarDegreeGradeName": input.SugarDegreeGradeName,
		"ProgramName":          programName,
	}
}

func realtimeSaveFruitInfoColumns() []string {
	return []string{
		"CustomerName",
		"FarmName",
		"FruitName",
		"StartTime",
		"EndTime",
		"StartedState",
		"CompletedState",
		"BatchWeight",
		"BatchNumber",
		"QualityGradeSum",
		"WeightOrSizeGradeSum",
		"ExportSum",
		"ColorGradeName",
		"ShapeGradeName",
		"FlawGradeName",
		"HardGradeName",
		"DensityGradeName",
		"SugarDegreeGradeName",
		"ProgramName",
	}
}

func realtimeSaveProgramName(tx *gorm.DB, fallback string) (string, error) {
	value, err := realtimeSaveConfigValue(tx, "用户配置参数")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) != "" {
		return value, nil
	}
	return fallback, nil
}

func realtimeSaveConfigValue(tx *gorm.DB, configType string) (string, error) {
	var item TbSysConfigs
	err := tx.Where("FModuleName = ? AND FType = ?", "RSS", configType).Order("FID desc").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return item.FValue, nil
}

func realtimeSaveReplaceGradeInfos(tx *gorm.DB, customerID int, grades []RealtimeGradeSaveInput) error {
	if err := tx.Where("CustomerID = ?", customerID).Delete(&TbGradeInfo{}).Error; err != nil {
		return err
	}
	if len(grades) == 0 {
		return nil
	}

	items := make([]TbGradeInfo, 0, len(grades))
	for _, grade := range grades {
		boxNumber := grade.BoxNumber
		fruitNumber := grade.FruitNumber
		fruitWeight := grade.FruitWeight
		weightOrSizeLimit := grade.WeightOrSizeLimit
		price := 0.0
		if grade.FPrice != nil {
			price = *grade.FPrice
		}
		item := TbGradeInfo{
			CustomerID:         customerID,
			GradeID:            grade.GradeID,
			BoxNumber:          &boxNumber,
			FruitNumber:        &fruitNumber,
			FruitWeight:        &fruitWeight,
			QualityName:        grade.QualityName,
			WeightOrSizeName:   grade.WeightOrSizeName,
			WeightOrSizeLimit:  &weightOrSizeLimit,
			SelectWeightOrSize: grade.SelectWeightOrSize,
			TraitWeightOrSize:  grade.TraitWeightOrSize,
			TraitColor:         grade.TraitColor,
			TraitShape:         grade.TraitShape,
			TraitFlaw:          grade.TraitFlaw,
			TraitHard:          grade.TraitHard,
			TraitDensity:       grade.TraitDensity,
			TraitSugarDegree:   grade.TraitSugarDegree,
			FPrice:             &price,
		}
		items = append(items, item)
	}
	return tx.Select(realtimeSaveGradeInfoColumns()).Create(&items).Error
}

func realtimeSaveGradeInfoColumns() []string {
	return []string{
		"CustomerID",
		"GradeID",
		"BoxNumber",
		"FruitNumber",
		"FruitWeight",
		"QualityName",
		"WeightOrSizeName",
		"WeightOrSizeLimit",
		"SelectWeightOrSize",
		"TraitWeightOrSize",
		"TraitColor",
		"TraitShape",
		"TraitFlaw",
		"TraitHard",
		"TraitDensity",
		"TraitSugarDegree",
		"FPrice",
	}
}

func realtimeSaveReplaceExportInfos(tx *gorm.DB, customerID int, exports []RealtimeExportSaveInput) error {
	if err := tx.Where("CustomerID = ?", customerID).Delete(&TbExportInfo{}).Error; err != nil {
		return err
	}
	if len(exports) == 0 {
		return nil
	}

	items := make([]TbExportInfo, 0, len(exports))
	for _, export := range exports {
		fruitNumber := export.FruitNumber
		fruitWeight := export.FruitWeight
		items = append(items, TbExportInfo{
			CustomerID:  customerID,
			ExportID:    export.ExportID,
			FruitNumber: &fruitNumber,
			FruitWeight: &fruitWeight,
			ExitName:    export.ExitName,
			SysID:       export.SysID,
		})
	}
	return tx.Select(realtimeSaveExportInfoColumns()).Create(&items).Error
}

func realtimeSaveExportInfoColumns() []string {
	return []string{
		"CustomerID",
		"ExportID",
		"FruitNumber",
		"FruitWeight",
		"ExitName",
	}
}

func realtimeSaveReplaceSysFruitInfos(tx *gorm.DB, customerID int, systems []RealtimeSysFruitSaveInput) error {
	if err := tx.Where("CustomerID = ?", customerID).Delete(&TbSysFruitInfo{}).Error; err != nil {
		return err
	}
	if len(systems) == 0 {
		return nil
	}

	items := make([]TbSysFruitInfo, 0, len(systems))
	for _, system := range systems {
		items = append(items, TbSysFruitInfo{
			CustomerID:  customerID,
			SystemID:    system.SystemID,
			BatchNumber: system.BatchNumber,
			BatchWeight: system.BatchWeight,
		})
	}
	return tx.Create(&items).Error
}

func realtimeSaveProcessInfo(tx *gorm.DB, savedAt time.Time, process *RealtimeFruitProcessSaveInput) error {
	if process == nil || strings.TrimSpace(process.RunningDate) == "" {
		return nil
	}
	if err := EnsureFruitProcessInfoYearTable(tx, savedAt.Year()); err != nil {
		return err
	}

	tableName := fruitProcessInfoTableName(savedAt.Year())
	var count int64
	if err := tx.Table(tableName).Where("RunningDate = ?", process.RunningDate).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	realWeightCount := realtimeFiniteFloat(process.RealWeightCount)
	realWeightCountPer := realtimeFiniteFloat(process.RealWeightCountPer)
	separationEfficiency := realtimeFiniteFloat(process.SeparationEfficiency)
	speedPercent := realtimeFiniteFloat(process.SpeedPercent)
	avgWeight := realtimeFiniteFloat(process.AvgWeight)
	item := TbFruitProcessInfo{
		RealWeightCount:      &realWeightCount,
		RealWeightCountPer:   &realWeightCountPer,
		SeparationEfficiency: &separationEfficiency,
		SpeedPercent:         &speedPercent,
		AvgWeight:            &avgWeight,
		RunningDate:          process.RunningDate,
	}
	return tx.Table(tableName).Select(realtimeSaveFruitProcessInfoColumns()).Create(&item).Error
}

func realtimeSaveFruitProcessInfoColumns() []string {
	return []string{
		"RealWeightCount",
		"RealWeightCountPer",
		"SeparationEfficiency",
		"SpeedPercent",
		"AvgWeight",
		"RunningDate",
	}
}

func realtimeFiniteFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}
