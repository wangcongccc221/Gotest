package database

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type DeviceInfoModel struct {
	DeviceID       string `json:"DeviceID"`
	DeviceName     string `json:"DeviceName"`
	Customer       string `json:"Customer"`
	Variety        string `json:"Variety"`
	ProductionDate string `json:"ProductionDate"`
	ExpirationDate string `json:"ExpirationDate"`
	Principal      string `json:"Principal"`
	Tel            string `json:"Tel"`
	CountryID      int    `json:"CountryID"`
	StateID        int    `json:"StateID"`
	CityID         int    `json:"CityID"`
	RegionID       int    `json:"RegionID"`
	Country        string `json:"Country"`
	State          string `json:"State"`
	City           string `json:"City"`
	Region         string `json:"Region"`
	DetailAddress  string `json:"DetailAddress"`
	IPMVersion     string `json:"IPMVersion"`
	HMIVersion     string `json:"HMIVersion"`
	SCMVersion     string `json:"SCMVersion"`
	FSMVersion     string `json:"FSMVersion"`
	DWMVersion     string `json:"DWMVersion"`
	WAMVersion     string `json:"WAMVersion"`
	IQMVersion     string `json:"IQMVersion"`
	BDMVersion     string `json:"BDMVersion"`
	Debugger       string `json:"Debugger"`
	Updater        string `json:"Updater"`
	Remark         string `json:"Remark"`
}

type DeviceInfoLoadResult struct {
	Info       DeviceInfoModel       `json:"Info"`
	Locations  DeviceInfoLocationSet `json:"Locations"`
	Registered bool                  `json:"Registered"`
	Message    string                `json:"Message"`
}

type DeviceInfoLocationSet struct {
	Countries []DeviceInfoLocation `json:"Countries"`
	States    []DeviceInfoLocation `json:"States"`
	Cities    []DeviceInfoLocation `json:"Cities"`
	Regions   []DeviceInfoLocation `json:"Regions"`
}

type DeviceInfoLocation struct {
	FID       int    `json:"FID"`
	FName     string `json:"FName"`
	FParentID int    `json:"FParentID"`
	FLevel    int    `json:"FLevel"`
}

type deviceInfoLocationRequest struct {
	FID int `json:"FID"`
}

type deviceInfoCloudCustomer struct {
	FCustomerName string `json:"FCustomerName"`
	FDeivceCode   string `json:"FDeivceCode"`
	FDeviceName   string `json:"FDeviceName"`
	FLeaveDate    string `json:"FLeaveDate"`
	FDueDate      string `json:"FDueDate"`
	FCountryID    int    `json:"FCountryID"`
	FStateID      int    `json:"FStateID"`
	FCityID       int    `json:"FCityID"`
	FRegionID     int    `json:"FRegionID"`
	FAddress      string `json:"FAddress"`
	FContacts     string `json:"FContacts"`
	FPhone        string `json:"FPhone"`
	FFruitName    string `json:"FFruitName"`
	FFSMVersion   string `json:"FFSMVersion"`
	FIPMVersion   string `json:"FIPMVersion"`
	FHMIVersion   string `json:"FHMIVersion"`
	FSCMVersion   string `json:"FSCMVersion"`
	FIQMVersion   string `json:"FIQMVersion"`
	FDWMVersion   string `json:"FDWMVersion"`
	FBDMVersion   string `json:"FBDMVersion"`
	FWAMVersion   string `json:"FWAMVersion"`
	FDebugger     string `json:"FDebugger"`
	FUpdater      string `json:"FUpdater"`
	FRemark       string `json:"FRemark"`
	FComputerID   string `json:"FComputerID"`
	FComputerUser string `json:"FComputerUser"`
	FOperator     string `json:"FOperator"`
}

func registerDeviceInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/DeviceInfo")
	group.POST("/GetDeviceInfo", handleDeviceInfoGet)
	group.POST("/GetLocationList", handleDeviceInfoGetLocations)
	group.POST("/GetByDeviceCode", handleDeviceInfoGetByDeviceCode)
	group.POST("/Register", handleDeviceInfoRegister)
}

func handleDeviceInfoGet(ctx *gin.Context) {
	result, err := LoadDeviceInfo()
	if err != nil {
		fmt.Printf("%s DeviceInfo GetDeviceInfo failed: %v\n", time.Now().Format("15:04:05.000"), err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func handleDeviceInfoGetLocations(ctx *gin.Context) {
	var request deviceInfoLocationRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	locations, err := GetDeviceInfoLocations(request.FID)
	if err != nil {
		fmt.Printf("%s DeviceInfo GetLocationList failed: parent=%d err=%v\n",
			time.Now().Format("15:04:05.000"), request.FID, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, locations)
}

func handleDeviceInfoGetByDeviceCode(ctx *gin.Context) {
	var request DeviceInfoModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	info, err := GetDeviceInfoByDeviceCode(request.DeviceID)
	if err != nil {
		fmt.Printf("%s DeviceInfo GetByDeviceCode failed: device=%s err=%v\n",
			time.Now().Format("15:04:05.000"), request.DeviceID, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, info)
}

func handleDeviceInfoRegister(ctx *gin.Context) {
	var request DeviceInfoModel
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	info, err := RegisterDeviceInfo(request)
	if err != nil {
		fmt.Printf("%s DeviceInfo Register failed: device=%s err=%v\n",
			time.Now().Format("15:04:05.000"), request.DeviceID, err)
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, info)
}

func LoadDeviceInfo() (DeviceInfoLoadResult, error) {
	info := defaultDeviceInfoModel()
	result := DeviceInfoLoadResult{Info: info}

	countries, err := GetDeviceInfoLocations(0)
	if err != nil {
		result.Message = err.Error()
	} else {
		result.Locations.Countries = countries
	}

	if secret, err := GetConfigValuePreferNonEmpty("SecretKey"); err == nil && strings.TrimSpace(secret) != "" {
		if err := updateCloudDeviceInfo(info); err != nil {
			fmt.Printf("%s DeviceInfo UpdateDeviceInfo warning: %v\n", time.Now().Format("15:04:05.000"), err)
		}
		cloudInfo, err := getCloudCustomerDeviceInfo()
		if err != nil {
			result.Message = err.Error()
			fmt.Printf("%s DeviceInfo GetCustomerDeviceInfo warning: %v\n", time.Now().Format("15:04:05.000"), err)
		} else {
			result.Info = mergeDeviceInfoWithCloud(info, cloudInfo)
			result.Registered = true
			if result.Info.DeviceID != "" {
				_ = SaveConfigValue("RSSDeviceCode", result.Info.DeviceID)
			}
			result.Locations = loadLocationSetForInfo(result.Info, result.Locations.Countries)
		}
	} else if code, err := GetConfigValuePreferNonEmpty("RSSDeviceCode"); err == nil && strings.TrimSpace(code) != "" {
		result.Info.DeviceID = strings.TrimSpace(code)
	}

	fmt.Printf("%s DeviceInfo GetDeviceInfo success: registered=%v, device=%s\n",
		time.Now().Format("15:04:05.000"), result.Registered, result.Info.DeviceID)
	return result, nil
}

func GetDeviceInfoLocations(parentID int) ([]DeviceInfoLocation, error) {
	request := deviceInfoLocationRequest{FID: parentID}
	postData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	response, err := callDeviceInfoCloudNoVerify("Customer/GetLocationList", string(postData))
	if err != nil {
		return nil, err
	}
	var rows []DeviceInfoLocation
	if strings.TrimSpace(response.Data) == "" {
		return rows, nil
	}
	if err := json.Unmarshal([]byte(response.Data), &rows); err != nil {
		return nil, fmt.Errorf("parse device locations: %w", err)
	}
	fmt.Printf("%s DeviceInfo GetLocationList success: parent=%d, count=%d\n",
		time.Now().Format("15:04:05.000"), parentID, len(rows))
	return rows, nil
}

func GetDeviceInfoByDeviceCode(deviceID string) (DeviceInfoModel, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return DeviceInfoModel{}, errors.New("DeviceID is required")
	}
	response, err := callDeviceInfoCloudNoVerify("Customer/GetCustomerDeviceInfoByDeviceCode", deviceID)
	if err != nil {
		return DeviceInfoModel{}, err
	}
	var cloud deviceInfoCloudCustomer
	if err := json.Unmarshal([]byte(response.Data), &cloud); err != nil {
		return DeviceInfoModel{}, fmt.Errorf("parse device info by code: %w", err)
	}
	info := mergeDeviceInfoWithCloud(defaultDeviceInfoModel(), cloud)
	if info.DeviceID == "" {
		info.DeviceID = deviceID
	}
	return info, nil
}

func RegisterDeviceInfo(info DeviceInfoModel) (DeviceInfoModel, error) {
	info.DeviceID = strings.TrimSpace(info.DeviceID)
	if info.DeviceID == "" {
		return DeviceInfoModel{}, errors.New("DeviceID is required")
	}
	cloud := cloudCustomerFromDeviceInfo(info)
	postData, err := json.Marshal(cloud)
	if err != nil {
		return DeviceInfoModel{}, err
	}
	response, err := callDeviceInfoCloudNoVerify("Customer/Register", string(postData))
	if err != nil {
		return DeviceInfoModel{}, err
	}
	secret := strings.TrimSpace(response.Data)
	if secret != "" {
		if err := SaveConfigValue("SecretKey", secret); err != nil {
			return DeviceInfoModel{}, err
		}
	}
	if err := SaveConfigValue("RSSDeviceCode", info.DeviceID); err != nil {
		return DeviceInfoModel{}, err
	}
	if strings.TrimSpace(info.ExpirationDate) != "" {
		_ = SaveConfigValue("DeviceUseEndDate", strings.TrimSpace(info.ExpirationDate))
	}
	fmt.Printf("%s DeviceInfo Register success: device=%s, secret=%v\n",
		time.Now().Format("15:04:05.000"), info.DeviceID, secret != "")
	return info, nil
}

func updateCloudDeviceInfo(info DeviceInfoModel) error {
	cloud := cloudCustomerFromDeviceInfo(info)
	postData, err := json.Marshal(cloud)
	if err != nil {
		return err
	}
	_, err = callDeviceConfigCloud("Customer/UpdateDeviceInfo", string(postData))
	return err
}

func getCloudCustomerDeviceInfo() (deviceInfoCloudCustomer, error) {
	response, err := callDeviceConfigCloud("Customer/GetCustomerDeviceInfo", "")
	if err != nil {
		return deviceInfoCloudCustomer{}, err
	}
	var cloud deviceInfoCloudCustomer
	if strings.TrimSpace(response.Data) == "" {
		return cloud, nil
	}
	if err := json.Unmarshal([]byte(response.Data), &cloud); err != nil {
		return deviceInfoCloudCustomer{}, fmt.Errorf("parse customer device info: %w", err)
	}
	return cloud, nil
}

func callDeviceInfoCloudNoVerify(endpoint string, postData string) (deviceConfigCloudEnvelope, error) {
	baseURL, err := resolveDeviceConfigCloudBaseURL()
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}
	requestURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBufferString(postData))
	if err != nil {
		return deviceConfigCloudEnvelope{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("devicetype", deviceConfigCloudDeviceType)
	request.Header.Set("api-version", deviceConfigCloudAPIVersion)
	request.Header.Set("language", resolveDeviceConfigLanguage())

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

func defaultDeviceInfoModel() DeviceInfoModel {
	rss := resolveVersionConfig([]string{"RSSVersion", "RssVersion", "VERSION_SHOW", "VersionShow"}, "2.0.0")
	info := DeviceInfoModel{
		DeviceID:       resolveConfigString([]string{"RSSDeviceCode", "DeviceCode"}, ""),
		ProductionDate: resolveConfigString([]string{"ProductionDate", "LeaveDate"}, "2000-01-01"),
		ExpirationDate: resolveConfigString([]string{"DeviceUseEndDate", "ExpirationDate", "DueDate"}, "2099-01-01"),
		IPMVersion:     resolveConfigString([]string{"IPMVersion"}, "IPM-// 00:00:00"),
		HMIVersion:     buildHMIVersionLabel(rss),
		SCMVersion:     resolveConfigString([]string{"SCMVersion"}, ""),
		FSMVersion:     resolveConfigString([]string{"FSMVersion", "FsmVersion"}, ""),
		DWMVersion:     resolveConfigString([]string{"DWMVersion"}, ""),
		WAMVersion:     resolveConfigString([]string{"WAMVersion", "WamVersion"}, ""),
		IQMVersion:     resolveConfigString([]string{"IQMVersion"}, ""),
		BDMVersion:     resolveConfigString([]string{"BDMVersion"}, ""),
	}
	return info
}

func resolveConfigString(keys []string, fallback string) string {
	for _, key := range keys {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func buildHMIVersionLabel(version string) string {
	text := strings.TrimSpace(version)
	if text == "" {
		return "HMI-V2.0.0"
	}
	if strings.Contains(text, "_") {
		return text
	}
	if strings.HasPrefix(strings.ToUpper(text), "HMI-") {
		return text
	}
	if strings.HasPrefix(strings.ToUpper(text), "V") {
		return "HMI-" + text
	}
	return "HMI-V" + text
}

func mergeDeviceInfoWithCloud(base DeviceInfoModel, cloud deviceInfoCloudCustomer) DeviceInfoModel {
	info := base
	info.DeviceID = preferString(cloud.FDeivceCode, info.DeviceID)
	info.DeviceName = preferString(cloud.FDeviceName, info.DeviceName)
	info.Customer = preferString(cloud.FCustomerName, info.Customer)
	info.Variety = preferString(cloud.FFruitName, info.Variety)
	info.ProductionDate = preferDateString(cloud.FLeaveDate, info.ProductionDate)
	info.ExpirationDate = preferDateString(cloud.FDueDate, info.ExpirationDate)
	info.Principal = preferString(cloud.FContacts, info.Principal)
	info.Tel = preferString(cloud.FPhone, info.Tel)
	info.CountryID = cloud.FCountryID
	info.StateID = cloud.FStateID
	info.CityID = cloud.FCityID
	info.RegionID = cloud.FRegionID
	info.DetailAddress = preferString(cloud.FAddress, info.DetailAddress)
	info.IPMVersion = preferString(cloud.FIPMVersion, info.IPMVersion)
	info.HMIVersion = preferString(cloud.FHMIVersion, info.HMIVersion)
	info.SCMVersion = preferString(cloud.FSCMVersion, info.SCMVersion)
	info.FSMVersion = preferString(cloud.FFSMVersion, info.FSMVersion)
	info.DWMVersion = preferString(cloud.FDWMVersion, info.DWMVersion)
	info.WAMVersion = preferString(cloud.FWAMVersion, info.WAMVersion)
	info.IQMVersion = preferString(cloud.FIQMVersion, info.IQMVersion)
	info.BDMVersion = preferString(cloud.FBDMVersion, info.BDMVersion)
	info.Debugger = preferString(cloud.FDebugger, info.Debugger)
	info.Updater = preferString(cloud.FUpdater, info.Updater)
	info.Remark = preferString(cloud.FRemark, info.Remark)
	return info
}

func cloudCustomerFromDeviceInfo(info DeviceInfoModel) deviceInfoCloudCustomer {
	return deviceInfoCloudCustomer{
		FCustomerName: info.Customer,
		FDeivceCode:   info.DeviceID,
		FDeviceName:   info.DeviceName,
		FComputerID:   resolveHostComputerID(),
		FComputerUser: resolveHostName(),
		FLeaveDate:    normalizeDeviceInfoDate(info.ProductionDate),
		FDueDate:      normalizeDeviceInfoDate(info.ExpirationDate),
		FCountryID:    info.CountryID,
		FStateID:      info.StateID,
		FCityID:       info.CityID,
		FRegionID:     info.RegionID,
		FAddress:      info.DetailAddress,
		FContacts:     info.Principal,
		FPhone:        info.Tel,
		FFruitName:    info.Variety,
		FFSMVersion:   info.FSMVersion,
		FIPMVersion:   info.IPMVersion,
		FHMIVersion:   info.HMIVersion,
		FSCMVersion:   info.SCMVersion,
		FIQMVersion:   info.IQMVersion,
		FDWMVersion:   info.DWMVersion,
		FBDMVersion:   info.BDMVersion,
		FWAMVersion:   info.WAMVersion,
		FDebugger:     info.Debugger,
		FUpdater:      info.Updater,
		FRemark:       info.Remark,
	}
}

func loadLocationSetForInfo(info DeviceInfoModel, countries []DeviceInfoLocation) DeviceInfoLocationSet {
	set := DeviceInfoLocationSet{Countries: countries}
	if info.CountryID > 0 {
		set.States, _ = GetDeviceInfoLocations(info.CountryID)
	}
	if info.StateID > 0 {
		set.Cities, _ = GetDeviceInfoLocations(info.StateID)
	}
	if info.CityID > 0 {
		set.Regions, _ = GetDeviceInfoLocations(info.CityID)
	}
	return set
}

func preferString(value string, fallback string) string {
	text := strings.TrimSpace(value)
	if text != "" {
		return text
	}
	return fallback
}

func preferDateString(value string, fallback string) string {
	text := normalizeDeviceInfoDate(value)
	if text != "" {
		return text
	}
	return fallback
}

func normalizeDeviceInfoDate(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, text, databaseChinaLocation); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	if len(text) >= 10 {
		return text[:10]
	}
	return text
}

func resolveHostName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func resolveHostComputerID() string {
	return resolveConfigString([]string{"ComputerID"}, "")
}
