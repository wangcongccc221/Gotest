package database

type TbPriceInfo struct {
	FID       int     `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	Price     float64 `json:"Price" gorm:"column:Price;type:decimal(10,5)"`
	PriceName string  `json:"PriceName" gorm:"column:PriceName;size:50"`
}

func (TbPriceInfo) TableName() string {
	return "tb_PriceInfo"
}
