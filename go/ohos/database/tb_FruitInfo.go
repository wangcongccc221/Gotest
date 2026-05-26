package database

type TbFruitInfo struct {
	CustomerID           int     `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement"`
	SysID                int     `json:"SysID" gorm:"column:SysID;default:0"`
	MajorCustomerID      int     `json:"MajorCustomerID" gorm:"column:MajorCustomerID;default:0"`
	ChainIdx             string  `json:"ChainIdx" gorm:"column:ChainIdx;size:50;default:0"`
	FBatchNo             string  `json:"FBatchNo" gorm:"column:FBatchNo;size:50"`
	CustomerName         string  `json:"CustomerName" gorm:"column:CustomerName;size:100"`
	FarmName             string  `json:"FarmName" gorm:"column:FarmName;size:100"`
	FruitName            string  `json:"FruitName" gorm:"column:FruitName;size:100"`
	SortBaseName         string  `json:"SortBaseName" gorm:"column:SortBaseName;size:100"`
	StartTime            string  `json:"StartTime" gorm:"column:StartTime;size:30"`
	EndTime              string  `json:"EndTime" gorm:"column:EndTime;size:30"`
	StartedState         string  `json:"StartedState" gorm:"column:StartedState;size:10"`
	CompletedState       string  `json:"CompletedState" gorm:"column:CompletedState;size:10"`
	BatchWeight          int     `json:"BatchWeight" gorm:"column:BatchWeight;default:0"`
	BatchNumber          int     `json:"BatchNumber" gorm:"column:BatchNumber;default:0"`
	QualityGradeSum      int     `json:"QualityGradeSum" gorm:"column:QualityGradeSum;default:0"`
	WeightOrSizeGradeSum int     `json:"WeightOrSizeGradeSum" gorm:"column:WeightOrSizeGradeSum;default:0"`
	ExportSum            int     `json:"ExportSum" gorm:"column:ExportSum;default:0"`
	ColorGradeName       string  `json:"ColorGradeName" gorm:"column:ColorGradeName;size:100"`
	ShapeGradeName       string  `json:"ShapeGradeName" gorm:"column:ShapeGradeName;size:100"`
	FlawGradeName        string  `json:"FlawGradeName" gorm:"column:FlawGradeName;size:100"`
	HardGradeName        string  `json:"HardGradeName" gorm:"column:HardGradeName;size:100"`
	DensityGradeName     string  `json:"DensityGradeName" gorm:"column:DensityGradeName;size:100"`
	SugarDegreeGradeName string  `json:"SugarDegreeGradeName" gorm:"column:SugarDegreeGradeName;size:100"`
	ProgramName          string  `json:"ProgramName" gorm:"column:ProgramName;size:100"`
	FVisible             int     `json:"FVisible" gorm:"column:FVisible;not null;default:1"`
	PageSize             int     `json:"PageSize" gorm:"-"`
	PageIndex            int     `json:"PageIndex" gorm:"-"`
	IsMerge              bool    `json:"IsMerge" gorm:"-"`
	SortColumn           string  `json:"SortColumn" gorm:"-"`
	SortOrder            string  `json:"SortOrder" gorm:"-"`
	FBatchWeight         float64 `json:"-" gorm:"-"`
}

func (TbFruitInfo) TableName() string {
	return "tb_FruitInfo"
}
