package database

import "time"

type TbFruitWeightFactorStatisticInfo struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate  *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate  *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCustomerID  *int       `json:"FCustomerID" gorm:"column:FCustomerID"`
	FSysID       *int       `json:"FSysID" gorm:"column:FSysID"`
	FChannelID   *int       `json:"FChannelID" gorm:"column:FChannelID"`
	FWeightValue *int       `json:"FWeightValue" gorm:"column:FWeightValue"`
	FYFactor     *int       `json:"FYFactor" gorm:"column:FYFactor"`
	FYValue      *float64   `json:"FYValue" gorm:"column:FYValue;type:double"`
	FYIndex      *int       `json:"FYIndex" gorm:"column:FYIndex"`
	FFruitNumber *int       `json:"FFruitNumber" gorm:"column:FFruitNumber"`
}

func (TbFruitWeightFactorStatisticInfo) TableName() string {
	return "tb_fruitweightfactor_statisticinfo"
}
