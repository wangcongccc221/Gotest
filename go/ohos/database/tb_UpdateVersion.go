package database

type TbUpdateVersion struct {
	FDeviceType   string `json:"FDeviceType" gorm:"column:FDeviceType;size:50"`
	FFile         string `json:"FFile" gorm:"column:FFile;size:255"`
	FType         int    `json:"FType" gorm:"column:FType;default:0"`
	FProject      string `json:"FProject" gorm:"column:FProject;size:100"`
	FVersion      string `json:"FVersion" gorm:"column:FVersion;size:50"`
	FZHDescribe   string `json:"FZHDescribe" gorm:"column:FZHDescribe;size:500"`
	FENDescribe   string `json:"FENDescribe" gorm:"column:FENDescribe;size:500"`
	FIsEncryption bool   `json:"FIsEncryption" gorm:"column:FIsEncryption"`
	FLevel        int    `json:"FLevel" gorm:"column:FLevel;default:0"`
	FStrFileData  string `json:"FStrFileData" gorm:"column:FStrFileData;type:text"`
	FFileData     []byte `json:"FFileData" gorm:"column:FFileData;type:blob"`
}

func (TbUpdateVersion) TableName() string {
	return "tb_UpdateVersion"
}
