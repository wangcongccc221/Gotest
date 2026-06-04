package database

type TbGradeDetailInfo struct {
	FID         int      `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	CustomerID  int      `json:"CustomerID" gorm:"column:CustomerID;not null"`
	QualityName string   `json:"QualityName" gorm:"column:QualityName;type:longtext"`
	FType       string   `json:"FType" gorm:"column:FType;type:longtext"`
	TraitValue  string   `json:"TraitValue" gorm:"column:TraitValue;type:longtext"`
	TraitName   string   `json:"TraitName" gorm:"column:TraitName;type:longtext"`
	FMin        *float64 `json:"FMin" gorm:"column:FMin;type:double"`
	FMax        *float64 `json:"FMax" gorm:"column:FMax;type:double"`
	F1Min       *float64 `json:"F1Min" gorm:"column:F1Min;type:double"`
	F1Max       *float64 `json:"F1Max" gorm:"column:F1Max;type:double"`
	F2Min       *float64 `json:"F2Min" gorm:"column:F2Min;type:double"`
	F2Max       *float64 `json:"F2Max" gorm:"column:F2Max;type:double"`
}

func (TbGradeDetailInfo) TableName() string {
	return "tb_gradedetailinfo"
}
