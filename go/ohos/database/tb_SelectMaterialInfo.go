package database

import "time"

type TbSelectMaterialInfo struct {
	FID               int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate       *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate       *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FOrderID          *int       `json:"FOrderID" gorm:"column:FOrderID"`
	FFunctionID       *int       `json:"FFunctionID" gorm:"column:FFunctionID"`
	FBoxWeight        *int       `json:"FBoxWeight" gorm:"column:FBoxWeight"`
	FCurrentBoxWeight *int       `json:"FCurrentBoxWeight" gorm:"column:FCurrentBoxWeight"`
	FBoxNum           *int       `json:"FBoxNum" gorm:"column:FBoxNum"`
	FGradeName        string     `json:"FGradeName" gorm:"column:FGradeName;type:longtext"`
}

func (TbSelectMaterialInfo) TableName() string {
	return "tb_selectmaterialinfo"
}
