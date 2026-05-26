package database

type TbSeparationEfficiencyInfo struct {
	ID              int    `json:"ID" gorm:"column:ID;primaryKey;autoIncrement"`
	EfficiencyValue string `json:"EfficiencyValue" gorm:"column:EfficiencyValue;size:50"`
	CurrentDate     string `json:"CurrentDate" gorm:"column:CurrentDate;size:30"`
	CurrentTime     string `json:"CurrentTime" gorm:"column:CurrentTime;size:30"`
}

func (TbSeparationEfficiencyInfo) TableName() string {
	return "tb_SeparationEfficiencyInfo"
}
