package database

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	projectPasswordDeviceType      = "BDMV1.0"
	projectPasswordAPIVersion      = "2.0"
	projectPasswordModernRemoteURL = "https://sd.reemoon.com/Api/"
)

type projectPasswordRequest struct {
	FPhone        string `json:"FPhone"`
	FValidateCode string `json:"FValidateCode,omitempty"`
}

type projectPasswordCloudEnvelope struct {
	ReturnCode    int             `json:"returnCode"`
	ReturnMessage string          `json:"returnMessage"`
	Data          json.RawMessage `json:"data"`
}

func CreateProjectPassword(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return errors.New("手机号输入错误！")
	}
	_, err := callProjectPasswordCloud("FruitSorting/CrateProjectPassword", projectPasswordRequest{
		FPhone: phone,
	})
	return err
}

func ValidateProjectPassword(phone string, validateCode string) error {
	phone = strings.TrimSpace(phone)
	validateCode = strings.TrimSpace(validateCode)
	if validateCode == "" {
		return errors.New("请填写验证码")
	}
	if phone == "" {
		return validateLocalProjectPassword(validateCode)
	}
	_, err := callProjectPasswordCloud("FruitSorting/ValidateProjectPassword", projectPasswordRequest{
		FPhone:        phone,
		FValidateCode: validateCode,
	})
	return err
}

func validateLocalProjectPassword(validateCode string) error {
	expected := strings.ToUpper(md5LowerHex(time.Now().Format("20060102")))
	if len(expected) > 6 {
		expected = expected[:6]
	}
	if expected != validateCode {
		return errors.New("密码输入错误！")
	}
	return nil
}

func callProjectPasswordCloud(endpoint string, payload projectPasswordRequest) (projectPasswordCloudEnvelope, error) {
	baseURLs, err := resolveProjectPasswordCloudBaseURLs()
	if err != nil {
		return projectPasswordCloudEnvelope{}, err
	}
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return projectPasswordCloudEnvelope{}, fmt.Errorf("marshal project password request: %w", err)
	}

	var lastErr error
	for _, baseURL := range baseURLs {
		envelope, err := callProjectPasswordCloudAtBase(baseURL, endpoint, requestBody)
		if err == nil {
			return envelope, nil
		}
		if _, ok := err.(projectPasswordCloudBusinessError); ok {
			return projectPasswordCloudEnvelope{}, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return projectPasswordCloudEnvelope{}, lastErr
	}
	return projectPasswordCloudEnvelope{}, fmt.Errorf("project password cloud %s has no available base URL", endpoint)
}

func callProjectPasswordCloudAtBase(baseURL string, endpoint string, requestBody []byte) (projectPasswordCloudEnvelope, error) {
	requestURL := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(requestBody))
	if err != nil {
		return projectPasswordCloudEnvelope{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("devicetype", projectPasswordDeviceType)
	request.Header.Set("api-version", projectPasswordAPIVersion)
	request.Header.Set("language", resolveDeviceConfigLanguage())

	response, err := deviceConfigHTTPClient.Do(request)
	if err != nil {
		return projectPasswordCloudEnvelope{}, fmt.Errorf("call project password cloud %s failed: %w", endpoint, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return projectPasswordCloudEnvelope{}, fmt.Errorf("project password cloud %s returned HTTP %d", endpoint, response.StatusCode)
	}

	var envelope projectPasswordCloudEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return projectPasswordCloudEnvelope{}, fmt.Errorf("parse project password cloud %s response: %w", endpoint, err)
	}
	if envelope.ReturnCode != 1 {
		return projectPasswordCloudEnvelope{}, projectPasswordCloudBusinessError{
			err: projectPasswordCloudError(endpoint, envelope),
		}
	}
	return envelope, nil
}

type projectPasswordCloudBusinessError struct {
	err error
}

func (e projectPasswordCloudBusinessError) Error() string {
	return e.err.Error()
}

func resolveProjectPasswordCloudBaseURLs() ([]string, error) {
	remoteURL, err := resolveProjectPasswordRemoteURL()
	if err != nil {
		return nil, err
	}
	candidates := make([]string, 0, 3)
	if canConnectProjectPasswordURL(deviceConfigCloudLocalURL, 750*time.Millisecond) {
		candidates = appendProjectPasswordBaseURL(candidates, deviceConfigCloudLocalURL)
	}
	if canConnectProjectPasswordURL(remoteURL, 750*time.Millisecond) {
		candidates = appendProjectPasswordBaseURL(candidates, remoteURL)
	}
	if !strings.EqualFold(remoteURL, projectPasswordModernRemoteURL) &&
		canConnectProjectPasswordURL(projectPasswordModernRemoteURL, 750*time.Millisecond) {
		candidates = appendProjectPasswordBaseURL(candidates, projectPasswordModernRemoteURL)
	}
	if len(candidates) == 0 {
		candidates = appendProjectPasswordBaseURL(candidates, remoteURL)
	}
	return candidates, nil
}

func appendProjectPasswordBaseURL(candidates []string, rawURL string) []string {
	baseURL := normalizeProjectPasswordCloudBaseURL(rawURL)
	for _, candidate := range candidates {
		if strings.EqualFold(candidate, baseURL) {
			return candidates
		}
	}
	return append(candidates, baseURL)
}

func resolveProjectPasswordRemoteURL() (string, error) {
	for _, key := range []string{"DeviceConfigCloudBaseUrl", "CloudApiUrl", "CurrentUrl", "BaseUrl"} {
		value, err := GetConfigValuePreferNonEmpty(key)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return normalizeProjectPasswordCloudBaseURL(value), nil
		}
	}
	return projectPasswordModernRemoteURL, nil
}

func normalizeProjectPasswordCloudBaseURL(value string) string {
	text := normalizeDeviceConfigCloudBaseURL(value)
	if strings.EqualFold(text, deviceConfigCloudRemoteURL) {
		return projectPasswordModernRemoteURL
	}
	return text
}

func canConnectProjectPasswordURL(rawURL string, timeout time.Duration) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	port := parsed.Port()
	if port == "" {
		if strings.EqualFold(parsed.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(parsed.Hostname(), port), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func projectPasswordCloudError(endpoint string, envelope projectPasswordCloudEnvelope) error {
	message := strings.TrimSpace(envelope.ReturnMessage)
	dataText := projectPasswordDataText(envelope.Data)
	if message == "" {
		message = dataText
	} else if dataText != "" {
		message += dataText
	}
	if message == "" {
		message = fmt.Sprintf("project password cloud %s failed", endpoint)
	}
	return errors.New(message)
}

func projectPasswordDataText(data json.RawMessage) string {
	text := strings.TrimSpace(string(data))
	if text == "" || text == "null" {
		return ""
	}
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		return strings.TrimSpace(stringValue)
	}
	return text
}
