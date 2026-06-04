package database

import "time"

type TbGradeRankDetail struct {
	FID               int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate       *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate       *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FGradeTypeID      int        `json:"FGradeTypeID" gorm:"column:FGradeTypeID;not null"`
	FDeviceID         string     `json:"FDeviceID" gorm:"column:FDeviceID;type:longtext;not null"`
	FWeightOrSizeName string     `json:"FWeightOrSizeName" gorm:"column:FWeightOrSizeName;type:longtext;not null"`
	FQualityName      string     `json:"FQualityName" gorm:"column:FQualityName;type:longtext;not null"`
}

func (TbGradeRankDetail) TableName() string {
	return "tb_graderankdetail"
}
