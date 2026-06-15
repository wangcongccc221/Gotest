package tcp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"gotest/ohos/database"
)

const (
	cTCPFruitMajorConfigName   = "水果种类大类"
	cTCPFruitMajorEnConfigName = "水果种类大类en"

	cTCPDefaultFruitMajorConfig = "1.脐橙(1-8);2.橘子(9-16);3.柠檬(17-24);4.蜜柚(25-32);5.柿子(33-40);6.雪梨(41-48);7.石榴(49-56);8.土豆(57-64);9.芒果(65-72);10.瓜果(73-80);11.猕猴桃(81-88);12.苹果(89-96);13.牛油果(97-104);14.百香果(105-112);15.桃子(113-120);16.山竹(121-128);17.芭乐(129-136);18.辣椒(137-144);19.凤梨(145-152);20.火龙果(153-160);21.金桔(161-168);22.樱桃(169-176);23.西梅(177-184);24.蓝莓(185-192);25.李子(193-200);26.枣子(201-208);27.圣女果(209-216);28.蜜桔(217-224);29.荔枝(225-232);"
	cTCPDefaultFruitMajorEn     = "1.Orange(1-8);2.Tangerine(9-16);3.Lemon(17-24);4.HoneyPomelo(25-32);5.Persimmon(33-40);6.SnowPear(41-48);7.Pomegranate(49-56);8.Potato(57-64);9.Mango(65-72);10.MelonFruit(73-80);11.Kiwifruit(81-88);12.Apple(89-96);13.Avocado(97-104);14.PassionfFruit(105-112);15.Peach(113-120);16.Mangosteen(121-128);17.Guala(129-136);18.Pepper(137-144);19.Pineapple(145-152);20.Pitaya(153-160);21.Kumquat(161-168);22.Cherry(169-176);23.PrunusMume(177-184);24.BlueBerry(185-192);25.Plum(193-200);26.Jujube(201-208);27.CherryTomato(209-216);28.Tangerine(217-224);29.Litchi(225-232);"
)

type FruitTypeConfigInfo struct {
	MajorTypes         string            `json:"majorTypes"`
	MajorTypesEn       string            `json:"majorTypesEn"`
	SelectedFruitTypes string            `json:"selectedFruitTypes"`
	SubTypeConfigs     map[string]string `json:"subTypeConfigs"`
}

var (
	defaultFruitSubTypeConfigs = map[string]string{
		"1.脐橙(1-8)":     "新鲜脐橙;储藏橙;南非脐橙;黄绿橙;夏橙;",
		"2.橘子(9-16)":    "南丰蜜桔;黄绿桔;橘柚;Tambore;",
		"3.柠檬(17-24)":   "南安岳柠檬;绿圆柠檬;",
		"4.蜜柚(25-32)":   "中粮蜜柚;白柚;黄绿柚;",
		"5.柿子(33-40)":   "甜柿;",
		"6.雪梨(41-48)":   "鸭梨;",
		"7.石榴(49-56)":   "百香果;",
		"11.猕猴桃(81-88)": "四川猕猴桃;",
	}

	lastFruitTypeConfigInfoSnapshot   FruitTypeConfigInfo
	lastFruitTypeConfigInfoFSMID      int32
	lastFruitTypeConfigInfoSnapshotOK bool
	lastFruitTypeConfigInfoSnapshotMu sync.RWMutex
)

func ReadFruitTypeConfigInfoFromLocalConfig() (FruitTypeConfigInfo, error) {
	info := defaultFruitTypeConfigInfo()

	majorTypes, err := database.GetConfigValue(cTCPFruitMajorConfigName)
	if err != nil {
		return FruitTypeConfigInfo{}, fmt.Errorf("read %s: %w", cTCPFruitMajorConfigName, err)
	}
	if strings.TrimSpace(majorTypes) != "" {
		info.MajorTypes = majorTypes
	}

	majorTypesEn, err := database.GetConfigValue(cTCPFruitMajorEnConfigName)
	if err != nil {
		return FruitTypeConfigInfo{}, fmt.Errorf("read %s: %w", cTCPFruitMajorEnConfigName, err)
	}
	if strings.TrimSpace(majorTypesEn) != "" {
		info.MajorTypesEn = majorTypesEn
	}

	selectedFruitTypes, err := database.GetConfigValue(cTCPSelectedFruitTypesConfigName)
	if err != nil {
		return FruitTypeConfigInfo{}, fmt.Errorf("read %s: %w", cTCPSelectedFruitTypesConfigName, err)
	}
	if strings.TrimSpace(selectedFruitTypes) != "" {
		info.SelectedFruitTypes = selectedFruitTypes
	}

	for _, majorType := range splitSemicolonConfig(info.MajorTypes) {
		subTypes, err := database.GetConfigValue(majorType)
		if err != nil {
			return FruitTypeConfigInfo{}, fmt.Errorf("read %s: %w", majorType, err)
		}
		if strings.TrimSpace(subTypes) == "" {
			subTypes = defaultFruitSubTypeConfigs[majorType]
		}
		info.SubTypeConfigs[majorType] = subTypes
	}

	return normalizeFruitTypeConfigInfo(info), nil
}

func SaveFruitTypeConfigInfoToLocalConfig(fsmID int32, info FruitTypeConfigInfo) error {
	info = normalizeFruitTypeConfigInfo(info)

	values := map[string]string{
		cTCPFruitMajorConfigName:         info.MajorTypes,
		cTCPFruitMajorEnConfigName:       info.MajorTypesEn,
		cTCPSelectedFruitTypesConfigName: info.SelectedFruitTypes,
	}
	keys := make([]string, 0, len(info.SubTypeConfigs))
	for key := range info.SubTypeConfigs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		values[key] = info.SubTypeConfigs[key]
	}

	if err := database.SaveConfigValues(values); err != nil {
		setCTCPServerLastMessage("水果配置事务保存失败: %v", err)
		return err
	}
	saved, err := ReadFruitTypeConfigInfoFromLocalConfig()
	if err != nil {
		setCTCPServerLastMessage("水果配置写后读取失败: %v", err)
		return err
	}
	if !reflect.DeepEqual(normalizeFruitTypeConfigInfo(saved), info) {
		err := fmt.Errorf("fruit type config readback mismatch")
		setCTCPServerLastMessage("水果配置写后校验失败: %v", err)
		return err
	}

	setCTCPServerLastMessage(
		"水果配置已保存并校验: fsmId=0x%04X, majorTypes=%d, selectedFruitTypesLen=%d, subTypeKeys=%d",
		uint32(fsmID),
		len(splitSemicolonConfig(info.MajorTypes)),
		len(info.SelectedFruitTypes),
		len(info.SubTypeConfigs),
	)
	return nil
}

func defaultFruitTypeConfigInfo() FruitTypeConfigInfo {
	info := FruitTypeConfigInfo{
		MajorTypes:         cTCPDefaultFruitMajorConfig,
		MajorTypesEn:       cTCPDefaultFruitMajorEn,
		SelectedFruitTypes: cTCPDefaultSelectedFruitTypes,
		SubTypeConfigs:     make(map[string]string),
	}
	for key, value := range defaultFruitSubTypeConfigs {
		info.SubTypeConfigs[key] = value
	}
	return info
}

func normalizeFruitTypeConfigInfo(info FruitTypeConfigInfo) FruitTypeConfigInfo {
	info.MajorTypes = strings.TrimSpace(info.MajorTypes)
	if info.MajorTypes == "" {
		info.MajorTypes = cTCPDefaultFruitMajorConfig
	}
	info.MajorTypesEn = strings.TrimSpace(info.MajorTypesEn)
	if info.MajorTypesEn == "" {
		info.MajorTypesEn = cTCPDefaultFruitMajorEn
	}
	info.SelectedFruitTypes = strings.TrimSpace(info.SelectedFruitTypes)
	if info.SelectedFruitTypes == "" {
		info.SelectedFruitTypes = cTCPDefaultSelectedFruitTypes
	}
	if info.SubTypeConfigs == nil {
		info.SubTypeConfigs = make(map[string]string)
	}

	normalizedSubTypes := make(map[string]string)
	for _, majorType := range splitSemicolonConfig(info.MajorTypes) {
		value := strings.TrimSpace(info.SubTypeConfigs[majorType])
		if value == "" {
			value = strings.TrimSpace(defaultFruitSubTypeConfigs[majorType])
		}
		normalizedSubTypes[majorType] = value
	}
	for key, value := range info.SubTypeConfigs {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := normalizedSubTypes[key]; !ok {
			normalizedSubTypes[key] = strings.TrimSpace(value)
		}
	}
	info.SubTypeConfigs = normalizedSubTypes
	return info
}

func splitSemicolonConfig(text string) []string {
	parts := strings.Split(text, ";")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func setLastFruitTypeConfigInfoSnapshot(fsmID int32, info FruitTypeConfigInfo) {
	lastFruitTypeConfigInfoSnapshotMu.Lock()
	lastFruitTypeConfigInfoSnapshot = normalizeFruitTypeConfigInfo(info)
	lastFruitTypeConfigInfoFSMID = fsmID
	lastFruitTypeConfigInfoSnapshotOK = true
	lastFruitTypeConfigInfoSnapshotMu.Unlock()
}

func latestFruitTypeConfigInfoSnapshot() (int32, FruitTypeConfigInfo, bool) {
	lastFruitTypeConfigInfoSnapshotMu.RLock()
	defer lastFruitTypeConfigInfoSnapshotMu.RUnlock()
	return lastFruitTypeConfigInfoFSMID, lastFruitTypeConfigInfoSnapshot, lastFruitTypeConfigInfoSnapshotOK
}
