package database

import "time"

type TbInfruscanFruitInfo struct {
	FCustomerID          int        `json:"FCustomerID" gorm:"column:FCustomerID;primaryKey;autoIncrement"`
	FCustomerName        string     `json:"FCustomerName" gorm:"column:FCustomerName;size:100"`
	FFarmName            string     `json:"FFarmName" gorm:"column:FFarmName;size:100"`
	FFruitName           string     `json:"FFruitName" gorm:"column:FFruitName;size:100"`
	FDate                *time.Time `json:"FDate" gorm:"column:FDate;type:datetime"`
	FBatchNumber         *int       `json:"FBatchNumber" gorm:"column:FBatchNumber"`
	FDirection           *int       `json:"FDirection" gorm:"column:FDirection"`
	FChannelNum          *int       `json:"FChannelNum" gorm:"column:FChannelNum"`
	FDerivation          *int       `json:"FDerivation" gorm:"column:FDerivation"`
	FFruitSizeRange      string     `json:"FFruitSizeRange" gorm:"column:FFruitSizeRange;size:50"`
	FModeler             string     `json:"FModeler" gorm:"column:FModeler;size:100"`
	FFruitTemperature    string     `json:"FFruitTemperature" gorm:"column:FFruitTemperature;size:100"`
	FImgPath1            string     `json:"FImgPath1" gorm:"column:FImgPath1;size:255"`
	FImgPath2            string     `json:"FImgPath2" gorm:"column:FImgPath2;size:255"`
	FIsBrix              *int8      `json:"FIsBrix" gorm:"column:FIsBrix;type:tinyint"`
	FIsAcidity           *int8      `json:"FIsAcidity" gorm:"column:FIsAcidity;type:tinyint"`
	FIsBrowning          *int8      `json:"FIsBrowning" gorm:"column:FIsBrowning;type:tinyint"`
	FIsHardness          *int8      `json:"FIsHardness" gorm:"column:FIsHardness;type:tinyint"`
	FIsWaterHeartDisease *int8      `json:"FIsWaterHeartDisease" gorm:"column:FIsWaterHeartDisease;type:tinyint"`
	FIsDryMatter         *int8      `json:"FIsDryMatter" gorm:"column:FIsDryMatter;type:tinyint"`
	FIsHollow            *int8      `json:"FIsHollow" gorm:"column:FIsHollow;type:tinyint"`
	FIsUpload            *int8      `json:"FIsUpload" gorm:"column:FIsUpload;type:tinyint"`
	FCreateDate          *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate          *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
}

func (TbInfruscanFruitInfo) TableName() string {
	return "tb_infruscanfruitinfo"
}
