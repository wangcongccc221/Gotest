package database

import "time"

type TbFruitDetails struct {
	FID            int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate    *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate    *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FFruitID       *uint      `json:"FFruitID" gorm:"column:FFruitID"`
	FSubSystemID   *uint      `json:"FSubSystemID" gorm:"column:FSubSystemID"`
	FCustomerID    *int       `json:"FCustomerID" gorm:"column:FCustomerID"`
	FChannelID     *uint      `json:"FChannelID" gorm:"column:FChannelID"`
	FExitID        *uint16    `json:"FExitID" gorm:"column:FExitID"`
	FColorRate0    *uint      `json:"FColorRate0" gorm:"column:FColorRate0"`
	FColorRate1    *uint      `json:"FColorRate1" gorm:"column:FColorRate1"`
	FColorRate2    *uint      `json:"FColorRate2" gorm:"column:FColorRate2"`
	FArea          *uint      `json:"FArea" gorm:"column:FArea"`
	FFlawArea      *uint      `json:"FFlawArea" gorm:"column:FFlawArea"`
	FVolume        *uint      `json:"FVolume" gorm:"column:FVolume"`
	FFlawNum       *uint      `json:"FFlawNum" gorm:"column:FFlawNum"`
	FMaxDR         *float64   `json:"FMaxDR" gorm:"column:FMaxDR;type:double"`
	FMinDR         *float64   `json:"FMinDR" gorm:"column:FMinDR;type:double"`
	FDiameter      *float64   `json:"FDiameter" gorm:"column:FDiameter;type:double"`
	FDiameterRatio *float64   `json:"FDiameterRatio" gorm:"column:FDiameterRatio;type:double"`
	FMinDRatio     *float64   `json:"FMinDRatio" gorm:"column:FMinDRatio;type:double"`
	FBruiseArea    *uint      `json:"FBruiseArea" gorm:"column:FBruiseArea"`
	FBruiseNum     *uint      `json:"FBruiseNum" gorm:"column:FBruiseNum"`
	FRotArea       *uint      `json:"FRotArea" gorm:"column:FRotArea"`
	FRotNum        *uint      `json:"FRotNum" gorm:"column:FRotNum"`
	FRigidity      *uint      `json:"FRigidity" gorm:"column:FRigidity"`
	FWater         *uint      `json:"FWater" gorm:"column:FWater"`
	FSugar         *float64   `json:"FSugar" gorm:"column:FSugar;type:double"`
	FAcidity       *float64   `json:"FAcidity" gorm:"column:FAcidity;type:double"`
	FMildew        *float64   `json:"FMildew" gorm:"column:FMildew;type:double"`
	FExarid        *float64   `json:"FExarid" gorm:"column:FExarid;type:double"`
	FPulp          *float64   `json:"FPulp" gorm:"column:FPulp;type:double"`
	FRipe          *float64   `json:"FRipe" gorm:"column:FRipe;type:double"`
	FWeight        *float64   `json:"FWeight" gorm:"column:FWeight;type:double"`
	FDensity       *float64   `json:"FDensity" gorm:"column:FDensity;type:double"`
	FGradeID       *uint      `json:"FGradeID" gorm:"column:FGradeID"`
	FProductID     *uint8     `json:"FProductID" gorm:"column:FProductID"`
	FLabelerID     *uint8     `json:"FLabelerID" gorm:"column:FLabelerID"`
	FBoxID         *uint      `json:"FBoxID" gorm:"column:FBoxID"`
	FPackNum       *uint16    `json:"FPackNum" gorm:"column:FPackNum"`
	FPackWeight    *float64   `json:"FPackWeight" gorm:"column:FPackWeight;type:double"`
	FPackArea      *uint      `json:"FPackArea" gorm:"column:FPackArea"`
	FPackVolume    *uint      `json:"FPackVolume" gorm:"column:FPackVolume"`
	FState         *int       `json:"FState" gorm:"column:FState"`
}

func (TbFruitDetails) TableName() string {
	return "tb_fruitdetails"
}
