package database

import "time"

type TbConfigs struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FModuleName string     `json:"FModuleName" gorm:"column:FModuleName;type:longtext"`
	FName       string     `json:"FName" gorm:"column:FName;type:longtext"`
	FFileData   []byte     `json:"FFileData" gorm:"column:FFileData;type:longblob;not null"`
}

func (TbConfigs) TableName() string {
	return "tb_configs"
}
