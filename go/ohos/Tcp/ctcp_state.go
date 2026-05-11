package tcp

import (
	"encoding/json"
	"sync"
	"time"
)

var (
	cTCPConfigSnapshotMu sync.Mutex
	cTCPLastConfig       *CTCPConfigSnapshot

	cTCPStatisticsSnapshotMu sync.Mutex
	cTCPLastStatistics       *CTCPStatisticsSnapshot
)

type CTCPStatisticsSnapshot struct {
	ServerName string
	Port       int
	RemoteAddr string
	SrcID      int32
	DstID      int32
	CmdID      int32
	ReceivedAt int64
	Statistics StStatistics
}

func saveCTCPConfigSnapshot(serverName string, port int, remoteAddr string, head cTCPServerCommandHead, payload []byte, sysConfig StSysConfig, gradeInfo StGradeInfo) CTCPConfigSnapshot {
	payloadCopy := append([]byte(nil), payload...)
	snapshot := CTCPConfigSnapshot{
		ServerName: serverName,
		Port:       port,
		RemoteAddr: remoteAddr,
		SrcID:      head.NSrcId,
		DstID:      head.NDstId,
		CmdID:      head.NCmdId,
		ReceivedAt: time.Now().UnixMilli(),
		RawPayload: payloadCopy,
		SysConfig:  sysConfig,
		GradeInfo:  gradeInfo,
	}

	cTCPConfigSnapshotMu.Lock()
	cTCPLastConfig = &snapshot
	cTCPConfigSnapshotMu.Unlock()

	return snapshot
}

func LastCTCPConfigSnapshot() (CTCPConfigSnapshot, bool) {
	cTCPConfigSnapshotMu.Lock()
	defer cTCPConfigSnapshotMu.Unlock()
	if cTCPLastConfig == nil {
		return CTCPConfigSnapshot{}, false
	}

	snapshot := *cTCPLastConfig
	snapshot.RawPayload = append([]byte(nil), cTCPLastConfig.RawPayload...)
	return snapshot, true
}

func LastCTCPConfigJSON() string {
	snapshot, ok := LastCTCPConfigSnapshot()
	if !ok {
		return "{}"
	}
	text, err := formatCTCPConfigSnapshotJSON(snapshot)
	if err != nil {
		return "{}"
	}
	return text
}

func saveCTCPStatisticsSnapshot(serverName string, port int, remoteAddr string, head cTCPServerCommandHead, stats StStatistics) CTCPStatisticsSnapshot {
	snapshot := CTCPStatisticsSnapshot{
		ServerName: serverName,
		Port:       port,
		RemoteAddr: remoteAddr,
		SrcID:      head.NSrcId,
		DstID:      head.NDstId,
		CmdID:      head.NCmdId,
		ReceivedAt: time.Now().UnixMilli(),
		Statistics: stats,
	}

	cTCPStatisticsSnapshotMu.Lock()
	cTCPLastStatistics = &snapshot
	cTCPStatisticsSnapshotMu.Unlock()

	return snapshot
}

func LastCTCPStatisticsSnapshot() (CTCPStatisticsSnapshot, bool) {
	cTCPStatisticsSnapshotMu.Lock()
	defer cTCPStatisticsSnapshotMu.Unlock()
	if cTCPLastStatistics == nil {
		return CTCPStatisticsSnapshot{}, false
	}
	return *cTCPLastStatistics, true
}

func LastCTCPStatisticsJSON() string {
	snapshot, ok := LastCTCPStatisticsSnapshot()
	if !ok {
		return "{}"
	}
	text, err := formatCTCPStatisticsSnapshotJSON(snapshot)
	if err != nil {
		return "{}"
	}
	return text
}

// formatCTCPStatisticsSnapshotJSON 把 StStatistics 转换成不含 padding 字段的可读 JSON。
func formatCTCPStatisticsSnapshotJSON(snapshot CTCPStatisticsSnapshot) (string, error) {
	s := snapshot.Statistics

	type statisticsView struct {
		MaxExitNum            int      `json:"maxExitNum"`
		NGradeCount           []uint64 `json:"nGradeCount"`
		NWeightGradeCount     []uint64 `json:"nWeightGradeCount"`
		NExitCount            []uint64 `json:"nExitCount"`
		NExitWeightCount      []uint64 `json:"nExitWeightCount"`
		NChannelTotalCount    uint64   `json:"nChannelTotalCount"`
		NChannelWeightCount   uint64   `json:"nChannelWeightCount"`
		NSubsysId             int32    `json:"nSubsysId"`
		NBoxGradeCount        []int32  `json:"nBoxGradeCount"`
		NBoxGradeWeight       []int32  `json:"nBoxGradeWeight"`
		NTotalCupNum          int32    `json:"nTotalCupNum"`
		NInterval             int32    `json:"nInterval"`
		NIntervalSumperminute int32    `json:"nIntervalSumperminute"`
		NCupState             uint16   `json:"nCupState"`
		NPulseInterval        uint16   `json:"nPulseInterval"`
		NUnpushFruitCount     uint16   `json:"nUnpushFruitCount"`
		NNetState             uint8    `json:"nNetState"`
		NWeightSetting        uint8    `json:"nWeightSetting"`
		NSCMState             uint8    `json:"nSCMState"`
		NIQSNetState          uint8    `json:"nIQSNetState"`
		NLockState            uint8    `json:"nLockState"`
		ExitBoxNum            []uint32 `json:"exitBoxNum"`
	}

	view := struct {
		ServerName string         `json:"serverName"`
		Port       int            `json:"port"`
		RemoteAddr string         `json:"remoteAddr"`
		SrcID      int32          `json:"srcId"`
		DstID      int32          `json:"dstId"`
		CmdID      int32          `json:"cmdId"`
		ReceivedAt int64          `json:"receivedAt"`
		Size       int            `json:"size"`
		Statistics statisticsView `json:"statistics"`
	}{
		ServerName: snapshot.ServerName,
		Port:       snapshot.Port,
		RemoteAddr: snapshot.RemoteAddr,
		SrcID:      snapshot.SrcID,
		DstID:      snapshot.DstID,
		CmdID:      snapshot.CmdID,
		ReceivedAt: snapshot.ReceivedAt,
		Size:       stStatisticsExpectedSize,
		Statistics: statisticsView{
			MaxExitNum:            cTCPServerStatisticsMaxExitNum,
			NGradeCount:           s.NGradeCount[:],
			NWeightGradeCount:     s.NWeightGradeCount[:],
			NExitCount:            s.NExitCount[:],
			NExitWeightCount:      s.NExitWeightCount[:],
			NChannelTotalCount:    s.NChannelTotalCount,
			NChannelWeightCount:   s.NChannelWeightCount,
			NSubsysId:             s.NSubsysId,
			NBoxGradeCount:        s.NBoxGradeCount[:],
			NBoxGradeWeight:       s.NBoxGradeWeight[:],
			NTotalCupNum:          s.NTotalCupNum,
			NInterval:             s.NInterval,
			NIntervalSumperminute: s.NIntervalSumperminute,
			NCupState:             s.NCupState,
			NPulseInterval:        s.NPulseInterval,
			NUnpushFruitCount:     s.NUnpushFruitCount,
			NNetState:             s.NNetState,
			NWeightSetting:        s.NWeightSetting,
			NSCMState:             s.NSCMState,
			NIQSNetState:          s.NIQSNetState,
			NLockState:            s.NLockState,
			ExitBoxNum:            s.ExitBoxNum[:],
		},
	}

	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatCTCPConfigSnapshotJSON(snapshot CTCPConfigSnapshot) (string, error) {
	view := struct {
		ServerName      string      `json:"serverName"`
		Port            int         `json:"port"`
		RemoteAddr      string      `json:"remoteAddr"`
		SrcID           int32       `json:"srcId"`
		DstID           int32       `json:"dstId"`
		CmdID           int32       `json:"cmdId"`
		ReceivedAt      int64       `json:"receivedAt"`
		RawPayloadBytes int         `json:"rawPayloadBytes"`
		SysConfig       StSysConfig `json:"sysConfig"`
		GradeInfo       StGradeInfo `json:"gradeInfo"`
	}{
		ServerName:      snapshot.ServerName,
		Port:            snapshot.Port,
		RemoteAddr:      snapshot.RemoteAddr,
		SrcID:           snapshot.SrcID,
		DstID:           snapshot.DstID,
		CmdID:           snapshot.CmdID,
		ReceivedAt:      snapshot.ReceivedAt,
		RawPayloadBytes: len(snapshot.RawPayload),
		SysConfig:       snapshot.SysConfig,
		GradeInfo:       snapshot.GradeInfo,
	}

	data, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
