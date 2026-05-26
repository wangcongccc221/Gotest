package database

type TbExportInfo struct {
	CustomerID  int    `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement:false"`
	ExportID    int    `json:"ExportID" gorm:"column:ExportID;primaryKey;autoIncrement:false"`
	FruitNumber int    `json:"FruitNumber" gorm:"column:FruitNumber;default:0"`
	FruitWeight int    `json:"FruitWeight" gorm:"column:FruitWeight;default:0"`
	ExitName    string `json:"ExitName" gorm:"column:ExitName;size:50"`
}

func (TbExportInfo) TableName() string {
	return "tb_ExportInfo"
}
