package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

func migrateORMTables(db *gorm.DB) error {
	models := []any{
		&DeviceRecord{},
		&TbBarcodeRules{},
		&TbBaseFault{},
		&TbBinTipInfo{},
		&TbClientInfo{},
		&TbConfigs{},
		&TbCustomer{},
		&TbCustomField{},
		&TbCustomFieldSource{},
		&TbCustomFieldValue{},
		&TbDeviceConfig{},
		&TbDeviceConfigDetails{},
		&TbEfficiencyInfo{},
		&TbEmptyBoxInfo{},
		&TbFruitInfo{},
		&TbGradeInfo{},
		&TbExportInfo{},
		&TbExportConfigs{},
		&TbExportDetailInfo{},
		&TbExitGroupInfo{},
		&TbFaultInfo{},
		&TbFeedingInfo{},
		&TbGradeDetailInfo{},
		&TbQualityInfo{},
		&TbSortExitInfo{},
		&TbFruitDetails{},
		&TbProcessInfoTable{},
		&TbFruitStatisticsInfo{},
		&TbSortExitSumInfo{},
		&TbFruitFactorInterval{},
		&TbFruitFactorStatisticInfo{},
		&TbFruitWeightFactorStatisticInfo{},
		&TbGradeRankDetail{},
		&TbGradeRankInfo{},
		&TbInfruscanFruitInfo{},
		&TbOrderInfo{},
		&TbOutletBoxDetailInfo{},
		&TbOutletBoxInfo{},
		&TbPackingInfo{},
		&TbPackingSpec{},
		&TbPalletBoxInfo{},
		&TbPalletInfo{},
		&TbParamDetail{},
		&TbParamType{},
		&TbProductBatchInfo{},
		&TbProductInfo{},
		&TbRSSLog{},
		&TbSampleInfo{},
		&TbSelectMaterialInfo{},
		&TbSoftSortBelts{},
		&TbSoftSortEvents{},
		&TbSpectrometerJDXInfo{},
		&TbSpectrometerRawInfo{},
		&TbSpectrometerSampleRawInfo{},
		&TbSysFruitInfo{},
		&TbFruitProcessInfoPercent{},
		&TbRunningTimeInfo{},
		&TbSeparationEfficiencyInfo{},
		&TbSysConfigs{},
		&TbPriceInfo{},
		&TbSqlInfo{},
		&TbUploadInfo{},
		&TbFileInfo{},
		&TbLocation{},
		&TbFTPInfo{},
		&TbUpdateVersion{},
	}
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			if isSQLiteTableAlreadyExistsError(err) {
				continue
			}
			return err
		}
	}
	year := time.Now().Year()
	if err := EnsureFruitProcessInfoYearTable(db, year); err != nil {
		return err
	}
	if err := EnsureFruitDetailsMonthTables(db, year); err != nil {
		return err
	}
	return EnsurePackingInfoMonthTables(db, year)
}

func isSQLiteTableAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "table") && strings.Contains(text, "already exists")
}

func EnsureFruitProcessInfoYearTable(db *gorm.DB, year int) error {
	if year < 2000 || year > 9999 {
		return fmt.Errorf("invalid fruit process info year: %d", year)
	}
	return db.Table(fruitProcessInfoTableName(year)).AutoMigrate(&TbFruitProcessInfo{})
}

func EnsureFruitDetailsMonthTables(db *gorm.DB, year int) error {
	if year < 2000 || year > 9999 {
		return fmt.Errorf("invalid fruit details year: %d", year)
	}
	for month := 1; month <= 12; month++ {
		if err := db.Table(fruitDetailsMonthTableName(year, month)).AutoMigrate(&TbFruitDetails{}); err != nil {
			return err
		}
	}
	return nil
}

func EnsurePackingInfoMonthTables(db *gorm.DB, year int) error {
	if year < 2000 || year > 9999 {
		return fmt.Errorf("invalid packing info year: %d", year)
	}
	for month := 1; month <= 12; month++ {
		if err := db.Table(packingInfoMonthTableName(year, month)).AutoMigrate(&TbPackingInfo{}); err != nil {
			return err
		}
	}
	return nil
}

func fruitDetailsMonthTableName(year int, month int) string {
	return fmt.Sprintf("tb_fruitdetails_%04d%02d", year, month)
}

func packingInfoMonthTableName(year int, month int) string {
	return fmt.Sprintf("tb_packinginfo%04d%02d", year, month)
}
