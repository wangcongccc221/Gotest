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
