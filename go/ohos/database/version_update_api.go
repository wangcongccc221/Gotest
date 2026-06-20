package database

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	versionUpdateLatestMessage = "current version is latest"
	versionUpdateRSS           = "RSS"
	versionUpdateFSM           = "FSM"
	versionUpdateWAM           = "WAM"
)

type VersionCheckResult struct {
	Items []VersionCheckItem `json:"Items"`
}

type VersionCheckItem struct {
	Name            string `json:"Name"`
	Version         string `json:"Version"`
	LocalVersion    string `json:"LocalVersion"`
	CloudVersion    string `json:"CloudVersion"`
	Content         string `json:"Content"`
	CanUpdate       bool   `json:"CanUpdate"`
	Project         string `json:"Project"`
	File            string `json:"File"`
	DownloadVersion string `json:"DownloadVersion"`
}

type VersionDownloadRequest struct {
	Name            string `json:"Name"`
	DownloadVersion string `json:"DownloadVersion"`
	File            string `json:"File"`
}

type VersionDownloadResult struct {
	Name      string `json:"Name"`
	FileName  string `json:"FileName"`
	LocalPath string `json:"LocalPath"`
}

func registerVersionUpdateRoutes(router *gin.Engine) {
	group := router.Group("/Api/Version")
	group.POST("/CheckVersion", handleVersionCheck)
	group.POST("/Download", handleVersionDownload)
}

func handleVersionCheck(ctx *gin.Context) {
	result, err := CheckUpdateVersions()
	if err != nil {
		fmt.Printf("%s CheckVersion failed: %v\n", time.Now().Format("15:04:05.000"), err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func handleVersionDownload(ctx *gin.Context) {
	var request VersionDownloadRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	result, err := DownloadUpdateVersion(request)
	if err != nil {
		fmt.Printf("%s DownloadVersion failed: name=%s version=%s err=%v\n",
			time.Now().Format("15:04:05.000"), request.Name, request.DownloadVersion, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func CheckUpdateVersions() (VersionCheckResult, error) {
	requestItems, local := buildVersionUpdateRequestItems()
	postDataBytes, err := json.Marshal(requestItems)
	if err != nil {
		return VersionCheckResult{}, err
	}
	postData := string(postDataBytes)
	fmt.Printf("%s CheckVersion: request versions RSS=%s FSM=%s WAM=%s\n",
		time.Now().Format("15:04:05.000"), requestItems[0].FVersion, requestItems[1].FVersion, requestItems[2].FVersion)

	response, err := callDeviceConfigCloud("Customer/GetDownLoadInfo", postData)
	if err != nil {
		return VersionCheckResult{}, err
	}

	var cloudItems []TbUpdateVersion
	if err := json.Unmarshal([]byte(response.Data), &cloudItems); err != nil {
		return VersionCheckResult{}, fmt.Errorf("parse update version response: %w", err)
	}
	result := VersionCheckResult{Items: buildVersionCheckItems(cloudItems, local)}
	fmt.Printf("%s CheckVersion success: cloudItems=%d uiItems=%d\n",
		time.Now().Format("15:04:05.000"), len(cloudItems), len(result.Items))
	return result, nil
}

func DownloadUpdateVersion(request VersionDownloadRequest) (VersionDownloadResult, error) {
	name := strings.ToUpper(strings.TrimSpace(request.Name))
	version := strings.TrimSpace(request.DownloadVersion)
	if name == "" || version == "" {
		return VersionDownloadResult{}, errors.New("update name and version are required")
	}

	payload := TbUpdateVersion{
		FType:    1,
		FVersion: version,
		FFile:    strings.TrimSpace(request.File),
	}
	postDataBytes, err := json.Marshal(payload)
	if err != nil {
		return VersionDownloadResult{}, err
	}
	fmt.Printf("%s DownloadVersion: name=%s version=%s\n", time.Now().Format("15:04:05.000"), name, version)
	response, err := callDeviceConfigCloud("Customer/Download", string(postDataBytes))
	if err != nil {
		return VersionDownloadResult{}, err
	}

	dataText := strings.TrimSpace(response.Data)
	if dataText == "" {
		return VersionDownloadResult{}, errors.New("cloud download returned empty data")
	}
	data, err := base64.StdEncoding.DecodeString(dataText)
	if err != nil {
		return VersionDownloadResult{}, fmt.Errorf("decode update file: %w", err)
	}
	dir, err := resolveVersionDownloadDir()
	if err != nil {
		return VersionDownloadResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return VersionDownloadResult{}, err
	}
	fileName := sanitizeVersionDownloadFileName(request.File, name, version)
	localPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return VersionDownloadResult{}, err
	}
	fmt.Printf("%s DownloadVersion success: name=%s path=%s bytes=%d\n",
		time.Now().Format("15:04:05.000"), name, localPath, len(data))
	return VersionDownloadResult{Name: name, FileName: fileName, LocalPath: localPath}, nil
}

type localVersionSet struct {
	RSS string
	FSM string
	WAM string
}

func buildVersionUpdateRequestItems() ([]TbUpdateVersion, localVersionSet) {
	local := localVersionSet{
		RSS: resolveVersionConfig([]string{"RSSVersion", "RssVersion", "VERSION_SHOW", "VersionShow"}, "2.0.0"),
		FSM: resolveVersionConfig([]string{"FSMVersion", "FsmVersion"}, "2.0.0"),
		WAM: resolveVersionConfig([]string{"WAMVersion", "WamVersion"}, "3.0.0"),
	}
	fsmRequest := buildControllerUpdateRequestVersion(local.FSM, "FSM_200", "FSM")
	wamRequest := buildControllerUpdateRequestVersion(local.WAM, "WAM_200", "WAM")
	rssRequest := "Linux_FruitSort200.48"
	if parts := strings.Split(fsmRequest, "_"); len(parts) >= 3 && fsmRequest != "FSM_200" {
		rssRequest = parts[0] + "_" + parts[1] + "_Linux_FruSort"
	}
	return []TbUpdateVersion{
		{FDeviceType: "Linux_FruitSort", FVersion: rssRequest},
		{FDeviceType: "1", FVersion: fsmRequest},
		{FDeviceType: "1", FVersion: wamRequest},
	}, local
}

func resolveVersionConfig(keys []string, fallback string) string {
	for _, key := range keys {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func buildControllerUpdateRequestVersion(localVersion string, fallback string, device string) string {
	text := strings.TrimSpace(localVersion)
	if text == "" || strings.HasPrefix(strings.ToUpper(text), "V") || !strings.Contains(text, "_") {
		return fallback
	}
	parts := strings.Split(text, "_")
	if len(parts) >= 3 {
		return parts[0] + "_" + parts[1] + "_" + device
	}
	return fallback
}

func buildVersionCheckItems(cloudItems []TbUpdateVersion, local localVersionSet) []VersionCheckItem {
	result := []VersionCheckItem{
		buildVersionCheckItem(versionUpdateRSS, cloudItemAt(cloudItems, 0), local.RSS),
		buildVersionCheckItem(versionUpdateFSM, cloudItemAt(cloudItems, 1), local.FSM),
		buildVersionCheckItem(versionUpdateWAM, cloudItemAt(cloudItems, 2), local.WAM),
	}
	return result
}

func cloudItemAt(items []TbUpdateVersion, index int) TbUpdateVersion {
	if index >= 0 && index < len(items) {
		return items[index]
	}
	return TbUpdateVersion{}
}

func buildVersionCheckItem(name string, cloud TbUpdateVersion, localVersion string) VersionCheckItem {
	versionLabel, cloudVersion := parseCloudVersion(name, cloud.FProject)
	if versionLabel == "" {
		versionLabel = "V" + normalizeVersionValue(localVersion)
	}
	canUpdate := cloudVersion != "" && checkUpdateVersion(cloudVersion, localVersion)
	content := versionUpdateLatestMessage
	if canUpdate {
		content = strings.TrimSpace(cloud.FZHDescribe)
		if content == "" {
			content = strings.TrimSpace(cloud.FENDescribe)
		}
	}
	return VersionCheckItem{
		Name:            name,
		Version:         versionLabel,
		LocalVersion:    normalizeVersionValue(localVersion),
		CloudVersion:    cloudVersion,
		Content:         content,
		CanUpdate:       canUpdate,
		Project:         cloud.FProject,
		File:            cloud.FFile,
		DownloadVersion: buildDownloadVersion(name, localVersion),
	}
}

func parseCloudVersion(name string, project string) (string, string) {
	parts := strings.Split(strings.TrimSpace(project), "_")
	index := 3
	if name == versionUpdateRSS {
		index = 4
	}
	label := ""
	if len(parts) > index {
		label = strings.TrimSpace(parts[index])
	}
	if label == "" || !strings.HasPrefix(strings.ToUpper(label), "V") {
		for _, part := range parts {
			candidate := strings.TrimSpace(part)
			if strings.HasPrefix(strings.ToUpper(candidate), "V") {
				label = candidate
				break
			}
		}
	}
	if label == "" {
		return "", ""
	}
	version := strings.TrimPrefix(strings.TrimPrefix(label, "V"), "v")
	version = strings.TrimSuffix(version, ".zip")
	version = strings.TrimSuffix(version, ".bin")
	return label, normalizeVersionValue(version)
}

func buildDownloadVersion(name string, localVersion string) string {
	text := strings.TrimSpace(localVersion)
	switch name {
	case versionUpdateRSS:
		fsmRequest := buildControllerUpdateRequestVersion(resolveVersionConfig([]string{"FSMVersion", "FsmVersion"}, "2.0.0"), "FSM_200", "FSM")
		if parts := strings.Split(fsmRequest, "_"); len(parts) >= 3 && fsmRequest != "FSM_200" {
			return parts[0] + "_" + parts[1] + "_Linux_FruSort"
		}
		return "Linux_FruitSort200.48"
	case versionUpdateFSM:
		return buildControllerUpdateRequestVersion(text, "FSM_200", "FSM")
	case versionUpdateWAM:
		if strings.TrimSpace(text) == "" || strings.HasPrefix(strings.ToUpper(text), "V") || !strings.Contains(text, "_") {
			return "WAM_200"
		}
		parts := strings.Split(text, "_")
		if len(parts) >= 2 {
			return parts[0] + "_" + parts[1] + "_Linux_WAM"
		}
		return "WAM_200"
	default:
		return text
	}
}

func checkUpdateVersion(updateVersion string, localVersion string) bool {
	updateParts := versionNumberParts(updateVersion)
	localParts := versionNumberParts(localVersion)
	if len(updateParts) != 3 || len(localParts) != 3 {
		return false
	}
	needRollback := false
	for index := 0; index < len(updateParts); index++ {
		if index < len(updateParts)-1 {
			if updateParts[index] > localParts[index] {
				return true
			}
			if updateParts[index] < localParts[index] {
				needRollback = true
			}
			continue
		}
		if needRollback {
			return false
		}
		return updateParts[index] > localParts[index]
	}
	return false
}

func versionNumberParts(value string) []int {
	normalized := normalizeVersionValue(value)
	parts := strings.Split(normalized, ".")
	if len(parts) < 3 {
		return []int{}
	}
	result := make([]int, 0, 3)
	for index := 0; index < 3; index++ {
		number, err := strconv.Atoi(parts[index])
		if err != nil {
			return []int{}
		}
		result = append(result, number)
	}
	return result
}

func normalizeVersionValue(value string) string {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(strings.TrimPrefix(text, "V"), "v")
	text = strings.TrimSuffix(text, ".zip")
	text = strings.TrimSuffix(text, ".bin")
	if strings.Contains(text, "_") {
		parts := strings.Split(text, "_")
		if len(parts) > 0 {
			text = parts[len(parts)-1]
			text = strings.TrimPrefix(strings.TrimPrefix(text, "V"), "v")
		}
	}
	text = strings.Trim(text, "\x00 ")
	if text == "" {
		return "0.0.0"
	}
	return text
}

func resolveVersionDownloadDir() (string, error) {
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
	return filepath.Join(baseDir, "updates", resolveDeviceConfigVersion()), nil
}

func sanitizeVersionDownloadFileName(fileName string, name string, version string) string {
	text := strings.TrimSpace(fileName)
	if text == "" || text != filepath.Base(text) || strings.ContainsAny(text, `/\`) {
		text = strings.ToUpper(strings.TrimSpace(name)) + "_" + sanitizeDeviceConfigPathPart(version) + ".bin"
	}
	return text
}
