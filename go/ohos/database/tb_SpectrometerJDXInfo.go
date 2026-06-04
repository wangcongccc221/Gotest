package database

type TbSpectrometerJDXInfo struct {
	FCustomerID  int      `json:"FCustomerID" gorm:"column:FCustomerID;primaryKey;autoIncrement:false"`
	FNumID       int      `json:"FNumID" gorm:"column:FNumID;primaryKey;autoIncrement:false"`
	FXCoordinate int      `json:"FXCoordinate" gorm:"column:FXCoordinate;primaryKey;autoIncrement:false"`
	FYCoordinate *float32 `json:"FYCoordinate" gorm:"column:FYCoordinate;type:float"`
}

func (TbSpectrometerJDXInfo) TableName() string {
	return "tb_spectrometerjdxinfo"
}
