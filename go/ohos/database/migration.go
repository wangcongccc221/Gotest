package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

func migrateORMTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&DeviceRecord{},
		&TbFruitInfo{},
		&TbGradeInfo{},
		&TbExportInfo{},
		&TbSysFruitInfo{},
		&TbFruitProcessInfoPercent{},
		&TbRunningTimeInfo{},
		&TbSeparationEfficiencyInfo{},
		&TbClientInfo{},
		&TbSysConfigs{},
		&TbCustomer{},
		&TbBaseFault{},
		&TbFaultInfo{},
		&TbPriceInfo{},
		&TbSqlInfo{},
		&TbUploadInfo{},
		&TbFileInfo{},
		&TbLocation{},
		&TbFTPInfo{},
		&TbUpdateVersion{},
	); err != nil {
		return err
	}
	return EnsureFruitProcessInfoYearTable(db, time.Now().Year())
}

func EnsureFruitProcessInfoYearTable(db *gorm.DB, year int) error {
	if year < 2000 || year > 9999 {
		return fmt.Errorf("invalid fruit process info year: %d", year)
	}
	return db.Table(fruitProcessInfoTableName(year)).AutoMigrate(&TbFruitProcessInfo{})
}
