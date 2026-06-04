package database

import "time"

type TbCustomField struct {
	FID            int        `json:"FID" gorm:"column:FID;primaryKey;autoIncrement"`
	FCreateDate    *time.Time `json:"FCreateDate" gorm:"column:FCreateDate;type:datetime(6)"`
	FModifyDate    *time.Time `json:"FModifyDate" gorm:"column:FModifyDate;type:datetime(6)"`
	FModule        string     `json:"FModule" gorm:"column:FModule;type:longtext"`
	FFieldKey      string     `json:"FFieldKey" gorm:"column:FFieldKey;type:longtext"`
	FDisplayName   string     `json:"FDisplayName" gorm:"column:FDisplayName;type:longtext"`
	FDataType      string     `json:"FDataType" gorm:"column:FDataType;type:longtext"`
	FIsRequired    *int       `json:"FIsRequired" gorm:"column:FIsRequired"`
	FDefaultValue  string     `json:"FDefaultValue" gorm:"column:FDefaultValue;type:longtext"`
	FSort          *int       `json:"FSort" gorm:"column:FSort"`
	FDataSourceID  *int       `json:"FDataSourceID" gorm:"column:FDataSourceID"`
	FStatus        *int       `json:"FStatus" gorm:"column:FStatus"`
	FCoordinateX   *int       `json:"FCoordinateX" gorm:"column:FCoordinateX"`
	FCoordinateY   *int       `json:"FCoordinateY" gorm:"column:FCoordinateY"`
	FDisplayTime   string     `json:"FDisplayTime" gorm:"column:FDisplayTime;type:longtext"`
	FCustomerField string     `json:"FCustomerField" gorm:"column:FCustomerField;type:longtext"`
}

func (TbCustomField) TableName() string {
	return "tb_customfield"
}
