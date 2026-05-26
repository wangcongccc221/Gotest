package database

import "time"

type TbFileInfo struct {
	FileName      string     `json:"FileName" gorm:"column:FileName;size:255"`
	Url           string     `json:"Url" gorm:"column:Url;size:500"`
	DirectoryName string     `json:"DirectoryName" gorm:"column:DirectoryName;size:255"`
	LastWriteTime *time.Time `json:"LastWriteTime" gorm:"column:LastWriteTime;type:datetime"`
}

func (TbFileInfo) TableName() string {
	return "tb_FileInfo"
}
