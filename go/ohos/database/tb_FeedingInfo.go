package database

import "time"

type TbFeedingInfo struct {
	FID             int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate     *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate     *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FOrderID        *int       `json:"FOrderID" gorm:"column:FOrderID"`
	FFeedID         *int       `json:"FFeedID" gorm:"column:FFeedID"`
	FCustomerCode   string     `json:"FCustomerCode" gorm:"column:FCustomerCode;type:longtext"`
	FBarcode        string     `json:"FBarcode" gorm:"column:FBarcode;type:longtext"`
	FBoxNum         *int       `json:"FBoxNum" gorm:"column:FBoxNum;default:0"`
	FWeight         *float64   `json:"FWeight" gorm:"column:FWeight;type:double;default:0"`
	FWeightTime     *time.Time `json:"FWeightTime" gorm:"column:FWeightTime;type:datetime"`
	FTraBoxNum      *int       `json:"FTraBoxNum" gorm:"column:FTraBoxNum;default:0"`
	FTareWeight     *float64   `json:"FTareWeight" gorm:"column:FTareWeight;type:double;default:0"`
	FTareWeightTime *time.Time `json:"FTareWeightTime" gorm:"column:FTareWeightTime;type:datetime"`
	FState          *int       `json:"FState" gorm:"column:FState;default:0"`
}

func (TbFeedingInfo) TableName() string {
	return "tb_feedinginfo"
}
