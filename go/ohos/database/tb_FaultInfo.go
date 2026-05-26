package database

import "time"

type TbFaultInfo struct {
	FID        int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FType      int        `json:"FType" gorm:"column:FType;default:0"`
	FCode      string     `json:"FCode" gorm:"column:FCode;size:50"`
	FName      string     `json:"FName" gorm:"column:FName;size:100"`
	FMessage   string     `json:"FMessage" gorm:"column:FMessage;size:255"`
	FStatus    int        `json:"FStatus" gorm:"column:FStatus"`
	FBeginDate *time.Time `json:"FBeginDate" gorm:"column:FBeginDate;type:datetime"`
	FEndDate   *time.Time `json:"FEndDate" gorm:"column:FEndDate;type:datetime"`
}

func (TbFaultInfo) TableName() string {
	return "tb_FaultInfo"
}
