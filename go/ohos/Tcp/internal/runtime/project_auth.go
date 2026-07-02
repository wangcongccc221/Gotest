package tcp

import (
	"strings"

	"gotest/ohos/database"
)

func (c *webSocketClient) handleCreateProjectPassword(control webSocketControlMessage) {
	phone := strings.TrimSpace(control.Phone)
	err := database.CreateProjectPassword(phone)
	if err != nil {
		setCTCPServerLastMessage("WebSocket createProjectPassword failed: %v", err)
		c.sendCommandAckDetail("createProjectPassword", 0, 0, 0, -1, err.Error(), control.RequestID)
		return
	}
	setCTCPServerLastMessage("WebSocket createProjectPassword success: phoneLen=%d", len(phone))
	c.sendCommandAckDetail("createProjectPassword", 0, 0, 0, 0, "sent", control.RequestID)
}

func (c *webSocketClient) handleValidateProjectPassword(control webSocketControlMessage) {
	phone := strings.TrimSpace(control.Phone)
	validateCode := strings.TrimSpace(control.ValidateCode)
	err := database.ValidateProjectPassword(phone, validateCode)
	if err != nil {
		setCTCPServerLastMessage("WebSocket validateProjectPassword failed: %v", err)
		c.sendCommandAckDetail("validateProjectPassword", 0, 0, 0, -1, err.Error(), control.RequestID)
		return
	}
	setCTCPServerLastMessage("WebSocket validateProjectPassword success: phoneLen=%d", len(phone))
	c.sendCommandAckDetail("validateProjectPassword", 0, 0, 0, 0, "sent", control.RequestID)
}
