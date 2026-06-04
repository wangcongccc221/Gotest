package database

import "time"

type TbPalletInfo struct {
	FID            int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate    *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate    *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	PalletID       string     `json:"PalletID" gorm:"column:PalletID;size:100"`
	SortExitID     *int       `json:"SortExitID" gorm:"column:SortExitID"`
	PalletCacheSet *int       `json:"PalletCacheSet" gorm:"column:PalletCacheSet"`
	PalletSumNum   *int       `json:"PalletSumNum" gorm:"column:PalletSumNum"`
	StorageName    string     `json:"StorageName" gorm:"column:StorageName;size:100"`
	PalletState    *int       `json:"PalletState" gorm:"column:PalletState"`
	FSubSystem     *int       `json:"FSubSystem" gorm:"column:FSubSystem"`
}

func (TbPalletInfo) TableName() string {
	return "tb_palletinfo"
}
