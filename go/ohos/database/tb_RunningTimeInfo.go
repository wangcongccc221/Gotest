package database

type TbRunningTimeInfo struct {
	ID          int    `json:"ID" gorm:"column:ID;primaryKey;autoIncrement"`
	RunningDate string `json:"RunningDate" gorm:"column:RunningDate;size:30"`
	StartTime   string `json:"StartTime" gorm:"column:StartTime;size:30"`
	StopTime    string `json:"StopTime" gorm:"column:StopTime;size:30"`
}

func (TbRunningTimeInfo) TableName() string {
	return "tb_RunningTimeInfo"
}
