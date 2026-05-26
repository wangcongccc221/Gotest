package database

import "time"

type TbSqlInfo struct {
	FID          int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FName        string     `json:"FName" gorm:"column:FName;size:50;uniqueIndex:Index_FName"`
	FIsExecute   bool       `json:"FIsExecute" gorm:"column:FIsExecute;type:bit(1)"`
	FExecuteDate *time.Time `json:"FExecuteDate" gorm:"column:FExecuteDate;type:datetime"`
	FIsInit      bool       `json:"FIsInit" gorm:"column:FIsInit;type:bit(1)"`
	FInitDate    *time.Time `json:"FInitDate" gorm:"column:FInitDate;type:datetime"`
}

func (TbSqlInfo) TableName() string {
	return "tb_SqlInfo"
}
