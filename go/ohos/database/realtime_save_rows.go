package database

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

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
