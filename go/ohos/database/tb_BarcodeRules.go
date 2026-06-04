package database

import "time"

type TbBarcodeRules struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCode       string     `json:"FCode" gorm:"column:FCode;type:longtext"`
	FName       string     `json:"FName" gorm:"column:FName;type:longtext"`
	FType       *int       `json:"FType" gorm:"column:FType"`
	FEntryNo    *int       `json:"FEntryNo" gorm:"column:FEntryNo"`
	FStartIndex *int       `json:"FStartIndex" gorm:"column:FStartIndex"`
	FLength     *int       `json:"FLength" gorm:"column:FLength"`
	FPadding    string     `json:"FPadding" gorm:"column:FPadding;type:longtext"`
}

func (TbBarcodeRules) TableName() string {
	return "tb_barcoderules"
}
