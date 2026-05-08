package modbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	modbusServerHost = "0.0.0.0"
	modbusServerPort = 5020
	modbusServerAddr = "0.0.0.0:5020"
)

var (
	modbusServerMu       sync.Mutex
	modbusListener       net.Listener
	modbusServerCancel   context.CancelFunc
	modbusServerRunning  bool
	modbusHoldingMu      sync.RWMutex
	modbusHoldingRecords = defaultModbusHoldingRegisters()
)

func StartModbusServer() int {
	return startModbusServer()
}

func StopModbusServer() int {
	return stopModbusServer()
}

func defaultModbusHoldingRegisters() map[uint16]uint16 {
	return map[uint16]uint16{
		0: 100,
		1: 200,
		2: 300,
		3: 400,
		4: 500,
	}
}

func startModbusServer() int {
	modbusServerMu.Lock()
	defer modbusServerMu.Unlock()

	if modbusServerRunning {
		return modbusServerPort
	}

	listener, err := net.Listen("tcp", modbusServerAddr)
	if err != nil {
		return -1
	}

	ctx, cancel := context.WithCancel(context.Background())
	modbusListener = listener
	modbusServerCancel = cancel
	modbusServerRunning = true

	go acceptModbusConnections(ctx, listener)
	return modbusServerPort
}

func stopModbusServer() int {
	modbusServerMu.Lock()
	listener := modbusListener
	cancel := modbusServerCancel
	modbusListener = nil
	modbusServerCancel = nil
	modbusServerRunning = false
	modbusServerMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if listener == nil {
		return 0
	}
	if err := listener.Close(); err != nil {
		return -1
	}
	return 0
}

func modbusServerStatus() map[string]any {
	modbusServerMu.Lock()
	running := modbusServerRunning
	modbusServerMu.Unlock()

	return map[string]any{
		"address": fmt.Sprintf("tcp://%s:%d", modbusServerHost, modbusServerPort),
		"port":    modbusServerPort,
		"running": running,
	}
}

func acceptModbusConnections(ctx context.Context, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		go handleModbusConnection(conn)
	}
}

func handleModbusConnection(conn net.Conn) {
	defer conn.Close()

	for {
		if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return
		}
		adu, err := readModbusTCPADU(conn)
		if err != nil {
			return
		}
		response := handleModbusTCPADU(adu)
		if _, err := conn.Write(response); err != nil {
			return
		}
	}
}

func readModbusTCPADU(reader io.Reader) ([]byte, error) {
	header := make([]byte, 7)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(header[4:6])
	if length == 0 || length > 253 {
		return nil, fmt.Errorf("invalid modbus length %d", length)
	}

	body := make([]byte, int(length)-1)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}

	adu := make([]byte, 0, len(header)+len(body))
	adu = append(adu, header...)
	adu = append(adu, body...)
	return adu, nil
}

func handleModbusTCPADU(adu []byte) []byte {
	transactionID := binary.BigEndian.Uint16(adu[0:2])
	unitID := adu[6]
	functionCode := adu[7]
	pdu := adu[7:]

	switch functionCode {
	case 3:
		return buildModbusReadHoldingResponse(transactionID, unitID, pdu)
	case 6:
		return buildModbusWriteSingleRegisterResponse(transactionID, unitID, pdu)
	case 16:
		return buildModbusWriteMultipleRegistersResponse(transactionID, unitID, pdu)
	default:
		return buildModbusException(transactionID, unitID, functionCode, 1)
	}
}

func buildModbusReadHoldingResponse(transactionID uint16, unitID uint8, pdu []byte) []byte {
	if len(pdu) != 5 {
		return buildModbusException(transactionID, unitID, 3, 3)
	}

	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	if quantity == 0 || quantity > 125 {
		return buildModbusException(transactionID, unitID, 3, 3)
	}

	data := make([]byte, quantity*2)
	modbusHoldingMu.RLock()
	for i := uint16(0); i < quantity; i++ {
		binary.BigEndian.PutUint16(data[i*2:i*2+2], modbusHoldingRecords[start+i])
	}
	modbusHoldingMu.RUnlock()

	responsePDU := make([]byte, 2+len(data))
	responsePDU[0] = 3
	responsePDU[1] = byte(len(data))
	copy(responsePDU[2:], data)
	return buildModbusTCPResponse(transactionID, unitID, responsePDU)
}

func buildModbusWriteSingleRegisterResponse(transactionID uint16, unitID uint8, pdu []byte) []byte {
	if len(pdu) != 5 {
		return buildModbusException(transactionID, unitID, 6, 3)
	}

	address := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])
	modbusHoldingMu.Lock()
	modbusHoldingRecords[address] = value
	modbusHoldingMu.Unlock()

	return buildModbusTCPResponse(transactionID, unitID, pdu)
}

func buildModbusWriteMultipleRegistersResponse(transactionID uint16, unitID uint8, pdu []byte) []byte {
	if len(pdu) < 6 {
		return buildModbusException(transactionID, unitID, 16, 3)
	}

	start := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])
	if quantity == 0 || quantity > 123 || byteCount != int(quantity)*2 || len(pdu) != 6+byteCount {
		return buildModbusException(transactionID, unitID, 16, 3)
	}

	modbusHoldingMu.Lock()
	for i := uint16(0); i < quantity; i++ {
		offset := 6 + int(i)*2
		modbusHoldingRecords[start+i] = binary.BigEndian.Uint16(pdu[offset : offset+2])
	}
	modbusHoldingMu.Unlock()

	responsePDU := make([]byte, 5)
	responsePDU[0] = 16
	binary.BigEndian.PutUint16(responsePDU[1:3], start)
	binary.BigEndian.PutUint16(responsePDU[3:5], quantity)
	return buildModbusTCPResponse(transactionID, unitID, responsePDU)
}

func buildModbusException(transactionID uint16, unitID uint8, functionCode uint8, exceptionCode uint8) []byte {
	return buildModbusTCPResponse(transactionID, unitID, []byte{functionCode | 0x80, exceptionCode})
}

func buildModbusTCPResponse(transactionID uint16, unitID uint8, pdu []byte) []byte {
	response := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(response[0:2], transactionID)
	binary.BigEndian.PutUint16(response[2:4], 0)
	binary.BigEndian.PutUint16(response[4:6], uint16(len(pdu)+1))
	response[6] = unitID
	copy(response[7:], pdu)
	return response
}
