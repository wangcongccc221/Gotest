package tcp

import "time"

var cTCPChinaLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func cTCPNow() time.Time {
	return time.Now().In(cTCPChinaLocation)
}

func cTCPLocalTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.In(cTCPChinaLocation)
}
