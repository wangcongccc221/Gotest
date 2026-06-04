package database

type TbSampleInfo struct {
	FID              int    `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCustomerID      *int   `json:"FCustomerID" gorm:"column:FCustomerID"`
	FStartTime       string `json:"FStartTime" gorm:"column:FStartTime;type:longtext"`
	FEndTime         string `json:"FEndTime" gorm:"column:FEndTime;type:longtext"`
	FGradeName       string `json:"FGradeName" gorm:"column:FGradeName;type:longtext"`
	FSampleNumber    *int   `json:"FSampleNumber" gorm:"column:FSampleNumber"`
	FHasSampleNumber *int   `json:"FHasSampleNumber" gorm:"column:FHasSampleNumber"`
	FBadNumber       *int   `json:"FBadNumber" gorm:"column:FBadNumber"`
}

func (TbSampleInfo) TableName() string {
	return "tb_sampleinfo"
}
