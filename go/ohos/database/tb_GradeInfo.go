package database

type TbGradeInfo struct {
	CustomerID         int     `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement:false"`
	GradeID            int     `json:"GradeID" gorm:"column:GradeID;primaryKey;autoIncrement:false"`
	BoxNumber          int     `json:"BoxNumber" gorm:"column:BoxNumber;default:0"`
	FruitNumber        int     `json:"FruitNumber" gorm:"column:FruitNumber;default:0"`
	FruitWeight        int     `json:"FruitWeight" gorm:"column:FruitWeight;default:0"`
	QualityName        string  `json:"QualityName" gorm:"column:QualityName;size:100"`
	WeightOrSizeName   string  `json:"WeightOrSizeName" gorm:"column:WeightOrSizeName;size:100"`
	WeightOrSizeLimit  float64 `json:"WeightOrSizeLimit" gorm:"column:WeightOrSizeLimit;type:double;default:0"`
	SelectWeightOrSize string  `json:"SelectWeightOrSize" gorm:"column:SelectWeightOrSize;size:10"`
	TraitWeightOrSize  string  `json:"TraitWeightOrSize" gorm:"column:TraitWeightOrSize;size:50"`
	TraitColor         string  `json:"TraitColor" gorm:"column:TraitColor;size:50"`
	TraitShape         string  `json:"TraitShape" gorm:"column:TraitShape;size:50"`
	TraitFlaw          string  `json:"TraitFlaw" gorm:"column:TraitFlaw;size:50"`
	TraitHard          string  `json:"TraitHard" gorm:"column:TraitHard;size:50"`
	TraitDensity       string  `json:"TraitDensity" gorm:"column:TraitDensity;size:50"`
	TraitSugarDegree   string  `json:"TraitSugarDegree" gorm:"column:TraitSugarDegree;size:50"`
	FPrice             float64 `json:"FPrice" gorm:"column:FPrice;type:varchar(50);not null;default:0.00"`
	NSizeMax           float64 `json:"nSizeMax" gorm:"column:nSizeMax;type:double;default:0.00"`
	NSizeMin           float64 `json:"nSizeMin" gorm:"column:nSizeMin;type:double;default:0.00"`
}

func (TbGradeInfo) TableName() string {
	return "tb_GradeInfo"
}
