package database

import "time"

type TbBaseFault struct {
	FID                    int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate            *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate            *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FType                  *int       `json:"FType" gorm:"column:FType"`
	FCode                  string     `json:"FCode" gorm:"column:FCode;type:longtext"`
	FName                  string     `json:"FName" gorm:"column:FName;type:longtext"`
	FGroupName             string     `json:"FGroupName" gorm:"column:FGroupName;type:longtext"`
	FMessage               string     `json:"FMessage" gorm:"column:FMessage;type:longtext"`
	FShutdownCondition     *int       `json:"FShutdownCondition" gorm:"column:FShutdownCondition"`
	FDecelerationCondition *int       `json:"FDecelerationCondition" gorm:"column:FDecelerationCondition"`
	FPopupCondition        *int       `json:"FPopupCondition" gorm:"column:FPopupCondition"`
	FPromptCondition       *int       `json:"FPromptCondition" gorm:"column:FPromptCondition"`
	FExitID                *int       `json:"FExitID" gorm:"column:FExitID"`
	FDuration              *int       `json:"FDuration" gorm:"column:FDuration"`
	FBitStart              *int       `json:"FBitStart" gorm:"column:FBitStart;default:0"`
	FBitType               *int       `json:"FBitType" gorm:"column:FBitType;default:0"`
	FPartIdx               *int       `json:"FPartIdx" gorm:"column:FPartIdx;default:0"`
	FStatus                int        `json:"FStatus" gorm:"-"`
	FOccurDate             *time.Time `json:"FOccurDate" gorm:"-"`
}

func (TbBaseFault) TableName() string {
	return "tb_Base_Fault"
}
