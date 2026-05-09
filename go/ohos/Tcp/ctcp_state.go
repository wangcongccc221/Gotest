package tcp

import (
	"encoding/json"
	"sync"
	"time"
)

var (
	cTCPConfigSnapshotMu sync.Mutex
	cTCPLastConfig       *CTCPConfigSnapshot
)

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
