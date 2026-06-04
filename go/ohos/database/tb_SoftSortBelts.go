package database

import "time"

type TbSoftSortBelts struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FBinTipID   string     `json:"FBinTipID" gorm:"column:FBinTipID;type:longtext"`
	FNumber     *int       `json:"FNumber" gorm:"column:FNumber;default:0"`
	FName       string     `json:"FName" gorm:"column:FName;size:100"`
}

func (TbSoftSortBelts) TableName() string {
	return "tb_softsortbelts"
}
