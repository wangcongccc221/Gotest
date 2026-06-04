package database

type TbExportDetailInfo struct {
	CustomerID       int    `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement:false"`
	ExportID         int    `json:"ExportID" gorm:"column:ExportID;primaryKey;autoIncrement:false"`
	QualityName      string `json:"QualityName" gorm:"column:QualityName;type:longtext"`
	WeightOrSizeName string `json:"WeightOrSizeName" gorm:"column:WeightOrSizeName;type:longtext"`
}

func (TbExportDetailInfo) TableName() string {
	return "tb_exportdetailinfo"
}
