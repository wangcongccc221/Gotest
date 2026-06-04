package database

import "time"

type TbParamDetail struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FItemID     string     `json:"FItemID" gorm:"column:FItemID;size:255;not null"`
	FVariable   string     `json:"FVariable" gorm:"column:FVariable;size:255;not null"`
	FParam      string     `json:"FParam" gorm:"column:FParam;size:255;not null"`
	FNo         *int       `json:"FNo" gorm:"column:FNo"`
	FNote       string     `json:"FNote" gorm:"column:FNote;size:8000;not null"`
	FVersion    *int       `json:"FVersion" gorm:"column:FVersion"`
	FState      *int       `json:"FState" gorm:"column:FState"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
}

func (TbParamDetail) TableName() string {
	return "tb_param_detail"
}
