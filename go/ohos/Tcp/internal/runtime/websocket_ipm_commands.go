package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func SendIpmCameraCommand(topic string, commandID int32, control webSocketControlMessage) (int, int32, int) {
	destID := normalizeDropDataDestID(control)
	payload, cameraNum, err := encodeIpmCameraCommandPayload(commandID, control)
	if err != nil {
		setCTCPServerLastMessage("WebSocket %s failed: %v, dest=0x%04X", topic, err, uint32(destID))
		return -1, destID, 0
	}

	targetIP, targetPort := resolveCTCPTarget(destID, commandID, "", 0)
	setCTCPServerLastMessage(
		"WebSocket %s: sending cmd=0x%04X, dest=0x%04X, target=%s:%d, payload=%d bytes, camera=%d",
		topic,
		uint32(commandID),
		uint32(destID),
		targetIP,
		targetPort,
		len(payload),
		cameraNum,
	)

	result := StartCTCPClient(targetIP, targetPort, destID, commandID, payload)
	if result != 0 {
		setCTCPServerLastMessage("WebSocket %s failed: cmd=0x%04X result=%d", topic, uint32(commandID), result)
		return result, destID, len(payload)
	}

	setCTCPServerLastMessage("WebSocket %s success: cmd=0x%04X sent, dest=0x%04X, camera=%d", topic, uint32(commandID), uint32(destID), cameraNum)
	return 0, destID, len(payload)
}

func encodeIpmCameraCommandPayload(commandID int32, control webSocketControlMessage) ([]byte, int, error) {
	switch commandID {
	case cTCPHCSingleSample, cTCPHCSingleSampleSpot, cTCPHCIpmShutdown, cTCPHCShutterAdjustOn, cTCPHCShutterAdjustOff:
		return nil, -1, nil
	case cTCPHCAutoBalanceOnCamera, cTCPHCAutoBalanceOn:
		payload, err := encodeIpmWhiteBalancePayload(control.WhiteBalancePayload)
		if err != nil {
			return nil, -1, err
		}
		return payload, int(payload[20]), nil
	case cTCPHCContinuousSampleOn:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		payload := []byte{byte(cameraNum), 0}
		if control.EvenShow != nil && *control.EvenShow {
			payload[1] = 1
		}
		return payload, cameraNum, nil
	case cTCPHCContinuousSampleOff:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		return []byte{byte(cameraNum)}, cameraNum, nil
	case cTCPHCShowBlobOn:
		cameraNum, err := requireIpmCameraNum(control)
		if err != nil {
			return nil, cameraNum, err
		}
		payload := make([]byte, 4)
		binary.LittleEndian.PutUint32(payload, uint32(cameraNum))
		return payload, cameraNum, nil
	default:
		return nil, -1, fmt.Errorf("unsupported IPM camera command: 0x%04X", uint32(commandID))
	}
}

func requireIpmCameraNum(control webSocketControlMessage) (int, error) {
	if control.CameraNum == nil {
		return 0, errors.New("empty cameraNum")
	}
	cameraNum := *control.CameraNum
	if cameraNum < 0 || cameraNum >= cTCPServerMaxCameraNum {
		return cameraNum, fmt.Errorf("cameraNum out of range: %d", cameraNum)
	}
	return cameraNum, nil
}

var arpMACPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{2}([-:][0-9a-f]{2}){5}\b`)

func lookupRemoteMAC(ip string) (string, error) {
	ip = strings.TrimSpace(ip)
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP: %q", ip)
	}
	setCTCPServerLastMessage("WebSocket ipmGetMac: ping %s", ip)
	if !NewCTcpClient().Ping(ip) {
		return "", fmt.Errorf("ping failed: %s", ip)
	}
	setCTCPServerLastMessage("WebSocket ipmGetMac: arp -a %s", ip)
	output, err := exec.Command("arp", "-a", ip).CombinedOutput()
	mac, ok := parseARPOutputMAC(string(output), ip)
	if ok {
		return mac, nil
	}
	if err != nil {
		return "", fmt.Errorf("arp failed: %v", err)
	}
	return "", fmt.Errorf("mac not found: %s", ip)
}

func parseARPOutputMAC(output string, ip string) (string, bool) {
	ip = strings.TrimSpace(ip)
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, ip) {
			continue
		}
		mac := arpMACPattern.FindString(line)
		if mac == "" {
			continue
		}
		return normalizeMACForDisplay(mac), true
	}
	return "", false
}

func normalizeMACForDisplay(mac string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(mac), ":", "-"))
}

func buildWakeOnLANPacket(mac string) ([]byte, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(mac), "-", ":")
	parts := strings.Split(normalized, ":")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid MAC: %q", mac)
	}
	macBytes := make([]byte, 6)
	for i, part := range parts {
		if len(part) != 2 {
			return nil, fmt.Errorf("invalid MAC: %q", mac)
		}
		value, err := strconv.ParseUint(part, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid MAC: %q", mac)
		}
		macBytes[i] = byte(value)
	}
	packet := make([]byte, 6+16*6)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for offset := 6; offset < len(packet); offset += 6 {
		copy(packet[offset:offset+6], macBytes)
	}
	return packet, nil
}

func subnetBroadcastForIP(ip string) string {
	parsed := net.ParseIP(strings.TrimSpace(ip)).To4()
	if parsed == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.255", parsed[0], parsed[1], parsed[2])
}

func sendWakeOnLAN(ip string, mac string) (int, error) {
	ip = strings.TrimSpace(ip)
	if net.ParseIP(ip) == nil {
		return 0, fmt.Errorf("invalid IP: %q", ip)
	}
	packet, err := buildWakeOnLANPacket(mac)
	if err != nil {
		return 0, err
	}
	targets := []string{subnetBroadcastForIP(ip), "255.255.255.255"}
	sentCount := 0
	var lastErr error
	for _, target := range targets {
		if target == "" {
			continue
		}
		targetSentCount := 0
		udpAddr := &net.UDPAddr{IP: net.ParseIP(target), Port: 9}
		for retry := 0; retry < 3; retry++ {
			conn, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				lastErr = err
				setCTCPServerLastMessage("WebSocket ipmWakeOnLan: dial failed, target=%s:9, retry=%d, error=%v", target, retry+1, err)
				continue
			}
			_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
			n, err := conn.Write(packet)
			_ = conn.Close()
			if err != nil {
				lastErr = err
				setCTCPServerLastMessage("WebSocket ipmWakeOnLan: write failed, target=%s:9, retry=%d, error=%v", target, retry+1, err)
				continue
			}
			if n == len(packet) {
				sentCount++
				targetSentCount++
			}
		}
		setCTCPServerLastMessage("WebSocket ipmWakeOnLan: target=%s:9 sentPackets=%d/3", target, targetSentCount)
	}
	if sentCount == 0 {
		if lastErr != nil {
			return 0, lastErr
		}
		return 0, errors.New("wol packet not sent")
	}
	return sentCount, nil
}

func encodeIpmWhiteBalancePayload(values []int) ([]byte, error) {
	const whiteBalancePayloadBytes = 24
	if len(values) != whiteBalancePayloadBytes {
		return nil, fmt.Errorf("whiteBalancePayload length=%d, want %d", len(values), whiteBalancePayloadBytes)
	}
	payload := make([]byte, whiteBalancePayloadBytes)
	for i, value := range values {
		if value < 0 || value > 255 {
			return nil, fmt.Errorf("whiteBalancePayload[%d] out of byte range: %d", i, value)
		}
		payload[i] = byte(value)
	}
	return payload, nil
}
