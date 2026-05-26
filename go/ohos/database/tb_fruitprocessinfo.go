package database

import "fmt"

type TbFruitProcessInfoPercent struct {
	FID                         int    `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	CustomerID                  int    `json:"CustomerID" gorm:"column:CustomerID"`
	RealWeightCountPercent      []byte `json:"RealWeightCountPercent" gorm:"column:RealWeightCountPercent;type:blob"`
	SeparationEfficiencyPercent []byte `json:"SeparationEfficiencyPercent" gorm:"column:SeparationEfficiencyPercent;type:blob"`
}

func (TbFruitProcessInfoPercent) TableName() string {
	return "tb_fruitprocessinfo"
}

type TbFruitProcessInfo struct {
	FID                  int     `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	RealWeightCount      float64 `json:"RealWeightCount" gorm:"column:RealWeightCount;type:float"`
	RealWeightCountPer   float64 `json:"RealWeightCountPer" gorm:"column:RealWeightCountPer;type:float"`
	SeparationEfficiency float64 `json:"SeparationEfficiency" gorm:"column:SeparationEfficiency;type:float"`
	SpeedPercent         float64 `json:"SpeedPercent" gorm:"column:SpeedPercent;type:float"`
	AvgWeight            float64 `json:"AvgWeight" gorm:"column:AvgWeight;type:float"`
	RunningDate          string  `json:"RunningDate" gorm:"column:RunningDate;size:20"`
}

func fruitProcessInfoTableName(year int) string {
	return fmt.Sprintf("tb_fruitprocessinfo_%d", year)
}
