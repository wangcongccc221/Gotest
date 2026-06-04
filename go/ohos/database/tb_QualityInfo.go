package database

type TbQualityInfo struct {
	FCustomerID         int      `json:"FCustomerID" gorm:"column:FCustomerID;primaryKey;autoIncrement:false"`
	FNumID              int      `json:"FNumID" gorm:"column:FNumID;primaryKey;autoIncrement:false"`
	FFruitSize          *int     `json:"FFruitSize" gorm:"column:FFruitSize"`
	FSpeed              *int     `json:"FSpeed" gorm:"column:FSpeed"`
	FCalTimes           *int     `json:"FCalTimes" gorm:"column:FCalTimes"`
	FBrix               *float32 `json:"FBrix" gorm:"column:FBrix;type:float"`
	FAcidity            *float32 `json:"FAcidity" gorm:"column:FAcidity;type:float"`
	FBrowning           *float32 `json:"FBrowning" gorm:"column:FBrowning;type:float"`
	FHardness           *float32 `json:"FHardness" gorm:"column:FHardness;type:float"`
	FWaterHeartDisease  *float32 `json:"FWaterHeartDisease" gorm:"column:FWaterHeartDisease;type:float"`
	FDryMatter          *float32 `json:"FDryMatter" gorm:"column:FDryMatter;type:float"`
	FHollow             *float32 `json:"FHollow" gorm:"column:FHollow;type:float"`
	FRBrix              *float32 `json:"FRBrix" gorm:"column:FRBrix;type:float"`
	FRAcidity           *float32 `json:"FRAcidity" gorm:"column:FRAcidity;type:float"`
	FRBrowning          *float32 `json:"FRBrowning" gorm:"column:FRBrowning;type:float"`
	FRHardness          *float32 `json:"FRHardness" gorm:"column:FRHardness;type:float"`
	FRWaterHeartDisease *float32 `json:"FRWaterHeartDisease" gorm:"column:FRWaterHeartDisease;type:float"`
	FRDryMatter         *float32 `json:"FRDryMatter" gorm:"column:FRDryMatter;type:float"`
	FRHollow            *float32 `json:"FRHollow" gorm:"column:FRHollow;type:float"`
}

func (TbQualityInfo) TableName() string {
	return "tb_qualityinfo"
}
