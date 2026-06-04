package database

type TbFruitInfo struct {
	CustomerID           int      `json:"CustomerID" gorm:"column:CustomerID;primaryKey;autoIncrement"`
	SysID                *int     `json:"SysID" gorm:"column:SysID;index:SysID"`
	FBatchNo             string   `json:"FBatchNo" gorm:"column:FBatchNo;type:longtext"`
	CustomerCode         string   `json:"CustomerCode" gorm:"column:CustomerCode;size:100"`
	CustomerName         string   `json:"CustomerName" gorm:"column:CustomerName;type:longtext"`
	FarmName             string   `json:"FarmName" gorm:"column:FarmName;type:longtext"`
	FruitName            string   `json:"FruitName" gorm:"column:FruitName;type:longtext"`
	StartTime            string   `json:"StartTime" gorm:"column:StartTime;type:longtext"`
	EndTime              string   `json:"EndTime" gorm:"column:EndTime;type:longtext"`
	StartedState         string   `json:"StartedState" gorm:"column:StartedState;type:longtext"`
	CompletedState       string   `json:"CompletedState" gorm:"column:CompletedState;type:longtext"`
	BatchWeight          *float64 `json:"BatchWeight" gorm:"column:BatchWeight;type:double;default:0"`
	BatchNumber          *uint64  `json:"BatchNumber" gorm:"column:BatchNumber;type:bigint unsigned;default:0"`
	QualityGradeSum      *int     `json:"QualityGradeSum" gorm:"column:QualityGradeSum"`
	WeightOrSizeGradeSum *int     `json:"WeightOrSizeGradeSum" gorm:"column:WeightOrSizeGradeSum"`
	ExportSum            *int     `json:"ExportSum" gorm:"column:ExportSum"`
	ColorGradeName       string   `json:"ColorGradeName" gorm:"column:ColorGradeName;type:longtext"`
	ShapeGradeName       string   `json:"ShapeGradeName" gorm:"column:ShapeGradeName;type:longtext"`
	FlawGradeName        string   `json:"FlawGradeName" gorm:"column:FlawGradeName;type:longtext"`
	HardGradeName        string   `json:"HardGradeName" gorm:"column:HardGradeName;type:longtext"`
	DensityGradeName     string   `json:"DensityGradeName" gorm:"column:DensityGradeName;type:longtext"`
	SugarDegreeGradeName string   `json:"SugarDegreeGradeName" gorm:"column:SugarDegreeGradeName;type:longtext"`
	ProgramName          string   `json:"ProgramName" gorm:"column:ProgramName;type:longtext"`
	Remarks              string   `json:"Remarks" gorm:"column:Remarks;type:longtext"`
	FVisible             *int     `json:"FVisible" gorm:"column:FVisible"`
	OrderID              *int     `json:"OrderID" gorm:"column:OrderID"`
	MajorCustomerID      *int     `json:"MajorCustomerID" gorm:"column:MajorCustomerID;index:MajorCustomerID"`
	ChainIdx             string   `json:"ChainIdx" gorm:"column:ChainIdx;type:longtext"`
	PlanCustomerID       string   `json:"PlanCustomerID" gorm:"column:PlanCustomerID;type:longtext"`
	BeforeMaterialID     string   `json:"BeforeMaterialID" gorm:"column:BeforeMaterialID;type:longtext"`
	IsEnd                string   `json:"IsEnd" gorm:"column:IsEnd;type:longtext"`
	IsSortQua            string   `json:"IsSortQua" gorm:"column:IsSortQua;type:longtext"`
	FUnit                string   `json:"FUnit" gorm:"column:FUnit;type:longtext"`
	Comment              string   `json:"Comment" gorm:"column:Comment;size:50"`
	SortBaseName         string   `json:"SortBaseName" gorm:"-"`
	PageSize             int      `json:"PageSize" gorm:"-"`
	PageIndex            int      `json:"PageIndex" gorm:"-"`
	IsMerge              bool     `json:"IsMerge" gorm:"-"`
	SortColumn           string   `json:"SortColumn" gorm:"-"`
	SortOrder            string   `json:"SortOrder" gorm:"-"`
	FBatchWeight         float64  `json:"-" gorm:"-"`
}

func (TbFruitInfo) TableName() string {
	return "tb_fruitinfo"
}
