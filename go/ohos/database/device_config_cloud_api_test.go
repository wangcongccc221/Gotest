package database

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeviceConfigCloudListDownloadAndDelete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "device-config.db")
	if err := InitORMWithPath(dbPath); err != nil {
		t.Fatalf("InitORMWithPath error = %v", err)
	}
	defer resetORMForTest()
	if err := SaveConfigValue("SecretKey", "unit-secret"); err != nil {
		t.Fatalf("save secret: %v", err)
	}
	if err := SaveConfigValue("VERSION_SHOW", "Vunit"); err != nil {
		t.Fatalf("save version: %v", err)
	}

	var seenDownload bool
	var seenDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("devicetype") != deviceConfigCloudDeviceType {
			t.Errorf("devicetype header = %q", r.Header.Get("devicetype"))
		}
		if r.Header.Get("secret-key") != "unit-secret" {
			t.Errorf("secret-key header = %q", r.Header.Get("secret-key"))
		}
		if strings.TrimSpace(r.Header.Get("signature")) == "" {
			t.Errorf("signature header is empty")
		}
		switch r.URL.Path {
		case "/Api/Customer/GetDeviceConfig":
			fmt.Fprint(w, `{"returnCode":1,"returnMessage":"","data":"[{\"FileName\":\"apple.exp\",\"LastWriteTime\":\"2026-06-18T09:30:00\"},{\"FileName\":\"banana.rjs\",\"LastWriteTime\":\"2026-06-17 08:00:00\"},{\"FileName\":\"ignore.txt\",\"LastWriteTime\":\"2026-06-16 08:00:00\"}]"}`)
		case "/Api/Customer/DownDeviceConfig":
			seenDownload = true
			fmt.Fprintf(w, `{"returnCode":1,"returnMessage":"","data":%q}`, base64.StdEncoding.EncodeToString([]byte("config-data")))
		case "/Api/Customer/DeleteDeviceConfig":
			seenDelete = true
			fmt.Fprint(w, `{"returnCode":1,"returnMessage":"","data":""}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := SaveConfigValue("DeviceConfigCloudBaseUrl", server.URL+"/Api/"); err != nil {
		t.Fatalf("save cloud url: %v", err)
	}

	files, err := ListDeviceConfigs()
	if err != nil {
		t.Fatalf("ListDeviceConfigs error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2: %#v", len(files), files)
	}
	if files[0].FileName != "apple.exp" || files[0].Type != "Process Config" {
		t.Fatalf("files[0] = %#v", files[0])
	}
	if files[1].FileName != "banana.rjs" || files[1].Type != "User Config" {
		t.Fatalf("files[1] = %#v", files[1])
	}

	result, err := DownloadDeviceConfig("apple.exp")
	if err != nil {
		t.Fatalf("DownloadDeviceConfig error = %v", err)
	}
	if !seenDownload {
		t.Fatalf("fake cloud download endpoint was not called")
	}
	data, err := os.ReadFile(result.LocalPath)
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != "config-data" {
		t.Fatalf("downloaded data = %q, want config-data", string(data))
	}
	if filepath.Dir(result.LocalPath) != filepath.Join(filepath.Dir(dbPath), "config", "Vunit") {
		t.Fatalf("local path = %s", result.LocalPath)
	}

	if err := DeleteDeviceConfig("apple.exp"); err != nil {
		t.Fatalf("DeleteDeviceConfig error = %v", err)
	}
	if !seenDelete {
		t.Fatalf("fake cloud delete endpoint was not called")
	}
	if _, err := os.Stat(result.LocalPath); !os.IsNotExist(err) {
		t.Fatalf("downloaded file still exists or stat error = %v", err)
	}
}

func TestValidateDeviceConfigFileName(t *testing.T) {
	if _, _, err := validateDeviceConfigFileName("../bad.exp"); err == nil {
		t.Fatalf("path traversal file name was accepted")
	}
	if _, _, err := validateDeviceConfigFileName("bad.txt"); err == nil {
		t.Fatalf("unsupported file extension was accepted")
	}
}
