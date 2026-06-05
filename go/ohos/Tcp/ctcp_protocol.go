package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

func recvCTCPSync(conn net.Conn) error {
	data := make([]byte, 4)
	if _, err := io.ReadFull(conn, data); err != nil {
		return err
	}
	if string(data) != "SYNC" {
		return fmt.Errorf("invalid sync %q", string(data))
	}
	return nil
}

func recvCTCPCommand(conn net.Conn) (cTCPServerCommandHead, error) {
	data := make([]byte, 12)
	if _, err := io.ReadFull(conn, data); err != nil {
		return cTCPServerCommandHead{}, err
	}

	head := cTCPServerCommandHead{
		NSrcId: int32(binary.LittleEndian.Uint32(data[0:4])),
		NDstId: int32(binary.LittleEndian.Uint32(data[4:8])),
		NCmdId: int32(binary.LittleEndian.Uint32(data[8:12])),
	}
	if head.NSrcId == -1 || head.NCmdId == -1 {
		return cTCPServerCommandHead{}, errors.New("invalid command head")
	}
	return head, nil
}

func recvCTCPPayload(conn net.Conn, head cTCPServerCommandHead) ([]byte, int, string, error) {
	if cTCPCommandHasPacketLength(head.NCmdId) {
		lengthData := make([]byte, 4)
		if _, err := io.ReadFull(conn, lengthData); err != nil {
			return nil, 0, "packet-length", err
		}

		length := int(int32(binary.LittleEndian.Uint32(lengthData)))
		if length < 0 || length > cTCPServerMaxPayload {
			return nil, len(lengthData), "packet-length", fmt.Errorf("invalid packet length %d", length)
		}

		payload := make([]byte, length)
		if length > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				return payload, len(lengthData), "packet-length", err
			}
		}
		return payload, len(lengthData) + len(payload), "packet-length", nil
	}

	if length, ok := cTCPKnownPayloadLength(head.NCmdId); ok {
		if length == 0 {
			return nil, 0, "known-zero", nil
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return payload, len(payload), "known-length", err
		}
		return payload, len(payload), "known-length", nil
	}

	payload, err := readUntilIdle(conn)
	return payload, len(payload), "read-until-idle", err
}

func cTCPCommandHasPacketLength(cmd int32) bool {
	switch cmd {
	case cmdIPMImageSpot, cmdIPMImage, cmdIPMImageSplice, cmdACSExitStop:
		return true
	default:
		return false
	}
}

func cTCPKnownPayloadLength(cmd int32) (int, bool) {
	switch cmd {
	case cmdFSMGetVersion, cmdWAMVersionInfo:
		return 64, true
	case cmdFSMVersionError, cmdFSMBurnFlashProgress, cmdFSMBootFlashProgress:
		return 4, true
	case cmdSIMDisplayOn, cmdSIMInspectionOff:
		return 0, true
	default:
		return 0, false
	}
}

func readUntilIdle(conn net.Conn) ([]byte, error) {
	buffer := make([]byte, 4096)
	payload := make([]byte, 0, 4096)

	for {
		_ = conn.SetReadDeadline(time.Now().Add(cTCPServerIdleReadTimeout))
		n, err := conn.Read(buffer)
		if n > 0 {
			payload = append(payload, buffer[:n]...)
		}
		if err == nil {
			continue
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return payload, nil
		}
		if errors.Is(err, io.EOF) {
			return payload, nil
		}
		return payload, err
	}
}
