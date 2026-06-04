package database

import "time"

type TbPackingSpec struct {
	FID         int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	CodeID      int        `json:"CodeID" gorm:"column:CodeID;not null;default:0"`
	FSpec       string     `json:"FSpec" gorm:"column:FSpec;size:100"`
	FWeight     *int       `json:"FWeight" gorm:"column:FWeight;default:0"`
	FPackType   *int       `json:"FPackType" gorm:"column:FPackType;default:0"`
	PackUnit    *int       `json:"PackUnit" gorm:"column:PackUnit;default:0"`
	ShareType   *int       `json:"ShareType" gorm:"column:ShareType;default:0"`
	ShareValue  *int       `json:"ShareValue" gorm:"column:ShareValue;default:0"`
	ShareUnit   *int       `json:"ShareUnit" gorm:"column:ShareUnit;default:0"`
	Cycle       *float32   `json:"Cycle" gorm:"column:Cycle;type:float;default:0"`
	Flow        *float32   `json:"Flow" gorm:"column:Flow;type:float;default:0"`
	RecycleFlag *bool      `json:"RecycleFlag" gorm:"column:RecycleFlag;type:bit(1)"`
	FImageUrl   string     `json:"FImageUrl" gorm:"column:FImageUrl;size:255"`
	FBoxColor   string     `json:"FBoxColor" gorm:"column:FBoxColor;size:20"`
	FStatus     *int       `json:"FStatus" gorm:"column:FStatus;default:0"`
	FCreateDate *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime"`
	FModifyDate *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime"`
}

func (TbPackingSpec) TableName() string {
	return "tb_packingspec"
}
