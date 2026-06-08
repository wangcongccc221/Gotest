package database

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	ormPackage   = "gorm.io/gorm"
	ormDriver    = "github.com/glebarez/sqlite"
	ormMemoryDSN = "file:harmony_go_orm?mode=memory&cache=shared"
)

type DeviceRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:80;not null"`
	Protocol  string    `json:"protocol" gorm:"size:40;not null;index"`
	Endpoint  string    `json:"endpoint" gorm:"size:160;not null"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type deviceRecordInput struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Endpoint string `json:"endpoint"`
}

var (
	ormMu             sync.Mutex
	activeORM         *gorm.DB
	activeORMDSN      string
	activeORMDatabase string
	ormInitErr        error
)

func registerORMRoutes(router *gin.Engine) {
	router.GET("/orm", handleORMStatus)
	router.POST("/orm/init", handleORMInit)
	router.GET("/orm/devices", handleORMListDevices)
	router.POST("/orm/devices", handleORMCreateDevice)

}

func RegisterRoutes(router *gin.Engine) {
	registerORMRoutes(router)
	registerFruitInfoRoutes(router)
	registerFruitProcessInfoRoutes(router)
}

func handleORMStatus(ctx *gin.Context) {
	status, code := ormStatus()
	ctx.JSON(code, status)
}

func handleORMInit(ctx *gin.Context) {
	if err := initORMWithPath(ctx.Query("path")); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	status, _ := ormStatus()
	ctx.JSON(http.StatusOK, status)
}

func handleORMListDevices(ctx *gin.Context) {
	devices, err := ormListDevices()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

func handleORMCreateDevice(ctx *gin.Context) {
	input, err := readDeviceRecordInput(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	device, err := ormCreateDevice(input)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, device)
}

func initORM() error {
	return initORMWithPath("")
}

func InitORM() error {
	return initORM()
}

func InitORMWithPath(dbPath string) error {
	return initORMWithPath(dbPath)
}

func initORMWithPath(dbPath string) error {
	dsn, database, err := resolveORMDSN(dbPath)
	if err != nil {
		return err
	}

	ormMu.Lock()
	defer ormMu.Unlock()

	if activeORM != nil {
		if activeORMDSN == dsn {
			return ormInitErr
		}
		closeActiveORMLocked()
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		ormInitErr = err
		return err
	}
	if err := migrateORMTables(db); err != nil {
		ormInitErr = err
		return err
	}
	if err := seedORMDevices(db); err != nil {
		ormInitErr = err
		return err
	}

	activeORM = db
	activeORMDSN = dsn
	activeORMDatabase = database
	ormInitErr = nil
	return nil
}

func ormStatus() (gin.H, int) { //
	db, database, err := getORMDBWithInfo()
	status := gin.H{
		"package":  ormPackage,
		"driver":   ormDriver,
		"database": database,
	}
	if err != nil {
		status["status"] = "error"
		status["error"] = err.Error()
		return status, http.StatusInternalServerError
	}

	var count int64
	if err := db.Model(&DeviceRecord{}).Count(&count).Error; err != nil {
		status["status"] = "error"
		status["error"] = err.Error()
		return status, http.StatusInternalServerError
	}

	status["status"] = "ready"
	status["records"] = count
	return status, http.StatusOK
}

func ormListDevices() ([]DeviceRecord, error) { //查询数据
	db, err := getORMDB()
	if err != nil {
		return nil, err
	}

	var devices []DeviceRecord
	if err := db.Order("id asc").Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}

func ormCreateDevice(input deviceRecordInput) (DeviceRecord, error) { //创建数据
	if err := validateDeviceRecordInput(&input); err != nil {
		return DeviceRecord{}, err
	}

	db, err := getORMDB()
	if err != nil {
		return DeviceRecord{}, err
	}

	device := DeviceRecord{
		Name:     input.Name,
		Protocol: input.Protocol,
		Endpoint: input.Endpoint,
	}
	if err := db.Create(&device).Error; err != nil {
		return DeviceRecord{}, err
	}
	return device, nil
}

func getORMDB() (*gorm.DB, error) { //返回信息
	db, _, err := getORMDBWithInfo()
	return db, err
}

func getORMDBWithInfo() (*gorm.DB, string, error) {
	ormMu.Lock()
	db := activeORM
	database := activeORMDatabase
	initErr := ormInitErr
	ormMu.Unlock()

	if db != nil {
		return db, database, initErr
	}
	return nil, "", errors.New("ORM database is not initialized")
}

func seedORMDevices(db *gorm.DB) error {
	var count int64
	if err := db.Model(&DeviceRecord{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Create(&[]DeviceRecord{
		{Name: "Gin HTTP", Protocol: "HTTP", Endpoint: "http://127.0.0.1:18080/ping"},
		{Name: "Gin WebSocket", Protocol: "WebSocket", Endpoint: "ws://127.0.0.1:18080/ws"},
		{Name: "OPC UA Demo", Protocol: "OPC UA", Endpoint: "opc.tcp://127.0.0.1:4840"},
		{Name: "Modbus TCP Demo", Protocol: "Modbus TCP", Endpoint: "tcp://127.0.0.1:5020"},
	}).Error
}

func readDeviceRecordInput(ctx *gin.Context) (deviceRecordInput, error) {
	input := deviceRecordInput{
		Name:     ctx.Query("name"),
		Protocol: ctx.Query("protocol"),
		Endpoint: ctx.Query("endpoint"),
	}

	if strings.Contains(ctx.GetHeader("Content-Type"), "application/json") {
		if err := ctx.ShouldBindJSON(&input); err != nil {
			return deviceRecordInput{}, err
		}
	}

	if err := validateDeviceRecordInput(&input); err != nil {
		return deviceRecordInput{}, err
	}
	return input, nil
}

func validateDeviceRecordInput(input *deviceRecordInput) error { //校验数据
	input.Name = strings.TrimSpace(input.Name)
	input.Protocol = strings.TrimSpace(input.Protocol)
	input.Endpoint = strings.TrimSpace(input.Endpoint)

	if input.Name == "" {
		return errors.New("name is required")
	}
	if input.Protocol == "" {
		return errors.New("protocol is required")
	}
	if input.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	return nil
}

func resolveORMDSN(dbPath string) (string, string, error) {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return ormMemoryDSN, "sqlite-memory", nil
	}

	cleanPath := filepath.Clean(dbPath)
	dir := filepath.Dir(cleanPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
	}
	return cleanPath, cleanPath, nil
}

func closeActiveORMLocked() {
	if activeORM != nil {
		if sqlDB, err := activeORM.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	activeORM = nil
	activeORMDSN = ""
	activeORMDatabase = ""
}

func resetORMForTest() {
	ormMu.Lock()
	closeActiveORMLocked()
	ormInitErr = nil
	ormMu.Unlock()
}
