package database

import "fmt"

type TbFruitProcessInfoPercent struct {
	FID                  int      `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	SysID                *int     `json:"SysID" gorm:"column:SysID"`
	ChannelID            *int     `json:"ChannelID" gorm:"column:ChannelID"`
	RealWeightCount      *float64 `json:"RealWeightCount" gorm:"column:RealWeightCount;type:double"`
	RealWeightCountPer   *float64 `json:"RealWeightCountPer" gorm:"column:RealWeightCountPer;type:double"`
	SeparationEfficiency *float64 `json:"SeparationEfficiency" gorm:"column:SeparationEfficiency;type:double"`
	SpeedPercent         *float64 `json:"SpeedPercent" gorm:"column:SpeedPercent;type:double"`
	AvgWeight            *float64 `json:"AvgWeight" gorm:"column:AvgWeight;type:double"`
	RunningDate          string   `json:"RunningDate" gorm:"column:RunningDate;type:longtext"`
}

func (TbFruitProcessInfoPercent) TableName() string {
	return "tb_fruitprocessinfo"
}

type TbFruitProcessInfo struct {
	FID                  int      `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	SysID                *int     `json:"SysID" gorm:"column:SysID"`
	ChannelID            *int     `json:"ChannelID" gorm:"column:ChannelID"`
	RealWeightCount      *float64 `json:"RealWeightCount" gorm:"column:RealWeightCount;type:double"`
	RealWeightCountPer   *float64 `json:"RealWeightCountPer" gorm:"column:RealWeightCountPer;type:double"`
	SeparationEfficiency *float64 `json:"SeparationEfficiency" gorm:"column:SeparationEfficiency;type:double"`
	SpeedPercent         *float64 `json:"SpeedPercent" gorm:"column:SpeedPercent;type:double"`
	AvgWeight            *float64 `json:"AvgWeight" gorm:"column:AvgWeight;type:double"`
	RunningDate          string   `json:"RunningDate" gorm:"column:RunningDate;type:longtext"`
}

func fruitProcessInfoTableName(year int) string {
	return fmt.Sprintf("tb_fruitprocessinfo_%d", year)
}
