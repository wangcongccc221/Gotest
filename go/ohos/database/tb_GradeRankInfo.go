package database

import "time"

type TbGradeRankInfo struct {
	FID              int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate      *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate      *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FSysID           int        `json:"FSysID" gorm:"column:FSysID;not null"`
	FDeviceID        string     `json:"FDeviceID" gorm:"column:FDeviceID;type:longtext;not null"`
	FType            string     `json:"FType" gorm:"column:FType;type:longtext;not null"`
	FGradeType       string     `json:"FGradeType" gorm:"column:FGradeType;type:longtext;not null"`
	FGradeDetailType string     `json:"FGradeDetailType" gorm:"column:FGradeDetailType;type:longtext;not null"`
	FStartDate       string     `json:"FStartDate" gorm:"column:FStartDate;type:longtext;not null"`
	FEndDate         string     `json:"FEndDate" gorm:"column:FEndDate;type:longtext;not null"`
}

func (TbGradeRankInfo) TableName() string {
	return "tb_graderankinfo"
}
