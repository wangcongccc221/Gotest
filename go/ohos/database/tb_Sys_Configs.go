package database

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TbSysConfigs struct {
	FID              int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FType            string     `json:"FType" gorm:"column:FType;type:longtext"`
	FValue           string     `json:"FValue" gorm:"column:FValue;type:longtext"`
	FModuleName      string     `json:"FModuleName" gorm:"column:FModuleName;type:longtext"`
	FVisible         int        `json:"FVisible" gorm:"column:FVisible;not null;default:0"`
	FEnType          string     `json:"FEnType" gorm:"column:FEnType;type:longtext"`
	FValueType       int        `json:"FValueType" gorm:"column:FValueType;not null;default:0"`
	FValueTypeDetail string     `json:"FValueTypeDetail" gorm:"column:FValueTypeDetail;type:longtext"`
	FSubSystem       *int       `json:"FSubSystem" gorm:"column:FSubSystem"`
	FCreateDate      *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FZhType          string     `json:"FZhType" gorm:"-"`
}

func (TbSysConfigs) TableName() string {
	return "tb_sys_configs"
}

func GetConfigValue(configType string) (string, error) {
	db, err := getORMDB()
	if err != nil {
		return "", err
	}
	return getConfigValueWithDB(db, configType)
}

func getConfigValueWithDB(db *gorm.DB, configType string) (string, error) {
	value, _, err := getConfigValueRow(db, configType, false)
	return value, err
}

func getConfigValueRow(db *gorm.DB, configType string, preferNonEmpty bool) (string, int, error) {
	var row struct {
		FID    int    `gorm:"column:FID"`
		FValue string `gorm:"column:FValue"`
	}

	query := db.Table((&TbSysConfigs{}).TableName()).
		Select("FID, FValue").
		Where("FModuleName = ? AND FType = ?", "RSS", configType)
	if preferNonEmpty {
		query = query.Where("TRIM(COALESCE(FValue, '')) <> ?", "")
	}

	err := query.Order("FID desc").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", 0, nil
	}
	if err != nil {
		return "", 0, err
	}
	return row.FValue, row.FID, nil
}

func GetConfigValuePreferNonEmpty(configType string) (string, error) {
	db, err := getORMDB()
	if err != nil {
		return "", err
	}

	value, fid, err := getConfigValueRow(db, configType, true)
	if err != nil {
		return "", err
	}
	if fid == 0 {
		return getConfigValueWithDB(db, configType)
	}
	return value, nil
}

func SaveConfigValue(configType string, value string) error {
	db, err := getORMDB()
	if err != nil {
		return err
	}
	return saveConfigValue(db, configType, value)
}

func SaveConfigValues(values map[string]string) error {
	db, err := getORMDB()
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for configType, value := range values {
			if err := saveConfigValue(tx, configType, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func saveConfigValue(db *gorm.DB, configType string, value string) error {
	now := time.Now()
	_, fid, err := getConfigValueRow(db, configType, false)
	if err != nil {
		return err
	}
	if fid == 0 {
		item := TbSysConfigs{
			FModuleName: "RSS",
			FType:       configType,
			FValue:      value,
			FCreateDate: &now,
			FVisible:    0,
		}
		return db.Create(&item).Error
	}

	return db.Table((&TbSysConfigs{}).TableName()).
		Where("FID = ?", fid).
		Update("FValue", value).
		Error
}
