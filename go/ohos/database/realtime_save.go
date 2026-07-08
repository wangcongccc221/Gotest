package database

import (
	"errors"
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
	// ClearWithinBatch 为真时(48 数据清零语义):计数器递减不结束批次,批内数据缩回重新累计
	ClearWithinBatch bool
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
		savedAt = databaseNow()
	} else {
		savedAt = databaseLocalTime(savedAt)
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
			// 计数器递减，说明设备清零或重启，需要结束旧批次并开始新批次;
			// 但数据清零(ClearWithinBatch,对齐48 HC_SERVICE_CMD_CLEAR)例外:批次保留,数据缩回重新累计
			if decreased && !input.ClearWithinBatch {
				endTime := savedAt.Format("2006-01-02 15:04:05")
				if err := tx.Model(&TbFruitInfo{}).
					Where("CustomerID = ?", fruit.CustomerID).
					Updates(map[string]any{
						"EndTime":        endTime,
						"CompletedState": "1", // 标记旧批次为已完成
					}).Error; err != nil {
					return err
				}
				fruit = TbFruitInfo{}
				hasFruit = false
			}
		}

		startTime := strings.TrimSpace(fruit.StartTime)
		if !hasFruit || fruit.StartedState != "1" || startTime == "" {
			startTime = savedAt.Format("2006-01-02 15:04:05")
		}

		if hasFruit {
			customerID = fruit.CustomerID
			update := realtimeSaveExistingFruitInfoValues(input, startTime, programName)
			if err := tx.Model(&TbFruitInfo{}).Where("CustomerID = ?", customerID).Updates(update).Error; err != nil {
				return err
			}
		} else {
			batchWeight := input.BatchWeight
			batchNumber := input.BatchNumber
			qualityGradeSum := input.QualityGradeSum
			weightOrSizeGradeSum := input.WeightOrSizeGradeSum
			exportSum := input.ExportSum
			fVisible := 1
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
				FVisible:             &fVisible,
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

// ClearCurrentFruitBatchData 立即清空当前未完成批次的累计数据(对齐 48 服务端 HC_SERVICE_CMD_CLEAR):
// 批次行与原开始时间保留,计数清零,等级/出口/子系统明细删除,后续统计在同批次上重新累计。
// 由数据清零指令直接触发,不依赖统计流——设备已停机时清零,库里也当场归零。
func ClearCurrentFruitBatchData() (int, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return 0, err
	}

	customerID := 0
	if err := db.Transaction(func(tx *gorm.DB) error {
		fruit, hasFruit, err := realtimeSaveCurrentFruitInfo(tx)
		if err != nil || !hasFruit {
			return err
		}
		customerID = fruit.CustomerID
		if err := tx.Model(&TbFruitInfo{}).Where("CustomerID = ?", customerID).Updates(map[string]any{
			"BatchWeight": 0,
			"BatchNumber": 0,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("CustomerID = ?", customerID).Delete(&TbGradeInfo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("CustomerID = ?", customerID).Delete(&TbExportInfo{}).Error; err != nil {
			return err
		}
		return tx.Where("CustomerID = ?", customerID).Delete(&TbSysFruitInfo{}).Error
	}); err != nil {
		return 0, err
	}
	return customerID, nil
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
