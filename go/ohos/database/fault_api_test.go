package database

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBaseFaultAPIListsRowsByType(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "base-fault-list.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	electricalType := 1
	systemType := 0
	rows := []TbBaseFault{
		{FType: &electricalType, FCode: "1", FName: "SCM", FGroupName: "SCM", FMessage: "通信超时"},
		{FType: &electricalType, FCode: "2", FName: "FSM", FGroupName: "FSM", FMessage: "出口传感器异常"},
		{FType: &systemType, FCode: "3", FName: "RSS", FGroupName: "RSS", FMessage: "数据库异常"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed base faults: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/Api/Fault/GetListBaseFaultInfo", bytes.NewBufferString(`{"FType":1}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var envelope fruitInfoAPIEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.ReturnCode != 1 {
		t.Fatalf("ReturnCode = %d, message=%q", envelope.ReturnCode, envelope.ReturnMessage)
	}
	var got []baseFaultAPIModel
	if err := json.Unmarshal([]byte(envelope.Data), &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2: %#v", len(got), got)
	}
	if got[0].FCode != "1" || got[0].FName != "SCM" || got[0].FMessage != "通信超时" {
		t.Fatalf("first row = %#v, want electrical SCM row", got[0])
	}
}

func TestBaseFaultAPISavesAndDeletesRows(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "base-fault-save.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	electricalType := 1
	existing := TbBaseFault{
		FType:      &electricalType,
		FCode:      "1",
		FName:      "SCM",
		FGroupName: "SCM",
		FMessage:   "旧信息",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing base fault: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)
	payload := []baseFaultAPIModel{
		{FID: existing.FID, FType: 1, FCode: "1", FName: "SCM", FGroupName: "", FMessage: "新信息"},
		{FType: 1, FCode: "2", FName: "FSM", FMessage: "新增故障"},
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/Api/Fault/SaveBaseFaults", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save status = %d, want %d", response.Code, http.StatusOK)
	}

	var updated TbBaseFault
	if err := db.Select("FID", "FMessage", "FGroupName").Where("FID = ?", existing.FID).First(&updated).Error; err != nil {
		t.Fatalf("read updated row: %v", err)
	}
	if updated.FMessage != "新信息" || updated.FGroupName != "SCM" {
		t.Fatalf("updated = %#v, want message updated and group defaulted", updated)
	}

	var inserted TbBaseFault
	if err := db.Select("FID", "FGroupName", "FMessage").Where("FCode = ? AND FName = ?", "2", "FSM").First(&inserted).Error; err != nil {
		t.Fatalf("read inserted row: %v", err)
	}
	if inserted.FGroupName != "FSM" || inserted.FMessage != "新增故障" {
		t.Fatalf("inserted = %#v, want group defaulted", inserted)
	}

	deleteBody := bytes.NewBufferString(`{"FID":` + strconv.Itoa(inserted.FID) + `}`)
	request = httptest.NewRequest(http.MethodPost, "/Api/Fault/DeleteBaseFaultInfo", deleteBody)
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d", response.Code, http.StatusOK)
	}
	var count int64
	if err := db.Model(&TbBaseFault{}).Where("FID = ?", inserted.FID).Count(&count).Error; err != nil {
		t.Fatalf("count deleted row: %v", err)
	}
	if count != 0 {
		t.Fatalf("deleted row count = %d, want 0", count)
	}
}
