package database

import "time"

type TbProductInfo struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	SysID       *int       `json:"SysID" gorm:"column:SysID"`
	CodeID      *int       `json:"CodeID" gorm:"column:CodeID"`
	ProductCode string     `json:"ProductCode" gorm:"column:ProductCode;size:50"`
	FeedPortID  *int       `json:"FeedPortID" gorm:"column:FeedPortID"`
	PackID      *int       `json:"PackID" gorm:"column:PackID"`
	GroupID     *int       `json:"GroupID" gorm:"column:GroupID"`
	ProductName string     `json:"ProductName" gorm:"column:ProductName;size:50"`
	SortGrade   string     `json:"SortGrade" gorm:"column:SortGrade;size:2000"`
	TipMachine  *int       `json:"TipMachine" gorm:"column:TipMachine"`
	TipType     *int       `json:"TipType" gorm:"column:TipType"`
	TipPer      *int       `json:"TipPer" gorm:"column:TipPer"`
	Remark      string     `json:"Remark" gorm:"column:Remark;size:300"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
	FStatus     *int       `json:"FStatus" gorm:"column:FStatus;default:0"`
}

func (TbProductInfo) TableName() string {
	return "tb_productinfo"
}
