package database

import "time"

type TbClientInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCode       string     `json:"FCode" gorm:"column:FCode;size:50"`
	FName       string     `json:"FName" gorm:"column:FName;size:100"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
	FIsCurrent  bool       `json:"FIsCurrent" gorm:"column:FIsCurrent"`
}

func (TbClientInfo) TableName() string {
	return "tb_ClientInfo"
}
