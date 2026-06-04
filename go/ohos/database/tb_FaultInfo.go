package database

import "time"

type TbFaultInfo struct {
	FID        int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FType      *int       `json:"FType" gorm:"column:FType"`
	FCode      string     `json:"FCode" gorm:"column:FCode;type:longtext"`
	FName      string     `json:"FName" gorm:"column:FName;type:longtext"`
	FMessage   string     `json:"FMessage" gorm:"column:FMessage;type:longtext"`
	FPartIdx   *int       `json:"FPartIdx" gorm:"column:FPartIdx;default:0"`
	FStatus    *int       `json:"FStatus" gorm:"column:FStatus"`
	FBeginDate *time.Time `json:"FBeginDate" gorm:"column:FBeginDate;type:datetime(6)"`
	FEndDate   *time.Time `json:"FEndDate" gorm:"column:FEndDate;type:datetime(6)"`
}

func (TbFaultInfo) TableName() string {
	return "tb_faultinfo"
}
