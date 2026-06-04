package database

import "time"

type TbBinTipInfo struct {
	FID         string     `json:"FID" gorm:"column:FID;size:255;primaryKey"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FName       string     `json:"FName" gorm:"column:FName;type:longtext"`
}

func (TbBinTipInfo) TableName() string {
	return "tb_bintipinfo"
}
