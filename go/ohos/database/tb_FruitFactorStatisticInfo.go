package database

import "time"

type TbFruitFactorStatisticInfo struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate  *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate  *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCustomerID  *int       `json:"FCustomerID" gorm:"column:FCustomerID"`
	FSysID       *int       `json:"FSysID" gorm:"column:FSysID"`
	FChannelID   *int       `json:"FChannelID" gorm:"column:FChannelID"`
	FFactor      *int       `json:"FFactor" gorm:"column:FFactor"`
	FFactorValue *float64   `json:"FFactorValue" gorm:"column:FFactorValue;type:double"`
	FFruitNumber *int       `json:"FFruitNumber" gorm:"column:FFruitNumber"`
}

func (TbFruitFactorStatisticInfo) TableName() string {
	return "tb_fruitfactor_statisticinfo"
}
