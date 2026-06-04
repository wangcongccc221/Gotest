package database

import "time"

type TbSortExitSumInfo struct {
	FID                    int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate            *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate            *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	SortExitID             int        `json:"SortExitID" gorm:"column:SortExitID;not null"`
	SortExitName           string     `json:"SortExitName" gorm:"column:SortExitName;size:100"`
	SortExitState          *int       `json:"SortExitState" gorm:"column:SortExitState"`
	PalletID               string     `json:"PalletID" gorm:"column:PalletID;type:longtext"`
	SortExitSingleMaxCache int        `json:"SortExitSingleMaxCache" gorm:"column:SortExitSingleMaxCache;not null"`
	SumCurrentCount        *int       `json:"SumCurrentCount" gorm:"column:SumCurrentCount"`
	PalletSingleMaxCache   int        `json:"PalletSingleMaxCache" gorm:"column:PalletSingleMaxCache;not null"`
	SumPalletQuantity      *int       `json:"SumPalletQuantity" gorm:"column:SumPalletQuantity"`
	FSubSystem             *int       `json:"FSubSystem" gorm:"column:FSubSystem"`
}

func (TbSortExitSumInfo) TableName() string {
	return "tb_sortexitsuminfo"
}
