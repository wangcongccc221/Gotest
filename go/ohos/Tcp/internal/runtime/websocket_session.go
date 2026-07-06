package tcp

import (
	"log"
	"sync"
)

type offCommandEntry struct {
	commandID int32
	destID    int32
}

type webSocketSessionState struct {
	mu      sync.Mutex
	pending map[int32]offCommandEntry
}

func newWebSocketSessionState() *webSocketSessionState {
	return &webSocketSessionState{
		pending: make(map[int32]offCommandEntry),
	}
}

// recordOnCommand 记录一个需要断开时自动关闭的 On 命令。
// onCommandID 是 On 命令的 ID（用作去重 key），offCommandID 是对应的 Off 命令 ID。
func (s *webSocketSessionState) recordOnCommand(onCommandID int32, offCommandID int32, destID int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[onCommandID] = offCommandEntry{
		commandID: offCommandID,
		destID:    destID,
	}
}

// clearOnCommand 在前端主动发 Off 命令时移除记录，避免断开时重复发送。
func (s *webSocketSessionState) clearOnCommand(onCommandID int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pending, onCommandID)
}

// flushOffCommands 在客户端断开时，对所有未关闭的 On 命令发送对应的 Off 命令到下位机。
func (s *webSocketSessionState) flushOffCommands() {
	s.mu.Lock()
	entries := make([]offCommandEntry, 0, len(s.pending))
	for _, entry := range s.pending {
		entries = append(entries, entry)
	}
	s.pending = make(map[int32]offCommandEntry)
	s.mu.Unlock()

	if len(entries) == 0 {
		log.Printf("[WS断开清理] 无未关闭的 On 命令")
		return
	}

	for _, entry := range entries {
		targetIP, targetPort := resolveCTCPTarget(entry.destID, entry.commandID, "", 0)
		result := StartCTCPClient(targetIP, targetPort, entry.destID, entry.commandID, nil)
		log.Printf("[WS断开清理] 自动发送 Off 命令 cmd=0x%04X, dest=0x%04X, target=%s:%d, result=%d",
			uint32(entry.commandID),
			uint32(entry.destID),
			targetIP,
			targetPort,
			result,
		)
	}
}

// recordWAMChannelOn 记录 WAM 通道级 On 命令（如数据追踪、波形捕捉）。
func (c *webSocketClient) recordWAMChannelOn(onCommandID int32, offCommandID int32, control webSocketControlMessage) {
	destID := normalizeWAMChannelDestID(control)
	c.session.recordOnCommand(onCommandID, offCommandID, destID)
}

// recordWAMOn 记录 WAM 子系统级 On 命令（如果杯测试、模拟脉冲）。
func (c *webSocketClient) recordWAMOn(onCommandID int32, offCommandID int32, control webSocketControlMessage) {
	destID := normalizeWAMDestID(control)
	c.session.recordOnCommand(onCommandID, offCommandID, destID)
}

// recordFSMOn 记录 FSM/IPM 级 On 命令（如果杯测试、水果分级、连续采集、快门调节）。
func (c *webSocketClient) recordFSMOn(onCommandID int32, offCommandID int32, control webSocketControlMessage) {
	destID := normalizeDropDataDestID(control)
	c.session.recordOnCommand(onCommandID, offCommandID, destID)
}
