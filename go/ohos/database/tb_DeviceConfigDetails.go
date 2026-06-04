package database

import "time"

type TbDeviceConfigDetails struct {
	FID             int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate     *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate     *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FDeviceID       *int       `json:"FDeviceID" gorm:"column:FDeviceID"`
	FName           string     `json:"FName" gorm:"column:FName;type:longtext"`
	FTargetName     string     `json:"FTargetName" gorm:"column:FTargetName;type:longtext"`
	FDataType       *int       `json:"FDataType" gorm:"column:FDataType"`
	FFunctionCode   *int       `json:"FFunctionCode" gorm:"column:FFunctionCode"`
	FNumberOfPoints *int       `json:"FNumberOfPoints" gorm:"column:FNumberOfPoints"`
	FRelatedID      *int       `json:"FRelatedID" gorm:"column:FRelatedID"`
	FMainRelatedID  *int       `json:"FMainRelatedID" gorm:"column:FMainRelatedID"`
	FIntervalTime   *int       `json:"FIntervalTime" gorm:"column:FIntervalTime"`
	FParams         string     `json:"FParams" gorm:"column:FParams;size:2000"`
	FSucceed        string     `json:"FSucceed" gorm:"column:FSucceed;size:2000"`
	FResult         string     `json:"FResult" gorm:"column:FResult;size:2000"`
	FState          *int       `json:"FState" gorm:"column:FState"`
}

func (TbDeviceConfigDetails) TableName() string {
	return "tb_deviceconfigdetails"
}
