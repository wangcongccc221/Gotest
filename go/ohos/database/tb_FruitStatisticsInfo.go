package database

import "time"

type TbFruitStatisticsInfo struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate  *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate  *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCustomerID  *int       `json:"FCustomerID" gorm:"column:FCustomerID"`
	FSysID       *int       `json:"FSysID" gorm:"column:FSysID"`
	FChannelID   *int       `json:"FChannelID" gorm:"column:FChannelID"`
	FExitID      *int       `json:"FExitID" gorm:"column:FExitID"`
	FProductID   *int       `json:"FProductID" gorm:"column:FProductID"`
	FGradeID     *uint      `json:"FGradeID" gorm:"column:FGradeID"`
	FFruitNumber *uint64    `json:"FFruitNumber" gorm:"column:FFruitNumber;type:bigint unsigned"`
	FFruitWeight *float64   `json:"FFruitWeight" gorm:"column:FFruitWeight;type:double"`
	FState       int        `json:"FState" gorm:"column:FState;default:0"`
}

func (TbFruitStatisticsInfo) TableName() string {
	return "tb_fruitstatisticsinfo"
}
