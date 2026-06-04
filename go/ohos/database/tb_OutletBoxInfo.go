package database

import "time"

type TbOutletBoxInfo struct {
	FID           int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate   *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate   *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FCustomerID   *int       `json:"FCustomerID" gorm:"column:FCustomerID;index:FCustomerID"`
	FOutlet       *int       `json:"FOutlet" gorm:"column:FOutlet;index:FOutlet"`
	FBoxID        *uint      `json:"FBoxID" gorm:"column:FBoxID;type:int unsigned;index:FBoxID"`
	FBinID        *int       `json:"FBinID" gorm:"column:FBinID"`
	FPackNum      *uint16    `json:"FPackNum" gorm:"column:FPackNum;type:smallint unsigned"`
	FPackWeight   *float64   `json:"FPackWeight" gorm:"column:FPackWeight;type:double"`
	FPackArea     *uint      `json:"FPackArea" gorm:"column:FPackArea;type:int unsigned"`
	FPackVol      *uint      `json:"FPackVol" gorm:"column:FPackVol;type:int unsigned"`
	FStatus       *int       `json:"FStatus" gorm:"column:FStatus;index:FStatus"`
	FUploadStatus *int       `json:"FUploadStatus" gorm:"column:FUploadStatus"`
	FPM           *int       `json:"FPM" gorm:"column:FPM;default:0"`
}

func (TbOutletBoxInfo) TableName() string {
	return "tb_outletboxinfo"
}
