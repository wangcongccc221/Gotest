package database

import "time"

type TbPackingInfo struct {
	FID                  int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate          *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate          *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	Barcode              string     `json:"Barcode" gorm:"column:Barcode;size:100"`
	SerialCode           string     `json:"SerialCode" gorm:"column:SerialCode;size:100"`
	CustomerID           *int       `json:"CustomerID" gorm:"column:CustomerID"`
	BatchStartTime       *time.Time `json:"BatchStartTime" gorm:"column:BatchStartTime;type:datetime(6)"`
	ExitID               *int       `json:"ExitID" gorm:"column:ExitID"`
	ExitName             string     `json:"ExitName" gorm:"column:ExitName;size:50"`
	ContainerNo          *int       `json:"ContainerNo" gorm:"column:ContainerNo"`
	ContainerName        string     `json:"ContainerName" gorm:"column:ContainerName;size:255"`
	GradeName            string     `json:"GradeName" gorm:"column:GradeName;size:510"`
	WeightOrSizeName     string     `json:"WeightOrSizeName" gorm:"column:WeightOrSizeName;size:255"`
	QualityName          string     `json:"QualityName" gorm:"column:QualityName;size:255"`
	DisplayName          string     `json:"DisplayName" gorm:"column:DisplayName;size:50"`
	PackingSpecID        *int       `json:"PackingSpecID" gorm:"column:PackingSpecID"`
	PackingSpec          string     `json:"PackingSpec" gorm:"column:PackingSpec;size:100"`
	PakcingWeight        *int       `json:"PakcingWeight" gorm:"column:PakcingWeight"`
	PakcingRealWeight    *int       `json:"PakcingRealWeight" gorm:"column:PakcingRealWeight"`
	PakcingAvgWeight     *float64   `json:"PakcingAvgWeight" gorm:"column:PakcingAvgWeight;type:double;default:0"`
	PakcingRangeWeight   *int       `json:"PakcingRangeWeight" gorm:"column:PakcingRangeWeight;default:0"`
	LabelerID            *int       `json:"LabelerID" gorm:"column:LabelerID"`
	PrintDate            *time.Time `json:"PrintDate" gorm:"column:PrintDate;type:datetime(6)"`
	BoxPositionDate      *time.Time `json:"BoxPositionDate" gorm:"column:BoxPositionDate;type:datetime(6)"`
	BoxFillFinishedDate  *time.Time `json:"BoxFillFinishedDate" gorm:"column:BoxFillFinishedDate;type:datetime(6)"`
	SortingID            *int       `json:"SortingID" gorm:"column:SortingID"`
	SortingExitID        *int       `json:"SortingExitID" gorm:"column:SortingExitID"`
	SortingExitName      string     `json:"SortingExitName" gorm:"column:SortingExitName;size:50"`
	SortingDate          *time.Time `json:"SortingDate" gorm:"column:SortingDate;type:datetime(6)"`
	PalletFID            *int       `json:"PalletFID" gorm:"column:PalletFID"`
	State                *int       `json:"State" gorm:"column:State"`
	FSubSystem           *int       `json:"FSubSystem" gorm:"column:FSubSystem"`
	Zespri               *bool      `json:"Zespri" gorm:"column:Zespri;type:tinyint(1)"`
	Rewe                 *bool      `json:"Rewe" gorm:"column:Rewe;type:tinyint(1)"`
	Origin               string     `json:"Origin" gorm:"column:Origin;type:longtext"`
	FruitClass           string     `json:"FruitClass" gorm:"column:FruitClass;type:longtext"`
	Variety              string     `json:"Variety" gorm:"column:Variety;type:longtext"`
	Count                string     `json:"Count" gorm:"column:Count;type:longtext"`
	Brand                string     `json:"Brand" gorm:"column:Brand;type:longtext"`
	GrowingMethod        string     `json:"GrowingMethod" gorm:"column:GrowingMethod;type:longtext"`
	PackStyle            string     `json:"PackStyle" gorm:"column:PackStyle;type:longtext"`
	LabellingIndicator   string     `json:"LabellingIndicator" gorm:"column:LabellingIndicator;type:longtext"`
	GramsRange           string     `json:"GramsRange" gorm:"column:GramsRange;type:longtext"`
	Packhouse            string     `json:"Packhouse" gorm:"column:Packhouse;type:longtext"`
	GlobalGapNumber      string     `json:"GlobalGapNumber" gorm:"column:GlobalGapNumber;type:longtext"`
	GlobalLocationNumber string     `json:"GlobalLocationNumber" gorm:"column:GlobalLocationNumber;type:longtext"`
	Year                 string     `json:"Year" gorm:"column:Year;type:longtext"`
	FruitSize            string     `json:"FruitSize" gorm:"column:FruitSize;type:longtext"`
	Kpin                 string     `json:"Kpin" gorm:"column:Kpin;type:longtext"`
	PalletNumber         string     `json:"PalletNumber" gorm:"column:PalletNumber;type:longtext"`
	PackSequence         string     `json:"PackSequence" gorm:"column:PackSequence;type:longtext"`
	GraderDropCode       string     `json:"GraderDropCode" gorm:"column:GraderDropCode;type:longtext"`
	Packrun              string     `json:"Packrun" gorm:"column:Packrun;type:longtext"`
	PackBarcode          string     `json:"PackBarcode" gorm:"column:PackBarcode;type:longtext"`
	ReweBarcode          string     `json:"ReweBarcode" gorm:"column:ReweBarcode;type:longtext"`
	BoxNumber            *int       `json:"BoxNumber" gorm:"column:BoxNumber;default:0"`
	WeightSysID          *int       `json:"WeightSysID" gorm:"column:WeightSysID;default:0"`
	BoxWeight            *float64   `json:"BoxWeight" gorm:"column:BoxWeight;type:double;default:0"`
	WeightingDate        *time.Time `json:"WeightingDate" gorm:"column:WeightingDate;type:datetime"`
}

func (TbPackingInfo) TableName() string {
	return "tb_packinginfo"
}
