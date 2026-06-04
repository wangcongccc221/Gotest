package database

import "time"

type TbProductBatchInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	SysID       *int       `json:"SysID" gorm:"column:SysID"`
	CustomerID  *int       `json:"CustomerID" gorm:"column:CustomerID;index:CustomerID"`
	FeedPortID  *int       `json:"FeedPortID" gorm:"column:FeedPortID"`
	CodeID      *int       `json:"CodeID" gorm:"column:CodeID;index:CodeID"`
	ProductName string     `json:"ProductName" gorm:"column:ProductName;type:longtext"`
	ProductCode string     `json:"ProductCode" gorm:"column:ProductCode;type:longtext"`
	SortGrade   string     `json:"SortGrade" gorm:"column:SortGrade;type:longtext"`
	PackName    string     `json:"PackName" gorm:"column:PackName;type:longtext;not null"`
	PackWeight  *int       `json:"PackWeight" gorm:"column:PackWeight"`
	TipMachine  *int       `json:"TipMachine" gorm:"column:TipMachine"`
	Remark      string     `json:"Remark" gorm:"column:Remark;type:longtext"`
}

func (TbProductBatchInfo) TableName() string {
	return "tb_productbatchinfo"
}
