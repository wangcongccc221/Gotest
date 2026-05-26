package database

type TbSysFruitInfo struct {
	FID         int `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	CustomerID  int `json:"CustomerID" gorm:"column:CustomerID;not null"`
	SystemID    int `json:"SystemID" gorm:"column:SystemID;not null"`
	BatchNumber int `json:"BatchNumber" gorm:"column:BatchNumber"`
	BatchWeight int `json:"BatchWeight" gorm:"column:BatchWeight"`
}

func (TbSysFruitInfo) TableName() string {
	return "tb_Sys_FruitInfo"
}
