package database

import "time"

type TbEfficiencyInfo struct {
	FID                 int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate         *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate         *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FModuleName         string     `json:"FModuleName" gorm:"column:FModuleName;type:longtext"`
	FEfficiencyValue    int        `json:"FEfficiencyValue" gorm:"column:FEfficiencyValue;not null"`
	FEfficiencyTimeSpan time.Time  `json:"FEfficiencyTimeSpan" gorm:"column:FEfficiencyTimeSpan;type:datetime(6);not null"`
}

func (TbEfficiencyInfo) TableName() string {
	return "tb_efficiencyinfo"
}
