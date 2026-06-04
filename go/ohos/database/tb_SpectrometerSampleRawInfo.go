package database

type TbSpectrometerSampleRawInfo struct {
	FCustomerID int  `json:"FCustomerID" gorm:"column:FCustomerID;primaryKey;autoIncrement:false"`
	FNumID      int  `json:"FNumID" gorm:"column:FNumID;primaryKey;autoIncrement:false"`
	FPixel      int  `json:"FPixel" gorm:"column:FPixel;primaryKey;autoIncrement:false"`
	FValue      *int `json:"FValue" gorm:"column:FValue"`
}

func (TbSpectrometerSampleRawInfo) TableName() string {
	return "tb_spectrometersamplerawinfo"
}
