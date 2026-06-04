package database

import "time"

type TbEmptyBoxInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FNumber     *int       `json:"FNumber" gorm:"column:FNumber"`
	FName       string     `json:"FName" gorm:"column:FName;type:longtext"`
	FCode       string     `json:"FCode" gorm:"column:FCode;type:longtext"`
}

func (TbEmptyBoxInfo) TableName() string {
	return "tb_emptyboxinfo"
}
