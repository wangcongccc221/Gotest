package database

type TbLocation struct {
	FID       int    `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCode     string `json:"FCode" gorm:"column:FCode;size:50"`
	FName     string `json:"FName" gorm:"column:FName;size:100"`
	FParentID int    `json:"FParentID" gorm:"column:FParentID;default:0"`
	FLevel    int    `json:"FLevel" gorm:"column:FLevel;default:0"`
}

func (TbLocation) TableName() string {
	return "tb_Location"
}
