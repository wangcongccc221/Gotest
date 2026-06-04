package database

import "time"

type TbExportConfigs struct {
	FID              int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	ExitID           int        `json:"ExitID" gorm:"column:ExitID;not null"`
	ExitName         string     `json:"ExitName" gorm:"column:ExitName;size:255"`
	ProductID        *int       `json:"ProductID" gorm:"column:ProductID;default:-1"`
	GroupID          *int       `json:"GroupID" gorm:"column:GroupID;default:-1"`
	SpecID           *int       `json:"SpecID" gorm:"column:SpecID"`
	ExitType         *int       `json:"ExitType" gorm:"column:ExitType"`
	ExitDisplayName  string     `json:"ExitDisplayName" gorm:"column:ExitDisplayName;size:255"`
	ExitTypeName     string     `json:"ExitTypeName" gorm:"column:ExitTypeName;size:50"`
	SpecName         string     `json:"SpecName" gorm:"column:SpecName;size:50"`
	SpecWeight       *int       `json:"SpecWeight" gorm:"column:SpecWeight"`
	FAfterMaterialID string     `json:"FAfterMaterialID" gorm:"column:FAfterMaterialID;size:50"`
	FCreateDate      *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate      *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
}

func (TbExportConfigs) TableName() string {
	return "tb_exportconfigs"
}
