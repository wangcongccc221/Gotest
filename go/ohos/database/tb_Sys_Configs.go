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

	var item TbSysConfigs
	err = db.Where("FModuleName = ? AND FType = ?", "RSS", configType).Order("FID desc").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return item.FValue, nil
}

func GetConfigValuePreferNonEmpty(configType string) (string, error) {
	db, err := getORMDB()
	if err != nil {
		return "", err
	}

	var item TbSysConfigs
	err = db.Where("FModuleName = ? AND FType = ? AND TRIM(COALESCE(FValue, '')) <> ?", "RSS", configType, "").Order("FID desc").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return GetConfigValue(configType)
	}
	if err != nil {
		return "", err
	}
	return item.FValue, nil
}

func SaveConfigValue(configType string, value string) error {
	db, err := getORMDB()
	if err != nil {
		return err
	}

	now := time.Now()
	var item TbSysConfigs
	err = db.Where("FModuleName = ? AND FType = ?", "RSS", configType).Order("FID desc").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		item = TbSysConfigs{
			FModuleName: "RSS",
			FType:       configType,
			FValue:      value,
			FCreateDate: &now,
			FVisible:    0,
		}
		return db.Create(&item).Error
	}
	if err != nil {
		return err
	}

	item.FValue = value
	return db.Save(&item).Error
}
