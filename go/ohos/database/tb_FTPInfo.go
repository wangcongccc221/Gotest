package database

type TbFTPInfo struct {
	FtpIp       string `json:"FtpIp" gorm:"column:FtpIp;size:50"`
	FtpPort     int32  `json:"FtpPort" gorm:"column:FtpPort;default:0"`
	FtpUserName string `json:"FtpUserName" gorm:"column:FtpUserName;size:100"`
	FtpPassword string `json:"FtpPassword" gorm:"column:FtpPassword;size:100"`
}

func (TbFTPInfo) TableName() string {
	return "tb_FTPInfo"
}
