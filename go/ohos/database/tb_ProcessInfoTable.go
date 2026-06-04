package database

type TbProcessInfoTable struct {
	FID                  int      `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	SysID                *int     `json:"SysID" gorm:"column:SysID"`
	ChannelID            *int     `json:"ChannelID" gorm:"column:ChannelID"`
	RealWeightCount      *float64 `json:"RealWeightCount" gorm:"column:RealWeightCount;type:double"`
	SeparationEfficiency *float64 `json:"SeparationEfficiency" gorm:"column:SeparationEfficiency;type:double"`
	AvgWeight            *float64 `json:"AvgWeight" gorm:"column:AvgWeight;type:double"`
	RealWeightCountPer   *float64 `json:"RealWeightCountPer" gorm:"column:RealWeightCountPer;type:double"`
	SpeedPercent         *float64 `json:"SpeedPercent" gorm:"column:SpeedPercent;type:double"`
	RunningDate          string   `json:"RunningDate" gorm:"column:RunningDate;size:20;not null"`
}

func (TbProcessInfoTable) TableName() string {
	return "tb_processinfotable"
}
