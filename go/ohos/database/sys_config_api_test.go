package database

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSysConfigAPIListsVisibleRowsFromDatabase(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sys-config-list.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	rows := []TbSysConfigs{
		{FModuleName: "RSS", FType: "MaxSpeed", FValue: "680", FVisible: 1, FEnType: "Max Speed", FValueType: 3, FZhType: "最大速度"},
		{FModuleName: "RSS", FType: "MaxRealWeightCount", FValue: "66", FVisible: 1, FEnType: "Max Real Weight Count", FValueType: 3, FZhType: "最大产量"},
		{FModuleName: "RSS", FType: "MaxEmptyContent", FValue: "", FVisible: 1, FEnType: "Max Empty Content", FValueType: 3, FZhType: "空内容配置"},
		{FModuleName: "RSS", FType: "MaxHidden", FValue: "1", FVisible: 0, FEnType: "Hidden", FValueType: 1},
		{FModuleName: "OTHER", FType: "MaxSpeed", FValue: "999", FVisible: 1, FEnType: "Other", FValueType: 3},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed sys config rows: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)

	body := bytes.NewBufferString(`{"FType":"Max","FVisible":1,"FModuleName":"RSS"}`)
	request := httptest.NewRequest(http.MethodPost, "/Api/SysConfig/GetListSysConfigs", body)
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
	var got []sysConfigAPIModel
	if err := json.Unmarshal([]byte(envelope.Data), &got); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3: %#v", len(got), got)
	}
	if got[0].FType != "MaxSpeed" || got[0].FValue != "680" || got[0].FZhType != "最大速度" {
		t.Fatalf("first row = %#v, want MaxSpeed with FZhType", got[0])
	}
	if got[1].FType != "MaxRealWeightCount" {
		t.Fatalf("second row FType = %q, want MaxRealWeightCount", got[1].FType)
	}
	if got[2].FType != "MaxEmptyContent" || got[2].FValue != "" {
		t.Fatalf("third row = %#v, want MaxEmptyContent with empty FValue", got[2])
	}
}

func TestSysConfigAPISavesRowsToDatabase(t *testing.T) {
	resetORMForTest()
	dbPath := filepath.Join(t.TempDir(), "sys-config-save.db")
	t.Cleanup(resetORMForTest)
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath: %v", err)
	}
	db, err := getInitializedFileORMDB()
	if err != nil {
		t.Fatalf("getInitializedFileORMDB: %v", err)
	}

	existing := TbSysConfigs{
		FModuleName:      "RSS",
		FType:            "MaxSpeed",
		FValue:           "680",
		FVisible:         1,
		FEnType:          "Max Speed",
		FValueType:       3,
		FValueTypeDetail: "",
		FZhType:          "最大速度",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing row: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)

	payload := []sysConfigAPIModel{
		{FID: existing.FID, FModuleName: "RSS", FType: "MaxSpeed", FValue: "700", FVisible: 1, FEnType: "Max Speed", FValueType: 3, FZhType: "最大速度"},
		{FModuleName: "RSS", FType: "NewConfig", FValue: "1", FVisible: 1, FEnType: "New Config", FValueType: 1, FValueTypeDetail: "0;1", FZhType: "新配置"},
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/Api/SysConfig/SaveSysConfigs", bytes.NewReader(requestBody))
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

	var updated TbSysConfigs
	if err := db.Select("FValue").
		Where("FModuleName = ? AND FType = ?", "RSS", "MaxSpeed").
		Order("FID desc").
		First(&updated).Error; err != nil {
		t.Fatalf("read updated row: %v", err)
	}
	if updated.FValue != "700" {
		t.Fatalf("updated FValue = %q, want 700", updated.FValue)
	}

	var inserted TbSysConfigs
	if err := db.Select("FValue", "FValueType", "FValueTypeDetail", "FZhType").
		Where("FModuleName = ? AND FType = ?", "RSS", "NewConfig").
		First(&inserted).Error; err != nil {
		t.Fatalf("read inserted row: %v", err)
	}
	if inserted.FValue != "1" || inserted.FValueType != 1 || inserted.FValueTypeDetail != "0;1" || inserted.FZhType != "新配置" {
		t.Fatalf("inserted row = %#v, want saved metadata", inserted)
	}
}
