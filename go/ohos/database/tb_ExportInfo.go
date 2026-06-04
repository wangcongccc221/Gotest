package database

type TbExportInfo struct {
	CustomerID       int      `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement:false"`
	ExportID         int      `json:"ExportID" gorm:"column:ExportID;primaryKey;autoIncrement:false"`
	FruitNumber      *uint64  `json:"FruitNumber" gorm:"column:FruitNumber;type:bigint unsigned;default:0"`
	FruitWeight      *float64 `json:"FruitWeight" gorm:"column:FruitWeight;type:double;default:0"`
	ExitName         string   `json:"ExitName" gorm:"column:ExitName;type:longtext"`
	FAfterMaterialID string   `json:"FAfterMaterialID" gorm:"column:FAfterMaterialID;type:longtext"`
	FSpecName        string   `json:"FSpecName" gorm:"column:FSpecName;type:longtext"`
	SysID            *int     `json:"SysID" gorm:"column:SysID;default:0"`
}

func (TbExportInfo) TableName() string {
	return "tb_exportinfo"
}
