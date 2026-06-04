package database

import "time"

type TbOrderInfo struct {
	FID             int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate     *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate     *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
	FBatchNo        string     `json:"FBatchNo" gorm:"column:FBatchNo;type:longtext"`
	FCustomerCode   string     `json:"FCustomerCode" gorm:"column:FCustomerCode;type:longtext"`
	FFruitName      string     `json:"FFruitName" gorm:"column:FFruitName;type:longtext"`
	FTotalWeight    *float64   `json:"FTotalWeight" gorm:"column:FTotalWeight;type:double"`
	FTotalBoxNum    *int       `json:"FTotalBoxNum" gorm:"column:FTotalBoxNum"`
	FPlanWeight     *float64   `json:"FPlanWeight" gorm:"column:FPlanWeight;type:double;default:0"`
	FPlanBoxNum     *int       `json:"FPlanBoxNum" gorm:"column:FPlanBoxNum;default:0"`
	FTareWeight     *float64   `json:"FTareWeight" gorm:"column:FTareWeight;type:double;default:0"`
	FTareBoxNum     *int       `json:"FTareBoxNum" gorm:"column:FTareBoxNum;default:0"`
	FPicker         string     `json:"FPicker" gorm:"column:FPicker;type:longtext"`
	FPickDate       time.Time  `json:"FPickDate" gorm:"column:FPickDate;type:datetime(6);not null"`
	FStock          string     `json:"FStock" gorm:"column:FStock;type:longtext"`
	FStartTime      string     `json:"FStartTime" gorm:"column:FStartTime;size:50"`
	FEndTime        string     `json:"FEndTime" gorm:"column:FEndTime;size:50"`
	FCompletedState *int       `json:"FCompletedState" gorm:"column:FCompletedState"`
	FRemark         string     `json:"FRemark" gorm:"column:FRemark;type:longtext"`
}

func (TbOrderInfo) TableName() string {
	return "tb_orderinfo"
}
