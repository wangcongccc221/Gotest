package database

import "time"

type TbSortExitInfo struct {
	FID            int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate    *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate    *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FSpecID        *int       `json:"FSpecID" gorm:"column:FSpecID"`
	FSpecName      string     `json:"FSpecName" gorm:"column:FSpecName;size:100"`
	SortExitID     *int       `json:"SortExitID" gorm:"column:SortExitID"`
	CurrentCount   *int       `json:"CurrentCount" gorm:"column:CurrentCount"`
	PalletQuantity *int       `json:"PalletQuantity" gorm:"column:PalletQuantity"`
	FStatus        *int       `json:"FStatus" gorm:"column:FStatus"`
	IsValid        *int       `json:"IsValid" gorm:"column:IsValid"`
	FSubSystem     *int       `json:"FSubSystem" gorm:"column:FSubSystem"`
}

func (TbSortExitInfo) TableName() string {
	return "tb_sortexitinfo"
}
