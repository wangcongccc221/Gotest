package database

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestFaultInfoAPIQueriesRowsByDateAndMessage(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "fault-info-query.db")
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
	activeStatus := 0
	recoveredStatus := 1
	firstBegin := time.Date(2026, 5, 20, 8, 30, 0, 0, databaseChinaLocation)
	secondBegin := time.Date(2026, 6, 10, 10, 0, 0, 0, databaseChinaLocation)
	secondEnd := time.Date(2026, 6, 10, 10, 10, 0, 0, databaseChinaLocation)
	rows := []TbFaultInfo{
		{FType: &electricalType, FCode: "1", FName: "FSM", FMessage: "出口传感器异常", FStatus: &activeStatus, FBeginDate: &firstBegin},
		{FType: &systemType, FCode: "2", FName: "RSS", FMessage: "数据库异常", FStatus: &recoveredStatus, FBeginDate: &secondBegin, FEndDate: &secondEnd},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed fault info: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/Api/Fault/GetListFaultInfoByDetailCondition",
		bytes.NewBufferString(`{"FStatus":-1,"FBeginDate":"2026-05-01 00:00:00","FEndDate":"2026-06-30 23:59:59","FMessage":"异常"}`))
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
	var got []faultInfoAPIModel
	if err := json.Unmarshal([]byte(envelope.Data), &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2: %#v", len(got), got)
	}
	if got[0].FCode != "2" || got[0].FStatus != 1 || got[0].FEndDate == "" {
		t.Fatalf("first row = %#v, want latest recovered row", got[0])
	}
}

func TestFaultInfoAPISavesAndReturnsUnfinishedRows(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "fault-info-save.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)
	payload := []faultInfoAPIModel{
		{FType: 1, FCode: "3", FName: "SCM", FMessage: "通信异常", FStatus: 0, FBeginDate: "2026-06-18 09:00:00"},
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/Api/Fault/SaveFaultInfos", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save status = %d, want %d", response.Code, http.StatusOK)
	}

	request = httptest.NewRequest(http.MethodPost, "/Api/Fault/GetListFaultInfo", bytes.NewBufferString(`{"FType":1,"FStatus":0}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unfinished status = %d, want %d", response.Code, http.StatusOK)
	}
	var envelope fruitInfoAPIEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode unfinished envelope: %v", err)
	}
	var unfinished []faultInfoAPIModel
	if err := json.Unmarshal([]byte(envelope.Data), &unfinished); err != nil {
		t.Fatalf("decode unfinished data: %v", err)
	}
	if len(unfinished) != 1 {
		t.Fatalf("len(unfinished) = %d, want 1: %#v", len(unfinished), unfinished)
	}

	unfinished[0].FStatus = 1
	unfinished[0].FEndDate = "2026-06-18 09:05:00"
	requestBody, err = json.Marshal(unfinished)
	if err != nil {
		t.Fatalf("marshal update payload: %v", err)
	}
	request = httptest.NewRequest(http.MethodPost, "/Api/Fault/SaveFaultInfos", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d", response.Code, http.StatusOK)
	}

	request = httptest.NewRequest(http.MethodPost, "/Api/Fault/GetListFaultInfo", bytes.NewBufferString(`{"FType":1,"FStatus":0}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("unfinished after close status = %d, want %d", response.Code, http.StatusOK)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode final envelope: %v", err)
	}
	if err := json.Unmarshal([]byte(envelope.Data), &unfinished); err != nil {
		t.Fatalf("decode final data: %v", err)
	}
	if len(unfinished) != 0 {
		t.Fatalf("len(unfinished) after close = %d, want 0: %#v", len(unfinished), unfinished)
	}
}
