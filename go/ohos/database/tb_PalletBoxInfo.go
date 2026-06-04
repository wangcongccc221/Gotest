package database

import "time"

type TbPalletBoxInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6);index:Index_tb_PalletBoxInfo,priority:2"`
	FExitID     *int       `json:"FExitID" gorm:"column:FExitID;index:Index_tb_PalletBoxInfo,priority:1"`
	FBoxNumber  *int       `json:"FBoxNumber" gorm:"column:FBoxNumber"`
}

func (TbPalletBoxInfo) TableName() string {
	return "tb_palletboxinfo"
}
