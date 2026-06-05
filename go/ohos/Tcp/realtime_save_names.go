package tcp

import (
	"math"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func realtimeSaveStoredGradeCount(value uint8) int {
	count := int(value)
	if count < 0 {
		return 0
	}
	if count > cTCPServerMaxSizeGradeNum {
		return cTCPServerMaxSizeGradeNum
	}
	return count
}

func realtimeSaveLoopGradeCount(value int) int {
	if value <= 0 {
		return 1
	}
	if value > cTCPServerMaxSizeGradeNum {
		return cTCPServerMaxSizeGradeNum
	}
	return value
}

func realtimeSaveExportSum(sys StSysConfig) int {
	count := int(sys.NExitNum)
	if count <= 0 {
		return MAX_EXIT_NUM
	}
	if count > MAX_EXIT_NUM {
		return MAX_EXIT_NUM
	}
	return count
}

func realtimeSaveSelectWeightOrSize(classifyType uint8) string {
	if ((classifyType>>2)&1) == 1 || ((classifyType>>3)&1) == 1 || ((classifyType>>4)&1) == 1 {
		return "0"
	}
	return "1"
}

func realtimeSaveJoinedNames(data []uint8, maxItems int) string {
	names := make([]string, 0, maxItems)
	for i := 0; i < maxItems; i++ {
		name := realtimeSaveFixedName(data, i)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "，")
}

func realtimeSaveFixedName(data []uint8, index int) string {
	if index < 0 {
		return ""
	}
	start := index * cTCPServerMaxTextLength
	end := start + cTCPServerMaxTextLength
	if start >= len(data) {
		return ""
	}
	if end > len(data) {
		end = len(data)
	}
	return realtimeSaveFixedText(data[start:end])
}

func realtimeSaveFixedText(data []uint8) string {
	raw := realtimeSaveCStringBytes(data)
	if len(raw) == 0 {
		return ""
	}
	if utf8.Valid(raw) {
		return strings.TrimSpace(string(raw))
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), string(raw))
	if err == nil {
		return strings.TrimSpace(decoded)
	}
	return cleanTCPServerCString(raw)
}

func realtimeSaveCStringBytes(data []uint8) []byte {
	end := len(data)
	for i, value := range data {
		if value == 0 {
			end = i
			break
		}
	}
	raw := make([]byte, end)
	copy(raw, data[:end])
	return raw
}

func realtimeSaveUint64ToInt(value uint64) int {
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

func realtimeSaveFloatToUint64(value float64) uint64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0
	}
	maxUint := ^uint64(0)
	if value >= float64(maxUint) {
		return maxUint
	}
	return uint64(value)
}
