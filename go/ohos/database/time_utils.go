package database

import "time"

var databaseChinaLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func databaseNow() time.Time {
	return time.Now().In(databaseChinaLocation)
}

func databaseLocalTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.In(databaseChinaLocation)
}
