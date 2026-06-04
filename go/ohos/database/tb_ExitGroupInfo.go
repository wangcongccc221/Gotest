package database

import "time"

type TbExitGroupInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	CodeID      int        `json:"CodeID" gorm:"column:CodeID;default:0"`
	GroupName   string     `json:"GroupName" gorm:"column:GroupName;size:50"`
	GroupColor  string     `json:"GroupColor" gorm:"column:GroupColor;size:50"`
	PackID      *int       `json:"PackID" gorm:"column:PackID"`
	GroupExit   *int64     `json:"GroupExit" gorm:"column:GroupExit;type:bigint"`
	FStatus     int        `json:"FStatus" gorm:"column:FStatus;default:0"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
}

func (TbExitGroupInfo) TableName() string {
	return "tb_exitgroupinfo"
}
