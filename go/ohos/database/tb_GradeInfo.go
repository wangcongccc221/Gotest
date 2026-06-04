package database

type TbGradeInfo struct {
	CustomerID         int      `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement:false"`
	GradeID            int      `json:"GradeID" gorm:"column:GradeID;primaryKey;autoIncrement:false"`
	BoxNumber          *float64 `json:"BoxNumber" gorm:"column:BoxNumber;type:double;default:0"`
	FruitNumber        *uint64  `json:"FruitNumber" gorm:"column:FruitNumber;type:bigint unsigned;default:0"`
	FruitWeight        *float64 `json:"FruitWeight" gorm:"column:FruitWeight;type:double;default:0"`
	BoxStandard        *float64 `json:"BoxStandard" gorm:"column:BoxStandard;type:double;default:0"`
	QualityName        string   `json:"QualityName" gorm:"column:QualityName;type:longtext"`
	WeightOrSizeName   string   `json:"WeightOrSizeName" gorm:"column:WeightOrSizeName;type:longtext"`
	WeightOrSizeLimit  *float64 `json:"WeightOrSizeLimit" gorm:"column:WeightOrSizeLimit;type:double"`
	SelectWeightOrSize string   `json:"SelectWeightOrSize" gorm:"column:SelectWeightOrSize;type:longtext"`
	TraitWeightOrSize  string   `json:"TraitWeightOrSize" gorm:"column:TraitWeightOrSize;type:longtext"`
	TraitColor         string   `json:"TraitColor" gorm:"column:TraitColor;type:longtext"`
	TraitShape         string   `json:"TraitShape" gorm:"column:TraitShape;type:longtext"`
	TraitFlaw          string   `json:"TraitFlaw" gorm:"column:TraitFlaw;type:longtext"`
	TraitHard          string   `json:"TraitHard" gorm:"column:TraitHard;type:longtext"`
	TraitDensity       string   `json:"TraitDensity" gorm:"column:TraitDensity;type:longtext"`
	TraitSugarDegree   string   `json:"TraitSugarDegree" gorm:"column:TraitSugarDegree;type:longtext"`
	FPrice             *float64 `json:"FPrice" gorm:"column:FPrice;type:double"`
	NSizeMax           float64  `json:"nSizeMax" gorm:"-"`
	NSizeMin           float64  `json:"nSizeMin" gorm:"-"`
}

func (TbGradeInfo) TableName() string {
	return "tb_gradeinfo"
}
