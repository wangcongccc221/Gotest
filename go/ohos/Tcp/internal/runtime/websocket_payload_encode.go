package tcp

import (
	"fmt"
	"unsafe"
)

func encodeGradeInfoPayload(grade StGradeInfo) ([]byte, error) {
	return encodeGradeInfoPayloadWithNameTexts(grade, nil)
}

func encodeGradeInfoPayloadWithNameTexts(grade StGradeInfo, names *webSocketGradeNameTexts) ([]byte, error) {
	grade = gradeInfoForGBKWireWithNameTexts(grade, names)
	size := int(unsafe.Sizeof(grade))
	if size != cTCP48StGradeInfoWireSize {
		return nil, fmt.Errorf("sizeof(StGradeInfo)=%d, expected=%d", size, cTCP48StGradeInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&grade)), size)
	copy(payload, src)
	return payload, nil
}

func encodeParasInfoPayload(paras StParas) ([]byte, error) {
	size := int(unsafe.Sizeof(paras))
	if size != cTCP48StParasWireSize {
		return nil, fmt.Errorf("sizeof(StParas)=%d, expected=%d", size, cTCP48StParasWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&paras)), size)
	copy(payload, src)
	return payload, nil
}

func encodeMotorInfoPayload(motor StMotorInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(motor))
	if size != cTCP48StMotorInfoWireSize {
		return nil, fmt.Errorf("sizeof(StMotorInfo)=%d, expected=%d", size, cTCP48StMotorInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&motor)), size)
	copy(payload, src)
	return payload, nil
}

func encodeGlobalExitInfoPayload(globalExitInfo StGlobalExitInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(globalExitInfo))
	if size != cTCP48StGlobalExitInfoSize {
		return nil, fmt.Errorf("sizeof(StGlobalExitInfo)=%d, expected=%d", size, cTCP48StGlobalExitInfoSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&globalExitInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeAnalogDensityPayload(densityInfo StAnalogDensity) ([]byte, error) {
	size := int(unsafe.Sizeof(densityInfo))
	expected := cTCP48AnalogDensitySlots * 4
	if size != expected {
		return nil, fmt.Errorf("sizeof(StAnalogDensity)=%d, expected=%d", size, expected)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&densityInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeWeightBaseInfoPayload(weightInfo StWeightBaseInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(weightInfo))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&weightInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeGlobalWeightBaseInfoPayload(globalWeightInfo StGlobalWeightBaseInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(globalWeightInfo))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&globalWeightInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeResetADPayload(resetAD StResetAD) ([]byte, error) {
	size := int(unsafe.Sizeof(resetAD))
	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&resetAD)), size)
	copy(payload, src)
	return payload, nil
}

func encodeExitInfoPayload(exitInfo StExitInfo) ([]byte, error) {
	size := int(unsafe.Sizeof(exitInfo))
	if size != cTCP48StExitInfoWireSize {
		return nil, fmt.Errorf("sizeof(StExitInfo)=%d, expected=%d", size, cTCP48StExitInfoWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&exitInfo)), size)
	copy(payload, src)
	return payload, nil
}

func encodeSysConfigPayload(sysConfig StSysConfig) ([]byte, error) {
	size := int(unsafe.Sizeof(sysConfig))
	if size != cTCP48StSysConfigWireSize {
		return nil, fmt.Errorf("sizeof(StSysConfig)=%d, expected=%d", size, cTCP48StSysConfigWireSize)
	}

	payload := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(&sysConfig)), size)
	copy(payload, src)
	return payload, nil
}
