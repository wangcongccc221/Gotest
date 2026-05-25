package database

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type TbSysConfigs struct {
	FID              int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FModuleName      string     `json:"FModuleName" gorm:"column:FModuleName;size:50;not null;default:RSS"`
	FType            string     `json:"FType" gorm:"column:FType;size:200"`
	FValue           string     `json:"FValue" gorm:"column:FValue;type:text"`
	FCreateDate      *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FVisible         int        `json:"FVisible" gorm:"column:FVisible;not null;default:0"`
	FEnType          string     `json:"FEnType" gorm:"column:FEnType;size:200"`
	FValueType       int        `json:"FValueType" gorm:"column:FValueType;type:varchar(200);default:0"`
	FValueTypeDetail string     `json:"FValueTypeDetail" gorm:"column:FValueTypeDetail;size:200"`
	FZhType          string     `json:"FZhType" gorm:"column:FZhType;size:200"`
}

func (TbSysConfigs) TableName() string {
	return "tb_Sys_Configs"
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
