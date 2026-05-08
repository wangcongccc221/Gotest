package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	modbus "github.com/aldas/go-modbus-client"
	"github.com/aldas/go-modbus-client/packet"
	"github.com/gin-gonic/gin"
)

const modbusRequestTimeout = 5 * time.Second

func registerModbusRoutes(router *gin.Engine) {
	router.GET("/modbus", handleModbusStatus)
	router.GET("/modbus/server", handleModbusServerStatus)
	router.POST("/modbus/server/start", handleModbusServerStart)
	router.POST("/modbus/server/stop", handleModbusServerStop)
	router.GET("/modbus/read-holding", handleModbusReadHolding)
	router.POST("/modbus/write-single", handleModbusWriteSingle)
}

func handleModbusStatus(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"package": "github.com/aldas/go-modbus-client",
		"status":  "ready",
	})
}

func handleModbusServerStatus(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, modbusServerStatus())
}

func handleModbusServerStart(ctx *gin.Context) {
	port := startModbusServer()
	if port < 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Modbus TCP server failed to start"})
		return
	}
	ctx.JSON(http.StatusOK, modbusServerStatus())
}

func handleModbusServerStop(ctx *gin.Context) {
	if code := stopModbusServer(); code != 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Modbus TCP server failed to stop"})
		return
	}
	ctx.JSON(http.StatusOK, modbusServerStatus())
}

func handleModbusReadHolding(ctx *gin.Context) { //处理读保持寄存器请求
	target := strings.TrimSpace(ctx.DefaultQuery("target", "tcp://127.0.0.1:5020"))
	unitID, err := parseUint8Query(ctx, "unit", 1)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	address, err := parseUint16Query(ctx, "address", 0)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	quantity, err := parseUint16Query(ctx, "quantity", 1)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := modbusReadHoldingRegisters(target, unitID, address, quantity)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func handleModbusWriteSingle(ctx *gin.Context) {
	target := strings.TrimSpace(ctx.DefaultQuery("target", "tcp://127.0.0.1:5020"))
	unitID, err := parseUint8Query(ctx, "unit", 1)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	address, err := parseUint16Query(ctx, "address", 0)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	value, err := parseUint16Query(ctx, "value", 0)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := modbusWriteSingleRegister(target, unitID, address, value)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func modbusReadHoldingRegisters(target string, unitID uint8, address uint16, quantity uint16) (map[string]any, error) {
	req, err := packet.NewReadHoldingRegistersRequestTCP(unitID, address, quantity)
	if err != nil {
		return nil, err
	}

	client := modbus.NewTCPClient()
	ctx, cancel := context.WithTimeout(context.Background(), modbusRequestTimeout)
	defer cancel()
	if err := client.Connect(ctx, target); err != nil {
		return nil, err
	}
	defer client.Close()

	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	holdingResp, ok := resp.(*packet.ReadHoldingRegistersResponseTCP)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T", resp)
	}
	registers, err := holdingResp.AsRegisters(address)
	if err != nil {
		return nil, err
	}

	values := make(map[string]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		addr := address + i
		value, err := registers.Uint16(addr)
		if err != nil {
			return nil, err
		}
		values[strconv.Itoa(int(addr))] = value
	}

	return map[string]any{
		"target":   target,
		"unit":     unitID,
		"address":  address,
		"quantity": quantity,
		"values":   values,
	}, nil
}

func modbusWriteSingleRegister(target string, unitID uint8, address uint16, value uint16) (map[string]any, error) {
	data := make([]byte, 2)
	binary.BigEndian.PutUint16(data, value)
	req, err := packet.NewWriteSingleRegisterRequestTCP(unitID, address, data)
	if err != nil {
		return nil, err
	}

	client := modbus.NewTCPClient()
	ctx, cancel := context.WithTimeout(context.Background(), modbusRequestTimeout)
	defer cancel()
	if err := client.Connect(ctx, target); err != nil {
		return nil, err
	}
	defer client.Close()

	if _, err := client.Do(ctx, req); err != nil {
		return nil, err
	}

	return map[string]any{
		"target":  target,
		"unit":    unitID,
		"address": address,
		"value":   value,
	}, nil
}

func parseUint8Query(ctx *gin.Context, name string, defaultValue uint8) (uint8, error) {
	value := strings.TrimSpace(ctx.Query(name))
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return uint8(parsed), nil
}

func parseUint16Query(ctx *gin.Context, name string, defaultValue uint16) (uint16, error) {
	value := strings.TrimSpace(ctx.Query(name))
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return uint16(parsed), nil
}
