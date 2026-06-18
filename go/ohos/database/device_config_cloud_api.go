package database

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	deviceConfigCloudDeviceType = "FruitSort200"
	deviceConfigCloudAPIVersion = "2.0"
	deviceConfigCloudLocalURL   = "http://192.168.10.29:8899/Api/"
	deviceConfigCloudRemoteURL  = "http://111.75.253.33:8899/Api/"
	deviceConfigDefaultVersion  = "V4.02.01"
)

var deviceConfigHTTPClient = &http.Client{Timeout: 10 * time.Second}

type DeviceConfigFileInfo struct {
	FileName string `json:"FileName"`
	Type     string `json:"Type"`
	Date     string `json:"Date"`
}

type DeviceConfigDownloadResult struct {
	FileName  string `json:"FileName"`
	LocalPath string `json:"LocalPath"`
}

type deviceConfigFileRequest struct {
	FileName string `json:"FileName"`
}

type deviceConfigCloudEnvelope struct {
	ReturnCode    int    `json:"returnCode"`
	ReturnMessage string `json:"returnMessage"`
	Data          string `json:"data"`
}

type deviceConfigCloudFileInfo struct {
	FileName      string          `json:"FileName"`
	LastWriteTime json.RawMessage `json:"LastWriteTime"`
}

func registerDeviceConfigCloudRoutes(router *gin.Engine) {
	group := router.Group("/Api/DeviceConfig")
	group.POST("/GetDeviceConfig", handleDeviceConfigList)
	group.POST("/DownDeviceConfig", handleDeviceConfigDownload)
	group.POST("/DeleteDeviceConfig", handleDeviceConfigDelete)
}

func handleDeviceConfigList(ctx *gin.Context) {
	files, err := ListDeviceConfigs()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, files)
}

func handleDeviceConfigDownload(ctx *gin.Context) {
	var request deviceConfigFileRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	result, err := DownloadDeviceConfig(request.FileName)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func handleDeviceConfigDelete(ctx *gin.Context) {
	var request deviceConfigFileRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	if err := DeleteDeviceConfig(request.FileName); err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, DeviceConfigDownloadResult{FileName: filepath.Base(request.FileName)})
}

func ListDeviceConfigs() ([]DeviceConfigFileInfo, error) {
	response, err := callDeviceConfigCloud("Customer/GetDeviceConfig", "")
	if err != nil {
		return nil, err
	}

	var cloudFiles []deviceConfigCloudFileInfo
	if err := json.Unmarshal([]byte(response.Data), &cloudFiles); err != nil {
		return nil, fmt.Errorf("parse cloud config list: %w", err)
	}

	files := make([]DeviceConfigFileInfo, 0, len(cloudFiles))
	for _, file := range cloudFiles {
		fileName := strings.TrimSpace(file.FileName)
		configType, ok := deviceConfigTypeByFileName(fileName)
		if !ok {
			continue
		}
		files = append(files, DeviceConfigFileInfo{
			FileName: fileName,
			Type:     configType,
			Date:     formatDeviceConfigCloudTime(file.LastWriteTime),
		})
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Date > files[j].Date
	})
	return files, nil
}

func DownloadDeviceConfig(fileName string) (DeviceConfigDownloadResult, error) {
	safeName, _, err := validateDeviceConfigFileName(fileName)
	if err != nil {
		return DeviceConfigDownloadResult{}, err
	}
	response, err := callDeviceConfigCloud("Customer/DownDeviceConfig", safeName)
	if err != nil {
		return DeviceConfigDownloadResult{}, err
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(response.Data))
	if err != nil {
		return DeviceConfigDownloadResult{}, fmt.Errorf("decode cloud config file %s: %w", safeName, err)
	}
	dir, err := resolveDeviceConfigLocalDir()
	if err != nil {
		return DeviceConfigDownloadResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return DeviceConfigDownloadResult{}, fmt.Errorf("create config directory: %w", err)
	}
	localPath := filepath.Join(dir, safeName)
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return DeviceConfigDownloadResult{}, fmt.Errorf("write config file %s: %w", safeName, err)
	}
	return DeviceConfigDownloadResult{FileName: safeName, LocalPath: localPath}, nil
}

func DeleteDeviceConfig(fileName string) error {
	safeName, _, err := validateDeviceConfigFileName(fileName)
	if err != nil {
		return err
	}
	if _, err := callDeviceConfigCloud("Customer/DeleteDeviceConfig", safeName); err != nil {
		return err
	}
	dir, err := resolveDeviceConfigLocalDir()
	if err != nil {
		return err
	}
	localPath := filepath.Join(dir, safeName)
	if err := os.Remove(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete local config file %s: %w", safeName, err)
	}
	return nil
}

func callDeviceConfigCloud(endpoint string, postData string) (deviceConfigCloudEnvelope, error) {
	secretKey, err := resolveDeviceConfigSecretKey()
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}
	baseURL, err := resolveDeviceConfigCloudBaseURL()
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}

	encryptedData := ""
	if strings.TrimSpace(postData) != "" {
		encryptedData, err = encryptDeviceConfigCloudAES(secretKey, postData)
		if err != nil {
			return deviceConfigCloudEnvelope{}, err
		}
	}
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	signatureText := md5LowerHex("data" + encryptedData + "timestamp" + timestamp)
	signature, err := encryptDeviceConfigCloudAES(secretKey, signatureText)
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}

	requestURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(encryptedData))
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("devicetype", deviceConfigCloudDeviceType)
	request.Header.Set("api-version", deviceConfigCloudAPIVersion)
	request.Header.Set("language", resolveDeviceConfigLanguage())
	request.Header.Set("secret-key", secretKey)
	request.Header.Set("signature", signature)
	request.Header.Set("timestamp", timestamp)

	response, err := deviceConfigHTTPClient.Do(request)
	if err != nil {
		return deviceConfigCloudEnvelope{}, fmt.Errorf("call cloud %s failed: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return deviceConfigCloudEnvelope{}, fmt.Errorf("cloud %s returned HTTP %d", endpoint, response.StatusCode)
	}

	var envelope deviceConfigCloudEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return deviceConfigCloudEnvelope{}, fmt.Errorf("parse cloud %s response: %w", endpoint, err)
	}
	if envelope.ReturnCode != 1 {
		message := strings.TrimSpace(envelope.ReturnMessage)
		if message == "" {
			message = fmt.Sprintf("cloud %s failed", endpoint)
		}
		return deviceConfigCloudEnvelope{}, errors.New(message)
	}
	return envelope, nil
}

func resolveDeviceConfigSecretKey() (string, error) {
	secret, err := GetConfigValuePreferNonEmpty("SecretKey")
	if err != nil {
		return "", err
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", errors.New("SecretKey is empty, cloud device config API is unavailable")
	}
	return secret, nil
}

func resolveDeviceConfigCloudBaseURL() (string, error) {
	for _, key := range []string{"DeviceConfigCloudBaseUrl", "CloudApiUrl", "CurrentUrl"} {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return normalizeDeviceConfigCloudBaseURL(value), nil
		}
	}
	for _, candidate := range []string{deviceConfigCloudLocalURL, deviceConfigCloudRemoteURL} {
		if canConnectDeviceConfigURL(candidate, time.Second) {
			return candidate, nil
		}
	}
	return deviceConfigCloudRemoteURL, nil
}

func normalizeDeviceConfigCloudBaseURL(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return deviceConfigCloudRemoteURL
	}
	text = strings.TrimRight(text, "/") + "/"
	if !strings.HasSuffix(strings.ToLower(text), "/api/") {
		text += "Api/"
	}
	return text
}

func canConnectDeviceConfigURL(rawURL string, timeout time.Duration) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", parsed.Host, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func resolveDeviceConfigLanguage() string {
	for _, key := range []string{"selectLanguage", "SelectLanguage", "Language"} {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Chinese"
}

func resolveDeviceConfigLocalDir() (string, error) {
	_, databasePath, err := getORMDBWithInfo()
	if err != nil {
		return "", err
	}
	baseDir := ""
	if databasePath != "" && !strings.HasPrefix(databasePath, "file:") {
		baseDir = filepath.Dir(databasePath)
	}
	if baseDir == "" || baseDir == "." {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(cacheDir, "gotest")
	}
	return filepath.Join(baseDir, "config", resolveDeviceConfigVersion()), nil
}

func resolveDeviceConfigVersion() string {
	for _, key := range []string{"VERSION_SHOW", "VersionShow", "软件版本"} {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err == nil && strings.TrimSpace(value) != "" {
			return sanitizeDeviceConfigPathPart(value)
		}
	}
	return deviceConfigDefaultVersion
}

func sanitizeDeviceConfigPathPart(value string) string {
	text := strings.TrimSpace(value)
	text = strings.ReplaceAll(text, "/", "_")
	text = strings.ReplaceAll(text, "\\", "_")
	if text == "" || text == "." || text == ".." {
		return deviceConfigDefaultVersion
	}
	return text
}

func validateDeviceConfigFileName(fileName string) (string, string, error) {
	safeName := strings.TrimSpace(fileName)
	if safeName == "" {
		return "", "", errors.New("config file name is required")
	}
	if safeName != filepath.Base(safeName) || strings.ContainsAny(safeName, `/\`) {
		return "", "", errors.New("invalid config file name")
	}
	configType, ok := deviceConfigTypeByFileName(safeName)
	if !ok {
		return "", "", fmt.Errorf("unsupported config file type: %s", filepath.Ext(safeName))
	}
	return safeName, configType, nil
}

func deviceConfigTypeByFileName(fileName string) (string, bool) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".exp", ".ejs":
		return "Process Config", true
	case ".cmc", ".rjs":
		return "User Config", true
	default:
		return "", false
	}
}

func formatDeviceConfigCloudTime(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		text = strings.TrimSpace(stringValue)
	}
	text = strings.Trim(text, `"`)
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, text, time.Local); err == nil {
			return parsed.Format("2006-01-02 15:04:05")
		}
	}
	return text
}

func md5LowerHex(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func encryptDeviceConfigCloudAES(secretKey string, plainText string) (string, error) {
	if strings.TrimSpace(secretKey) == "" || plainText == "" {
		return "", nil
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0x20
	}
	copy(key, []byte(secretKey))
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	padded := pkcs7Pad([]byte(plainText), block.BlockSize())
	encrypted := make([]byte, len(padded))
	for start := 0; start < len(padded); start += block.BlockSize() {
		block.Encrypt(encrypted[start:start+block.BlockSize()], padded[start:start+block.BlockSize()])
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	result := make([]byte, len(data)+padding)
	copy(result, data)
	for index := len(data); index < len(result); index++ {
		result[index] = byte(padding)
	}
	return result
}
