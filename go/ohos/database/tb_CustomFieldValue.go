package database

import "time"

type TbCustomFieldValue struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FModule     string     `json:"FModule" gorm:"column:FModule;type:longtext"`
	FFieldID    *int       `json:"FFieldID" gorm:"column:FFieldID"`
	FFunctionID *int       `json:"FFunctionID" gorm:"column:FFunctionID"`
	FValue      string     `json:"FValue" gorm:"column:FValue;type:longtext"`
}

func (TbCustomFieldValue) TableName() string {
	return "tb_customfieldvalue"
}
