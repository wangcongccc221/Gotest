package database

import "time"

type TbCustomer struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FName       string     `json:"FName" gorm:"column:FName;size:30"`
	FPassWord   string     `json:"FPassWord" gorm:"column:FPassWord;size:30"`
	FCode       string     `json:"FCode" gorm:"column:FCode;size:30"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FIsUse      string     `json:"FIsUse" gorm:"column:FIsUse;size:1"`
}

func (TbCustomer) TableName() string {
	return "tb_Customer"
}
