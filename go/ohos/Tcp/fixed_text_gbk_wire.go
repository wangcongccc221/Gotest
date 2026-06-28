package tcp

import (
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type webSocketGradeNameTexts struct {
	SizeGradeNames    []string `json:"sizeGradeNames,omitempty"`
	QualityGradeNames []string `json:"qualityGradeNames,omitempty"`
}

func writeGBKFixedTextSlot(dst []byte, text string) {
	for i := range dst {
		dst[i] = 0
	}
	if len(dst) == 0 {
		return
	}

	pos := 0
	for _, r := range strings.ReplaceAll(text, "\x00", "") {
		encoded := encodeRuneToGBK(r)
		if len(encoded) == 0 {
			continue
		}
		if pos+len(encoded) > len(dst) {
			break
		}
		copy(dst[pos:], encoded)
		pos += len(encoded)
	}
}

func writeUTF8FixedTextSlot(dst []byte, text string) {
	for i := range dst {
		dst[i] = 0
	}
	if len(dst) == 0 {
		return
	}

	pos := 0
	for _, r := range strings.ReplaceAll(text, "\x00", "") {
		encoded := []byte(string(r))
		if pos+len(encoded) > len(dst) {
			break
		}
		copy(dst[pos:], encoded)
		pos += len(encoded)
	}
}

func rewriteFixedTextSlotAsGBK(dst []uint8) {
	text := realtimeSaveFixedText(dst)
	writeGBKFixedTextSlot(dst, text)
}

func rewriteFixedTextSlotsAsGBK(dst []uint8, slotSize int) {
	if slotSize <= 0 {
		return
	}
	for start := 0; start < len(dst); start += slotSize {
		end := start + slotSize
		if end > len(dst) {
			end = len(dst)
		}
		rewriteFixedTextSlotAsGBK(dst[start:end])
	}
}

func writeGBKFixedTextSlotsFromStrings(dst []uint8, slotSize int, texts []string) {
	if slotSize <= 0 || len(texts) == 0 {
		return
	}
	for index, text := range texts {
		start := index * slotSize
		if start >= len(dst) {
			break
		}
		end := start + slotSize
		if end > len(dst) {
			end = len(dst)
		}
		writeGBKFixedTextSlot(dst[start:end], text)
	}
}

func decodeGBKFixedTextSlot(src []uint8) string {
	raw := realtimeSaveCStringBytes(src)
	if len(raw) == 0 {
		return ""
	}
	decoded, _, err := transform.String(simplifiedchinese.GBK.NewDecoder(), string(raw))
	if err != nil {
		return cleanTCPServerCString(raw)
	}
	return strings.TrimSpace(decoded)
}

func rewriteGBKFixedTextSlotAsUTF8(dst []uint8) {
	text := decodeGBKFixedTextSlot(dst)
	writeUTF8FixedTextSlot(dst, text)
}

func rewriteGBKFixedTextSlotsAsUTF8(dst []uint8, slotSize int) {
	if slotSize <= 0 {
		return
	}
	for start := 0; start < len(dst); start += slotSize {
		end := start + slotSize
		if end > len(dst) {
			end = len(dst)
		}
		rewriteGBKFixedTextSlotAsUTF8(dst[start:end])
	}
}

func encodeRuneToGBK(r rune) []byte {
	encoded, _, err := transform.String(encoding.ReplaceUnsupported(simplifiedchinese.GBK.NewEncoder()), string(r))
	if err != nil {
		return []byte("?")
	}
	return []byte(encoded)
}

func gradeInfoForGBKWire(grade StGradeInfo) StGradeInfo {
	rewriteFixedTextSlotAsGBK(grade.StrFruitName[:])
	rewriteFixedTextSlotsAsGBK(grade.StrSizeGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StrQualityGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StDensityGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StrColorGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StrShapeGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StFlawareaGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StBruiseGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StRotGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StSugarGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StAcidityGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StHollowGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StSkinGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StBrownGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StTangxinGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StRigidityGradeName[:], cTCPServerMaxTextLength)
	rewriteFixedTextSlotsAsGBK(grade.StWaterGradeName[:], cTCPServerMaxTextLength)
	return grade
}

func gradeInfoForGBKWireWithNameTexts(grade StGradeInfo, names *webSocketGradeNameTexts) StGradeInfo {
	grade = gradeInfoForGBKWire(grade)
	if names == nil {
		return grade
	}
	writeGBKFixedTextSlotsFromStrings(grade.StrSizeGradeName[:], cTCPServerMaxTextLength, names.SizeGradeNames)
	writeGBKFixedTextSlotsFromStrings(grade.StrQualityGradeName[:], cTCPServerMaxTextLength, names.QualityGradeNames)
	return grade
}

func gradeInfoFromGBKWireForFrontend(grade StGradeInfo) StGradeInfo {
	rewriteGBKFixedTextSlotAsUTF8(grade.StrFruitName[:])
	rewriteGBKFixedTextSlotsAsUTF8(grade.StrSizeGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StrQualityGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StDensityGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StrColorGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StrShapeGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StFlawareaGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StBruiseGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StRotGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StSugarGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StAcidityGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StHollowGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StSkinGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StBrownGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StTangxinGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StRigidityGradeName[:], cTCPServerMaxTextLength)
	rewriteGBKFixedTextSlotsAsUTF8(grade.StWaterGradeName[:], cTCPServerMaxTextLength)
	return grade
}

func stGlobalFromGBKWireForFrontend(global StGlobal) StGlobal {
	global.Grade = gradeInfoFromGBKWireForFrontend(global.Grade)
	return global
}

func fixedTextStringsFromGBKWire(src []uint8, slotSize int, count int) []string {
	if slotSize <= 0 || count <= 0 {
		return nil
	}
	maxCount := len(src) / slotSize
	if count > maxCount {
		count = maxCount
	}
	names := make([]string, 0, count)
	for index := 0; index < count; index++ {
		start := index * slotSize
		text := decodeGBKFixedTextSlot(src[start : start+slotSize])
		names = append(names, text)
	}
	return names
}

func gradeNameTextsFromGBKWire(grade StGradeInfo) webSocketGradeNameTexts {
	sizeCount := int(grade.NSizeGradeNum)
	if sizeCount > cTCPServerMaxSizeGradeNum {
		sizeCount = cTCPServerMaxSizeGradeNum
	}
	qualityCount := int(grade.NQualityGradeNum)
	if qualityCount > cTCPServerMaxQualityGradeNum {
		qualityCount = cTCPServerMaxQualityGradeNum
	}
	return webSocketGradeNameTexts{
		SizeGradeNames:    fixedTextStringsFromGBKWire(grade.StrSizeGradeName[:], cTCPServerMaxTextLength, sizeCount),
		QualityGradeNames: fixedTextStringsFromGBKWire(grade.StrQualityGradeName[:], cTCPServerMaxTextLength, qualityCount),
	}
}
