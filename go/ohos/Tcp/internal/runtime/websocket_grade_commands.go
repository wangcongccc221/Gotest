package tcp

import (
	"errors"
	"fmt"
	"strings"

	"gotest/ohos/Tcp/state"
)

func DragLevelData(control webSocketControlMessage) (int, int32, int) {
	if control.Grade != nil {
		return SendGradeInfoData("dropdata", cTCPHCGradeInfo, control)
	}

	destID := normalizeDropDataDestID(control)
	if len(control.DropGrades) == 0 && len(control.GradeExits) == 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: empty gradeExits, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket dropdata failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	if len(control.DropGrades) > 0 {
		if err := applyGradeDropAction(&grade, control.Action, control.ExitNo, control.DropGrades); err != nil {
			setCTCPServerLastMessage("WebSocket dropdata failed: %v", err)
			return -1, destID, 0
		}
	} else if err := applyGradeExitMapping(&grade, control.GradeExits); err != nil {
		setCTCPServerLastMessage("WebSocket dropdata failed: %v", err)
		return -1, destID, 0
	}
	setCTCPServerLastMessage(
		"WebSocket dropdata exitLOW/exitHight after apply: dest=0x%04X, action=%s, exitNo=%d, dropGrades=%d, gradeExits=%d, exitBits=%s",
		uint32(destID),
		control.Action,
		control.ExitNo,
		len(control.DropGrades),
		len(control.GradeExits),
		summarizeGradeExitLowHigh(grade),
	)

	payload, err := encodeGradeInfoPayloadWithNameTexts(grade, control.GradeNameTexts)
	if err != nil {
		setCTCPServerLastMessage("WebSocket dropdata failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket dropdata: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExits=%s, byExit=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket dropdata failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	requestStGlobalAfterConfigCommand("dropdata", destID)
	setCTCPServerLastMessage("WebSocket dropdata success: HC_CMD_GRADE_INFO sent, dest=0x%04X, refresh StGlobal scheduled", uint32(destID))
	return 0, destID, len(payload)
}

func ClearGradeExitData(control webSocketControlMessage) (int, int32, int) { //清空数据
	destID := normalizeDropDataDestID(control)
	grade, ok := latestGradeInfo(destID)
	if !ok {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: no cached StGradeInfo, dest=0x%04X", uint32(destID))
		return -1, destID, 0
	}

	before := summarizeGradeExitMappings(grade)
	beforeByExit := summarizeGradeExitMappingsByExit(grade)
	beforeLowHigh := summarizeGradeExitLowHigh(grade)
	clearGradeExitMappings(&grade)
	setCTCPServerLastMessage(
		"WebSocket clearExitGrades exitLOW/exitHight: dest=0x%04X, before=%s, after=%s",
		uint32(destID),
		beforeLowHigh,
		summarizeGradeExitLowHigh(grade),
	)

	payload, err := encodeGradeInfoPayloadWithNameTexts(grade, control.GradeNameTexts)
	if err != nil {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: encode StGradeInfo: %v", err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, cTCPHCGradeInfo, "", 0)
	setCTCPServerLastMessage(
		"WebSocket clearExitGrades: sending HC_CMD_GRADE_INFO(0x%04X), dest=0x%04X, target=%s:%d, payload=%d bytes, activeExitsBefore=%s, byExitBefore=%s, activeExitsAfter=%s, byExitAfter=%s",
		uint32(cTCPHCGradeInfo),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		before,
		beforeByExit,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, cTCPHCGradeInfo, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket clearExitGrades failed: HC_CMD_GRADE_INFO result=%d", result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	setCTCPServerLastMessage("WebSocket clearExitGrades success: HC_CMD_GRADE_INFO sent, dest=0x%04X", uint32(destID))
	return 0, destID, len(payload)
}

func SendGradeInfoData(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) { //发送等级信息数据
	destID := normalizeDropDataDestID(control)
	if commandID == cTCPHCGradeInfo && control.LevelAuxConfig != nil {
		if err := saveLevelAuxConfigFromControl(control.FSMID, *control.LevelAuxConfig); err != nil {
			setCTCPServerLastMessage("WebSocket %s failed: save levelAuxConfig: %v", topic, err)
			return -1, destID, 0
		}
	}
	if control.Grade == nil {
		setCTCPServerLastMessage("WebSocket %s failed: empty StGradeInfo, dest=0x%04X", topic, uint32(destID))
		return -1, destID, 0
	}

	grade := *control.Grade
	if commandID == cTCPHCColorGradeInfo {
		logQualityParameterFields("下发 saveQualityData", destID, grade)
	}
	setCTCPServerLastMessage(
		"WebSocket %s received StGradeInfo: dest=0x%04X, sizeGrade=%d, qualityGrade=%d, activeExits=%s, byExit=%s",
		topic,
		uint32(destID),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)
	setCTCPServerLastMessage(
		"WebSocket %s received exitLOW/exitHight: dest=0x%04X, sizeGrade=%d, qualityGrade=%d, exitBits=%s",
		topic,
		uint32(destID),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitLowHigh(grade),
	)
	if commandID == cTCPHCGradeInfo {
		if cached, ok := latestGradeInfo(destID); ok {
			if shouldPreserveCachedGradeExits(grade, cached, control.PreserveCachedGradeExits) {
				mergeGradeExitState(&grade, cached)
				setCTCPServerLastMessage(
					"WebSocket %s merged cached exit state: dest=0x%04X, activeExits=%s, byExit=%s",
					topic,
					uint32(destID),
					summarizeGradeExitMappings(grade),
					summarizeGradeExitMappingsByExit(grade),
				)
				setCTCPServerLastMessage(
					"WebSocket %s merged cached exitLOW/exitHight: dest=0x%04X, exitBits=%s",
					topic,
					uint32(destID),
					summarizeGradeExitLowHigh(grade),
				)
			}
		}
	}

	payload, err := encodeGradeInfoPayloadWithNameTexts(grade, control.GradeNameTexts)
	if err != nil {
		setCTCPServerLastMessage("WebSocket %s failed: encode StGradeInfo: %v", topic, err)
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, sizeGrade=%d, qualityGrade=%d, activeExits=%s, byExit=%s",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		grade.NSizeGradeNum,
		grade.NQualityGradeNum,
		summarizeGradeExitMappings(grade),
		summarizeGradeExitMappingsByExit(grade),
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	cacheLatestGradeInfo(destID, grade)
	requestStGlobalAfterConfigCommand(topic, destID)
	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X, refresh StGlobal scheduled", topic, uint32(commandID), uint32(destID))
	return 0, destID, len(payload)
}

func cacheLatestGradeInfo(destID int32, grade StGradeInfo) {
	setLastStGradeInfoSnapshot(grade)
	state.SetGradeInfoForDest(destID, grade)
}

func latestGradeInfo(destID int32) (StGradeInfo, bool) {
	return state.LatestGradeInfoForDest(destID)
}

func summarizeGradeExitMappings(grade StGradeInfo) string {
	const maxSummaryItems = 12

	parts := make([]string, 0, maxSummaryItems)
	total := 0
	for index, item := range grade.Grades {
		if item.ExitLow == 0 && item.ExitHigh == 0 {
			continue
		}
		total++
		if len(parts) < maxSummaryItems {
			parts = append(parts, fmt.Sprintf(
				"%d:low=0x%08X,high=0x%08X,exit=%d",
				index,
				item.ExitLow,
				item.ExitHigh,
				item.Exit(),
			))
		}
	}
	if total == 0 {
		return "none"
	}
	if total > len(parts) {
		return fmt.Sprintf("%s (+%d more)", strings.Join(parts, ","), total-len(parts))
	}
	return strings.Join(parts, ",")
}

func summarizeGradeExitLowHigh(grade StGradeInfo) string {
	const maxSummaryItems = 16

	parts := make([]string, 0, maxSummaryItems)
	total := 0
	for index, item := range grade.Grades {
		if item.ExitLow == 0 && item.ExitHigh == 0 {
			continue
		}
		total++
		if len(parts) < maxSummaryItems {
			parts = append(parts, fmt.Sprintf(
				"%d:exitLOW=0x%08X,exitHight=0x%08X,exit64=%d",
				index,
				item.ExitLow,
				item.ExitHigh,
				item.Exit(),
			))
		}
	}
	if total == 0 {
		return "none"
	}
	if total > len(parts) {
		return fmt.Sprintf("%s (+%d more)", strings.Join(parts, ","), total-len(parts))
	}
	return strings.Join(parts, ",")
}

func summarizeGradeExitMappingsByExit(grade StGradeInfo) string {
	const maxExitSummaryItems = 12
	const maxGradeSummaryItems = 16

	parts := make([]string, 0, maxExitSummaryItems)
	for exitNo := 1; exitNo <= 48; exitNo++ {
		mask := uint64(1) << uint(exitNo-1)
		gradeIndexes := make([]string, 0, maxGradeSummaryItems)
		total := 0
		for gradeIndex, item := range grade.Grades {
			if item.Exit()&mask == 0 {
				continue
			}
			total++
			if len(gradeIndexes) < maxGradeSummaryItems {
				gradeIndexes = append(gradeIndexes, fmt.Sprintf("%d", gradeIndex))
			}
		}
		if total == 0 {
			continue
		}
		if total > len(gradeIndexes) {
			gradeIndexes = append(gradeIndexes, fmt.Sprintf("+%d", total-len(gradeIndexes)))
		}
		parts = append(parts, fmt.Sprintf("exit%d=[%s]", exitNo, strings.Join(gradeIndexes, "|")))
		if len(parts) >= maxExitSummaryItems {
			break
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}

func normalizeDropDataDestID(control webSocketControlMessage) int32 { //标准化目标ID
	if control.DestID != 0 {
		return control.DestID
	}
	if control.FSMID != 0 {
		return control.FSMID
	}
	return cTCPDefaultFSMID
}

func normalizeValveTestDestID(control webSocketControlMessage) int32 {
	if control.DestID != 0 {
		return control.DestID
	}
	fsmID := control.FSMID
	if fsmID == 0 {
		fsmID = cTCPDefaultFSMID
	}
	channelIndex := 0
	if control.ChannelIndex != nil {
		channelIndex = *control.ChannelIndex
	}
	if channelIndex < 0 {
		channelIndex = 0
	}
	if channelIndex >= cTCPMaxChannelNum {
		channelIndex = cTCPMaxChannelNum - 1
	}
	subsysIndex := getSubsysIndex(fsmID)
	if subsysIndex < 0 {
		subsysIndex = 0
	}
	return encodeChannel(subsysIndex, channelIndex, channelIndex)
}

func applyGradeDropAction(grade *StGradeInfo, action string, exitNo int, grades []webSocketDropGrade) error {
	add, err := normalizeDropAction(action)
	if err != nil {
		return err
	}
	if exitNo < 1 || exitNo > 64 {
		return fmt.Errorf("exitNo out of range: %d", exitNo)
	}
	if len(grades) == 0 {
		return errors.New("empty drop grades")
	}

	op := "add"
	if !add {
		op = "remove"
	}
	for _, item := range grades {
		index, err := resolveDropGradeIndex(item)
		if err != nil {
			return err
		}
		beforeLow := grade.Grades[index].ExitLow
		beforeHigh := grade.Grades[index].ExitHigh
		applyGradeExitBit(&grade.Grades[index], exitNo, add)
		setCTCPServerLastMessage(
			"WebSocket dropdata exitLOW/exitHight item: action=%s, exitNo=%d, gradeIndex=%d, %s, name=%q, beforeLOW=0x%08X, beforeHight=0x%08X, afterLOW=0x%08X, afterHight=0x%08X",
			op,
			exitNo,
			index,
			formatDropGradeCoord(item),
			item.Name,
			beforeLow,
			beforeHigh,
			grade.Grades[index].ExitLow,
			grade.Grades[index].ExitHigh,
		)
	}
	if add {
		enableGradeExit(grade, exitNo)
	}
	return nil
}

func normalizeDropAction(action string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", "add", "append", "up":
		return true, nil
	case "remove", "delete", "del", "down":
		return false, nil
	default:
		return false, fmt.Errorf("unknown drop action: %s", action)
	}
}

func resolveDropGradeIndex(item webSocketDropGrade) (int, error) {
	if item.Row != nil || item.Col != nil {
		if item.Row == nil || item.Col == nil {
			return 0, errors.New("grade row/col must be provided together")
		}
		if *item.Row < 0 || *item.Row >= cTCPServerMaxQualityGradeNum {
			return 0, fmt.Errorf("grade row out of range: %d", *item.Row)
		}
		if *item.Col < 0 || *item.Col >= cTCPServerMaxSizeGradeNum {
			return 0, fmt.Errorf("grade col out of range: %d", *item.Col)
		}
		return *item.Row*cTCPServerMaxSizeGradeNum + *item.Col, nil
	}
	if item.Index != nil {
		if *item.Index < 0 || *item.Index >= cTCPServerMaxQualityGradeNum*cTCPServerMaxSizeGradeNum {
			return 0, fmt.Errorf("grade index out of range: %d", *item.Index)
		}
		return *item.Index, nil
	}
	return 0, errors.New("grade row/col or index is required")
}

func formatDropGradeCoord(item webSocketDropGrade) string {
	if item.Row != nil && item.Col != nil {
		return fmt.Sprintf("row=%d, col=%d", *item.Row, *item.Col)
	}
	if item.Index != nil {
		return fmt.Sprintf("index=%d", *item.Index)
	}
	return "row=-, col=-"
}

func applyGradeExitBit(item *StGradeItemInfo, exitNo int, add bool) {
	mask := uint64(1) << uint(exitNo-1)
	exit64 := uint64(item.ExitLow) | (uint64(item.ExitHigh) << 32)
	if add {
		exit64 |= mask
	} else {
		exit64 &^= mask
	}
	item.ExitLow = uint32(exit64)
	item.ExitHigh = uint32(exit64 >> 32)
}

func clearGradeExitMappings(grade *StGradeInfo) {
	if grade == nil {
		return
	}
	for i := range grade.Grades {
		grade.Grades[i].ExitLow = 0
		grade.Grades[i].ExitHigh = 0
	}
}

func mergeGradeExitState(target *StGradeInfo, cached StGradeInfo) {
	for i := 0; i < len(target.Grades) && i < len(cached.Grades); i++ {
		target.Grades[i].ExitLow |= cached.Grades[i].ExitLow
		target.Grades[i].ExitHigh |= cached.Grades[i].ExitHigh
	}
	for i := 0; i < len(target.ExitEnabled) && i < len(cached.ExitEnabled); i++ {
		target.ExitEnabled[i] = int32(uint32(target.ExitEnabled[i]) | uint32(cached.ExitEnabled[i]))
	}
}

func shouldPreserveCachedGradeExits(target StGradeInfo, cached StGradeInfo, explicitPreserve *bool) bool {
	if explicitPreserve != nil {
		return *explicitPreserve
	}
	return target.NSizeGradeNum == cached.NSizeGradeNum &&
		target.NQualityGradeNum == cached.NQualityGradeNum &&
		target.NClassifyType == cached.NClassifyType
}

func enableGradeExit(grade *StGradeInfo, exitNo int) {
	if exitNo < 1 || exitNo > 64 {
		return
	}
	bucket := 0
	shift := exitNo - 1
	if shift >= 32 {
		bucket = 1
		shift -= 32
	}
	grade.ExitEnabled[bucket] = int32(uint32(grade.ExitEnabled[bucket]) | (uint32(1) << uint(shift)))
}

func applyGradeExitMapping(grade *StGradeInfo, exits []webSocketGradeExit) error {
	for _, item := range exits {
		if item.Index < 0 || item.Index >= len(grade.Grades) {
			return fmt.Errorf("grade index out of range: %d", item.Index)
		}

		exitMask := uint64(item.Exit)
		current := grade.Grades[item.Index].Exit()
		next := current | exitMask
		beforeLow := grade.Grades[item.Index].ExitLow
		beforeHigh := grade.Grades[item.Index].ExitHigh
		grade.Grades[item.Index].ExitLow = uint32(next)
		grade.Grades[item.Index].ExitHigh = uint32(next >> 32)
		setCTCPServerLastMessage(
			"WebSocket gradeExits exitLOW/exitHight item: gradeIndex=%d, exitMask=%d, beforeLOW=0x%08X, beforeHight=0x%08X, afterLOW=0x%08X, afterHight=0x%08X",
			item.Index,
			exitMask,
			beforeLow,
			beforeHigh,
			grade.Grades[item.Index].ExitLow,
			grade.Grades[item.Index].ExitHigh,
		)
	}
	return nil
}
