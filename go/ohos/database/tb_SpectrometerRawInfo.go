package database

type TbSpectrometerRawInfo struct {
	FCustomerID int      `json:"FCustomerID" gorm:"column:FCustomerID;primaryKey;autoIncrement:false"`
	FPixel      int      `json:"FPixel" gorm:"column:FPixel;primaryKey;autoIncrement:false"`
	FWaveL      *float32 `json:"FWaveL" gorm:"column:FWaveL;type:float"`
	FDarkR      *int     `json:"FDarkR" gorm:"column:FDarkR"`
	FDarkSS     *int     `json:"FDarkS_S" gorm:"column:FDarkS_S"`
	FDarkSM     *int     `json:"FDarkS_M" gorm:"column:FDarkS_M"`
	FDarkSL     *int     `json:"FDarkS_L" gorm:"column:FDarkS_L"`
	FREF        *int     `json:"FREF" gorm:"column:FREF"`
}

func (TbSpectrometerRawInfo) TableName() string {
	return "tb_spectrometerrawinfo"
}
