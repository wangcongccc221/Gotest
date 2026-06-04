package database

import "time"

type TbOutletBoxDetailInfo struct {
	FID           int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate   *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate   *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FOutletBoxID  *int       `json:"FOutletBoxID" gorm:"column:FOutletBoxID"`
	FProduct      string     `json:"FProduct" gorm:"column:FProduct;type:longtext"`
	FWeightOrSize string     `json:"FWeightOrSize" gorm:"column:FWeightOrSize;type:longtext"`
	FQuality      string     `json:"FQuality" gorm:"column:FQuality;type:longtext"`
	FPackNum      *uint16    `json:"FPackNum" gorm:"column:FPackNum;type:smallint unsigned"`
	FPackWeight   *float64   `json:"FPackWeight" gorm:"column:FPackWeight;type:double"`
	FPackArea     *uint      `json:"FPackArea" gorm:"column:FPackArea;type:int unsigned"`
	FPackVol      *uint      `json:"FPackVol" gorm:"column:FPackVol;type:int unsigned"`
}

func (TbOutletBoxDetailInfo) TableName() string {
	return "tb_outletboxdetailinfo"
}
