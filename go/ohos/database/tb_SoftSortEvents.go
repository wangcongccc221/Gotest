package database

import "time"

type TbSoftSortEvents struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FOrderID    *int       `json:"FOrderID" gorm:"column:FOrderID"`
	FBeltID     *int       `json:"FBeltID" gorm:"column:FBeltID"`
	FStartTime  *time.Time `json:"FStartTime" gorm:"column:FStartTime;type:datetime(6)"`
	FEndTime    *time.Time `json:"FEndTime" gorm:"column:FEndTime;type:datetime(6)"`
}

func (TbSoftSortEvents) TableName() string {
	return "tb_softsortevents"
}
