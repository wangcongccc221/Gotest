package tcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	cTCPProjectSettingsSchemeVersion = 1
)

type ProjectSettingsSchemeMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Version   int    `json:"version"`
}

type ProjectSettingsScheme struct {
	Version            int                    `json:"version"`
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	CreatedAt          string                 `json:"createdAt"`
	UpdatedAt          string                 `json:"updatedAt"`
	StGlobalJSON       string                 `json:"stGlobalJson,omitempty"`
	FruitTypeConfig    FruitTypeConfigInfo    `json:"fruitTypeConfig"`
	LevelAuxConfig     LevelAuxConfigInfo     `json:"levelAuxConfig"`
	ExitInfos          StExitInfos            `json:"exitInfos"`
	ExitDisplay        ExitDisplayInfo        `json:"exitDisplay"`
	ExitAdditionalText ExitAdditionalTextInfo `json:"exitAdditionalText"`
}

func BuildCurrentProjectSettingsSchemeForLocalFile(fsmID int32, name string) (ProjectSettingsScheme, error) {
	now := time.Now().Format(time.RFC3339)
	meta := ProjectSettingsSchemeMeta{
		ID:        newProjectSettingsSchemeID(),
		Name:      normalizeProjectSettingsSchemeName(name),
		CreatedAt: now,
		UpdatedAt: now,
		Version:   cTCPProjectSettingsSchemeVersion,
	}

	scheme, err := buildCurrentProjectSettingsScheme(meta)
	if err != nil {
		return ProjectSettingsScheme{}, err
	}
	setCTCPServerLastMessage("工程设置方案JSON已生成: fsmId=0x%04X, name=%s, id=%s", uint32(fsmID), meta.Name, meta.ID)
	return scheme, nil
}

func ApplyProjectSettingsSchemeJSON(fsmID int32, jsonText string) (ProjectSettingsScheme, error) {
	scheme, err := parseProjectSettingsSchemeJSON(jsonText)
	if err != nil {
		return ProjectSettingsScheme{}, err
	}

	now := time.Now().Format(time.RFC3339)
	if scheme.ID == "" {
		scheme.ID = newProjectSettingsSchemeID()
	}
	if scheme.CreatedAt == "" {
		scheme.CreatedAt = now
	}
	if scheme.UpdatedAt == "" {
		scheme.UpdatedAt = now
	}
	scheme = normalizeProjectSettingsScheme(scheme)

	if err := ApplyProjectSettingsSchemeToLocalConfig(fsmID, scheme); err != nil {
		return ProjectSettingsScheme{}, err
	}

	setCTCPServerLastMessage("工程设置方案已从本地JSON加载: fsmId=0x%04X, name=%s, id=%s", uint32(fsmID), scheme.Name, scheme.ID)
	return scheme, nil
}

func ApplyProjectSettingsSchemeToLocalConfig(fsmID int32, scheme ProjectSettingsScheme) error {
	scheme = normalizeProjectSettingsScheme(scheme)
	if err := SaveFruitTypeConfigInfoToLocalConfig(fsmID, scheme.FruitTypeConfig); err != nil {
		return err
	}
	if err := SaveLevelAuxConfigInfoToLocalConfig(fsmID, scheme.LevelAuxConfig); err != nil {
		return err
	}
	if err := SaveStExitInfosToLocalConfig(fsmID, scheme.ExitInfos); err != nil {
		return err
	}
	if err := SaveExitDisplayInfoToLocalConfig(fsmID, scheme.ExitDisplay); err != nil {
		return err
	}
	if err := SaveExitAdditionalTextInfoToLocalConfig(fsmID, scheme.ExitAdditionalText); err != nil {
		return err
	}

	setLastFruitTypeConfigInfoSnapshot(fsmID, scheme.FruitTypeConfig)
	setLastLevelAuxConfigInfoSnapshot(fsmID, scheme.LevelAuxConfig)
	setLastStExitInfosSnapshot(fsmID, scheme.ExitInfos)
	setLastExitDisplayInfoSnapshot(fsmID, scheme.ExitDisplay)
	setLastExitAdditionalTextInfoSnapshot(fsmID, scheme.ExitAdditionalText)
	publishFruitTypeConfigData(fsmID, scheme.FruitTypeConfig)
	publishLevelAuxConfigData(fsmID, scheme.LevelAuxConfig)
	publishExitInfosData(fsmID, scheme.ExitInfos)
	publishExitDisplayData(fsmID, scheme.ExitDisplay)
	publishExitAdditionalTextData(fsmID, scheme.ExitAdditionalText)
	publishProjectSettingsStGlobalSnapshot(scheme.StGlobalJSON)
	return nil
}

func publishProjectSettingsStGlobalSnapshot(jsonText string) {
	jsonText = strings.TrimSpace(jsonText)
	if jsonText == "" {
		return
	}
	if err := PublishWebSocketJSON(webSocketTopicStGlobal, jsonText); err != nil {
		setCTCPServerLastMessage("工程设置方案 StGlobal 快照推送失败: %v", err)
		return
	}
	setCTCPLastStGlobalFullJSON(jsonText)
}

func formatProjectSettingsSchemeJSON(scheme ProjectSettingsScheme) (string, error) {
	payload, err := json.MarshalIndent(normalizeProjectSettingsScheme(scheme), "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func parseProjectSettingsSchemeJSON(jsonText string) (ProjectSettingsScheme, error) {
	jsonText = strings.TrimSpace(jsonText)
	if jsonText == "" {
		return ProjectSettingsScheme{}, fmt.Errorf("project settings scheme json is empty")
	}
	var scheme ProjectSettingsScheme
	if err := json.Unmarshal([]byte(jsonText), &scheme); err != nil {
		return ProjectSettingsScheme{}, fmt.Errorf("parse project settings scheme json: %w", err)
	}
	return normalizeProjectSettingsScheme(scheme), nil
}

func projectSettingsSchemeFileName(scheme ProjectSettingsScheme) string {
	name := normalizeProjectSettingsSchemeName(scheme.Name)
	name = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "工程方案"
	}
	if strings.HasSuffix(strings.ToLower(name), ".json") {
		return name
	}
	return name + ".json"
}

func buildCurrentProjectSettingsScheme(meta ProjectSettingsSchemeMeta) (ProjectSettingsScheme, error) {
	fruitTypeConfig, err := ReadFruitTypeConfigInfoFromLocalConfig()
	if err != nil {
		return ProjectSettingsScheme{}, err
	}
	levelAuxConfig, err := ReadLevelAuxConfigInfoFromLocalConfig()
	if err != nil {
		return ProjectSettingsScheme{}, err
	}
	exitInfos, ok, err := ReadStExitInfosFromLocalConfig()
	if err != nil {
		return ProjectSettingsScheme{}, err
	}
	if !ok {
		exitInfos = defaultStExitInfos()
	}
	exitDisplay, err := ReadExitDisplayInfoFromLocalConfig()
	if err != nil {
		return ProjectSettingsScheme{}, err
	}
	exitAdditionalText, err := ReadExitAdditionalTextInfoFromLocalConfig()
	if err != nil {
		return ProjectSettingsScheme{}, err
	}

	return normalizeProjectSettingsScheme(ProjectSettingsScheme{
		Version:            cTCPProjectSettingsSchemeVersion,
		ID:                 meta.ID,
		Name:               meta.Name,
		CreatedAt:          meta.CreatedAt,
		UpdatedAt:          meta.UpdatedAt,
		StGlobalJSON:       LastCTCPStGlobalFullJSON(),
		FruitTypeConfig:    fruitTypeConfig,
		LevelAuxConfig:     levelAuxConfig,
		ExitInfos:          exitInfos,
		ExitDisplay:        exitDisplay,
		ExitAdditionalText: exitAdditionalText,
	}), nil
}

func newProjectSettingsSchemeID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%06d", now.Format("20060102150405"), now.Nanosecond()/1000)
}

func normalizeProjectSettingsSchemeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "未命名工程方案"
	}
	if len([]rune(name)) > 80 {
		return string([]rune(name)[:80])
	}
	return name
}

func normalizeProjectSettingsScheme(scheme ProjectSettingsScheme) ProjectSettingsScheme {
	scheme.Version = cTCPProjectSettingsSchemeVersion
	scheme.ID = strings.TrimSpace(scheme.ID)
	scheme.Name = normalizeProjectSettingsSchemeName(scheme.Name)
	scheme.CreatedAt = strings.TrimSpace(scheme.CreatedAt)
	scheme.UpdatedAt = strings.TrimSpace(scheme.UpdatedAt)
	scheme.StGlobalJSON = strings.TrimSpace(scheme.StGlobalJSON)
	scheme.FruitTypeConfig = normalizeFruitTypeConfigInfo(scheme.FruitTypeConfig)
	scheme.LevelAuxConfig = normalizeLevelAuxConfigInfo(scheme.LevelAuxConfig)
	return scheme
}
