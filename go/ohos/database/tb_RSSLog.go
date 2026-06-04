package database

import "time"

type TbRSSLog struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FLevel      string     `json:"FLevel" gorm:"column:FLevel;type:longtext;not null"`
	FType       string     `json:"FType" gorm:"column:FType;type:longtext;not null"`
	FUserID     *int       `json:"FUserID" gorm:"column:FUserID"`
	FMessage    string     `json:"FMessage" gorm:"column:FMessage;type:longtext;not null"`
}

func (TbRSSLog) TableName() string {
	return "tb_rss_log"
}
