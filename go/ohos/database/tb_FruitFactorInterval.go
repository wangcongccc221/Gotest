package database

import "time"

type TbFruitFactorInterval struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate  *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate  *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCustomerID  *int       `json:"FCustomerID" gorm:"column:FCustomerID"`
	FFactorType  *int       `json:"FFactorType" gorm:"column:FFactorType"`
	FFactorID    *int       `json:"FFactorID" gorm:"column:FFactorID"`
	FFactorMin   *float64   `json:"FFactorMin" gorm:"column:FFactorMin;type:double"`
	FFactorMax   *float64   `json:"FFactorMax" gorm:"column:FFactorMax;type:double"`
	FFruitNumber *int       `json:"FFruitNumber" gorm:"column:FFruitNumber"`
}

func (TbFruitFactorInterval) TableName() string {
	return "tb_fruitfactorinterval"
}
