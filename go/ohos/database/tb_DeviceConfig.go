package database

import "time"

type TbDeviceConfig struct {
	FID                int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate        *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate        *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FModuleName        string     `json:"FModuleName" gorm:"column:FModuleName;type:longtext"`
	FTaskName          string     `json:"FTaskName" gorm:"column:FTaskName;size:100"`
	FName              string     `json:"FName" gorm:"column:FName;type:longtext"`
	FIP                string     `json:"FIP" gorm:"column:FIP;type:longtext"`
	FPort              *int       `json:"FPort" gorm:"column:FPort"`
	FUserName          string     `json:"FUserName" gorm:"column:FUserName;type:longtext"`
	FPassword          string     `json:"FPassword" gorm:"column:FPassword;type:longtext"`
	FVirtualHost       string     `json:"FVirtualHost" gorm:"column:FVirtualHost;type:longtext"`
	FType              *int       `json:"FType" gorm:"column:FType"`
	FFunctionID        *int       `json:"FFunctionID" gorm:"column:FFunctionID"`
	FFunctionCode      *int       `json:"FFunctionCode" gorm:"column:FFunctionCode"`
	FCodePrefix        string     `json:"FCodePrefix" gorm:"column:FCodePrefix;type:longtext"`
	FCodeSuffix        string     `json:"FCodeSuffix" gorm:"column:FCodeSuffix;type:longtext"`
	FTriggerCode       string     `json:"FTriggerCode" gorm:"column:FTriggerCode;type:longtext"`
	FNoRead            string     `json:"FNoRead" gorm:"column:FNoRead;type:longtext"`
	FHearBeat          string     `json:"FHearBeat" gorm:"column:FHearBeat;size:100"`
	FHearBeatTime      *int       `json:"FHearBeatTime" gorm:"column:FHearBeatTime;default:0"`
	FMaxRetries        *int       `json:"FMaxRetries" gorm:"column:FMaxRetries"`
	FTimeout           *int       `json:"FTimeout" gorm:"column:FTimeout"`
	FRelatedID         *int       `json:"FRelatedID" gorm:"column:FRelatedID"`
	FDetailRelatedID   *int       `json:"FDetailRelatedID" gorm:"column:FDetailRelatedID"`
	FFailedID          *int       `json:"FFailedID" gorm:"column:FFailedID"`
	FFailedDetailID    *int       `json:"FFailedDetailID" gorm:"column:FFailedDetailID"`
	FFaileds           *int       `json:"FFaileds" gorm:"column:FFaileds"`
	FExceptionID       *int       `json:"FExceptionID" gorm:"column:FExceptionID"`
	FDetailExceptionID *int       `json:"FDetailExceptionID" gorm:"column:FDetailExceptionID"`
	FExceptions        *int       `json:"FExceptions" gorm:"column:FExceptions"`
	FEnable            *bool      `json:"FEnable" gorm:"column:FEnable;type:tinyint(1)"`
}

func (TbDeviceConfig) TableName() string {
	return "tb_deviceconfig"
}
