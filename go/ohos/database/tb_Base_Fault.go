package database

import "time"

type TbBaseFault struct {
	FID                    int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FType                  int        `json:"FType" gorm:"column:FType;default:0"`
	FCode                  string     `json:"FCode" gorm:"column:FCode;size:50"`
	FName                  string     `json:"FName" gorm:"column:FName;size:100"`
	FGroupName             string     `json:"FGroupName" gorm:"column:FGroupName;size:100"`
	FMessage               string     `json:"FMessage" gorm:"column:FMessage;size:255"`
	FShutdownCondition     int        `json:"FShutdownCondition" gorm:"column:FShutdownCondition;default:0"`
	FDecelerationCondition int        `json:"FDecelerationCondition" gorm:"column:FDecelerationCondition;default:0"`
	FPopupCondition        int        `json:"FPopupCondition" gorm:"column:FPopupCondition;default:0"`
	FPromptCondition       int        `json:"FPromptCondition" gorm:"column:FPromptCondition;default:0"`
	FExitID                int        `json:"FExitID" gorm:"column:FExitID;default:0"`
	FDuration              int        `json:"FDuration" gorm:"column:FDuration;default:0"`
	FCreateDate            *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate            *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
	FStatus                int        `json:"FStatus" gorm:"column:FStatus;default:0"`
	FOccurDate             *time.Time `json:"FOccurDate" gorm:"column:FOccurDate;type:datetime"`
}

func (TbBaseFault) TableName() string {
	return "tb_Base_Fault"
}
