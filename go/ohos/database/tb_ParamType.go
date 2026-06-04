package database

import "time"

type TbParamType struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FModuleName  string     `json:"FModuleName" gorm:"column:FModuleName;size:50;not null"`
	FCategory    string     `json:"FCategory" gorm:"column:FCategory;size:100;not null"`
	FSubCategory string     `json:"FSubCategory" gorm:"column:FSubCategory;size:100;not null"`
	FItemID      string     `json:"FItemID" gorm:"column:FItemID;size:255;not null"`
	FState       *int       `json:"FState" gorm:"column:FState"`
	FVersion     *int       `json:"FVersion" gorm:"column:FVersion"`
	FCreateDate  *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate  *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
}

func (TbParamType) TableName() string {
	return "tb_param_type"
}
