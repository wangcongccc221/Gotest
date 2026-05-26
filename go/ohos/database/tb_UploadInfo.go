package database

type TbUploadInfo struct {
	FID           int    `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FType         string `json:"FType" gorm:"column:FType;size:50"`
	FTableName    string `json:"FTableName" gorm:"column:FTableName;size:100"`
	FCurrentRow   int    `json:"FCurrentRow" gorm:"column:FCurrentRow;default:1"`
	FMaxUploadRow int    `json:"FMaxUploadRow" gorm:"column:FMaxUploadRow;default:10"`
}

func (TbUploadInfo) TableName() string {
	return "tb_UploadInfo"
}
