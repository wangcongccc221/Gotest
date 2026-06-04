package database

import "time"

type TbCustomFieldSource struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FSourceName string     `json:"FSourceName" gorm:"column:FSourceName;type:longtext;not null"`
	FParentID   int        `json:"FParentID" gorm:"column:FParentID;not null"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
}

func (TbCustomFieldSource) TableName() string {
	return "tb_customfieldsource"
}
