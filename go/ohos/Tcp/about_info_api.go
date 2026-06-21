package tcp

import (
	"encoding/json"
	"fmt"
	"gotest/ohos/database"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type aboutAPIEnvelope struct {
	ReturnCode    int    `json:"returnCode"`
	ReturnMessage string `json:"returnMessage"`
	Data          string `json:"data"`
}

type AboutInfo struct {
	SubSystems        []AboutSubSystem `json:"SubSystems"`
	SelectedIndex     int              `json:"SelectedIndex"`
	HC                AboutSection     `json:"HC"`
	IPM               AboutSection     `json:"IPM"`
	FSM               AboutSection     `json:"FSM"`
	WAM               AboutSection     `json:"WAM"`
	IllustrateTitle   string           `json:"IllustrateTitle"`
	IllustrateText    string           `json:"IllustrateText"`
	SourceMessage     string           `json:"SourceMessage"`
	LastStGlobalTime  string           `json:"LastStGlobalTime"`
	LastFSMVersionRaw string           `json:"LastFSMVersionRaw"`
	LastWAMVersionRaw string           `json:"LastWAMVersionRaw"`
}

type AboutSubSystem struct {
	Index int    `json:"Index"`
	Name  string `json:"Name"`
}

type AboutSection struct {
	Title           string `json:"Title"`
	DateOfRelease   string `json:"DateOfRelease"`
	Version         string `json:"Version"`
	SoftwareVersion string `json:"SoftwareVersion"`
	HardwareVersion string `json:"HardwareVersion"`
}

type aboutInfoRequest struct {
	SubSystemIndex int `json:"SubSystemIndex"`
}

type aboutStGlobalSnapshot struct {
	StGlobal   StGlobal
	ReceivedAt time.Time
}

var (
	aboutInfoMu          sync.Mutex
	aboutInfoLastGlobal  aboutStGlobalSnapshot
	aboutInfoHasGlobal   bool
	aboutInfoFSMVersions = map[int]string{}
	aboutInfoWAMVersions = map[int]string{}
)

func registerAboutInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/About")
	group.POST("/GetAboutInfo", handleAboutInfoGet)
}

func handleAboutInfoGet(ctx *gin.Context) {
	var request aboutInfoRequest
	_ = ctx.ShouldBindJSON(&request)
	info := BuildAboutInfoForSubSystem(request.SubSystemIndex)
	aboutAPILog("GetAboutInfo success: subsystem=%d, subsystems=%d, hc=%q, fsm=%q, wam=%q",
		info.SelectedIndex, len(info.SubSystems), info.HC.Version, info.FSM.SoftwareVersion, info.WAM.SoftwareVersion)
	aboutAPIOK(ctx, info)
}

func aboutAPIOK(ctx *gin.Context, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		aboutAPIFail(ctx, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, aboutAPIEnvelope{
		ReturnCode:    1,
		ReturnMessage: "",
		Data:          string(bytes),
	})
}

func aboutAPIFail(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, aboutAPIEnvelope{
		ReturnCode:    0,
		ReturnMessage: message,
		Data:          "",
	})
}

func cacheAboutStGlobal(stg StGlobal) {
	aboutInfoMu.Lock()
	aboutInfoLastGlobal = aboutStGlobalSnapshot{
		StGlobal:   stg,
		ReceivedAt: time.Now(),
	}
	aboutInfoHasGlobal = true
	aboutInfoMu.Unlock()
}

func cacheAboutVersion(head cTCPServerCommandHead, payload []byte) {
	subsysIndex := getSubsysIndex(head.NSrcId)
	if subsysIndex < 0 {
		subsysIndex = 0
	}
	version := strings.TrimRight(string(payload), "\x00\r\n ")
	aboutInfoMu.Lock()
	if head.NCmdId == cmdFSMGetVersion {
		aboutInfoFSMVersions[subsysIndex] = version
	} else if head.NCmdId == cmdWAMVersionInfo {
		aboutInfoWAMVersions[subsysIndex] = version
	}
	aboutInfoMu.Unlock()
}

func BuildAboutInfo() AboutInfo {
	return BuildAboutInfoForSubSystem(0)
}

func BuildAboutInfoForSubSystem(subsysIndex int) AboutInfo {
	snapshot, hasGlobal, fsmVersions, wamVersions := aboutInfoSnapshot()
	rssVersion, compileTime := readLocalControlVersion()
	if rssVersion == "" {
		rssVersion = resolveAboutVersionConfig([]string{"RSSVersion", "RssVersion", "VERSION_SHOW", "VersionShow"}, "2.0.0")
	}
	if compileTime == "" {
		compileTime = resolveAboutVersionConfig([]string{"CompileTime", "HCCompileTime"}, "2000/01/01 08:00:00")
	}

	subsysCount := 1
	ipmDate := "0000/00/00 00:00:00"
	fsmDate := "0000/00/00"
	lastStGlobalTime := ""
	if hasGlobal {
		if snapshot.StGlobal.Sys.NSubsysNum > 0 {
			subsysCount = int(snapshot.StGlobal.Sys.NSubsysNum)
		}
		ipmDate = formatIPMCompileDate(snapshot.StGlobal.CIPMInfo[:])
		fsmDate = formatControllerCompileDate(snapshot.StGlobal.CFSMInfo[:])
		lastStGlobalTime = snapshot.ReceivedAt.Format("2006-01-02 15:04:05")
	}
	if subsysIndex < 0 {
		subsysIndex = 0
	}
	if subsysIndex >= subsysCount {
		subsysIndex = subsysCount - 1
	}

	fsmRaw := preferAboutVersion(fsmVersions[subsysIndex], resolveAboutVersionConfig([]string{"FSMVersionRaw", "FSMVersion", "FsmVersion"}, ""))
	wamRaw := preferAboutVersion(wamVersions[subsysIndex], resolveAboutVersionConfig([]string{"WAMVersionRaw", "WAMVersion", "WamVersion"}, ""))
	fsmSoft, fsmHard := splitAboutControllerVersion(fsmRaw, "FSM")
	wamSoft, wamHard := splitAboutControllerVersion(wamRaw, "WAM")
	hcVersion := buildAboutHCVersion(rssVersion, fsmSoft)
	illustrateTitle := resolveAboutVersionConfig([]string{"AboutIllustrateTitle", "AboutDescriptionTitle"}, "")
	illustrateText := resolveAboutVersionConfig([]string{"AboutIllustrateText", "AboutDescription"}, "")

	source := "cached StGlobal"
	if !hasGlobal {
		source = "fallback config"
	}

	return AboutInfo{
		SubSystems:        buildAboutSubSystems(subsysCount),
		SelectedIndex:     subsysIndex,
		HC:                AboutSection{Title: "HC Infomation", DateOfRelease: compileTime, Version: hcVersion},
		IPM:               AboutSection{Title: "IPM Infomation", DateOfRelease: ipmDate, Version: resolveAboutVersionConfig([]string{"IPMVersionName"}, "IPM5.0")},
		FSM:               AboutSection{Title: "FSM Infomation", DateOfRelease: fsmDate, SoftwareVersion: fsmSoft, HardwareVersion: fsmHard},
		WAM:               AboutSection{Title: "WAM Infomation", DateOfRelease: resolveAboutVersionConfig([]string{"WAMCompileDate"}, "//"), SoftwareVersion: wamSoft, HardwareVersion: wamHard},
		IllustrateTitle:   illustrateTitle,
		IllustrateText:    illustrateText,
		SourceMessage:     source,
		LastStGlobalTime:  lastStGlobalTime,
		LastFSMVersionRaw: fsmRaw,
		LastWAMVersionRaw: wamRaw,
	}
}

func aboutInfoSnapshot() (aboutStGlobalSnapshot, bool, map[int]string, map[int]string) {
	aboutInfoMu.Lock()
	defer aboutInfoMu.Unlock()
	fsm := make(map[int]string, len(aboutInfoFSMVersions))
	for key, value := range aboutInfoFSMVersions {
		fsm[key] = value
	}
	wam := make(map[int]string, len(aboutInfoWAMVersions))
	for key, value := range aboutInfoWAMVersions {
		wam[key] = value
	}
	return aboutInfoLastGlobal, aboutInfoHasGlobal, fsm, wam
}

func buildAboutSubSystems(count int) []AboutSubSystem {
	if count < 1 {
		count = 1
	}
	if count > cTCPMaxSubsysNum {
		count = cTCPMaxSubsysNum
	}
	items := make([]AboutSubSystem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, AboutSubSystem{Index: i, Name: fmt.Sprintf("%d Subsystem", i+1)})
	}
	return items
}

func readLocalControlVersion() (string, string) {
	paths := []string{"control"}
	if executable, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(executable), "control"))
	}
	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		version := ""
		compileTime := ""
		for _, line := range strings.Split(string(bytes), "\n") {
			text := strings.TrimSpace(line)
			if strings.HasPrefix(text, "Version") {
				version = afterColon(text)
			} else if strings.HasPrefix(text, "CompileTime") {
				compileTime = afterColon(text)
			}
		}
		return version, compileTime
	}
	return "", ""
}

func afterColon(text string) string {
	parts := strings.SplitN(text, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func resolveAboutVersionConfig(keys []string, fallback string) string {
	for _, key := range keys {
		value, err := database.GetConfigValuePreferNonEmpty(key)
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func preferAboutVersion(primary string, fallback string) string {
	text := strings.TrimSpace(strings.Trim(primary, "\x00"))
	if text != "" {
		return text
	}
	return strings.TrimSpace(strings.Trim(fallback, "\x00"))
}

func splitAboutControllerVersion(raw string, device string) (string, string) {
	text := strings.TrimRight(raw, "\x00\r\n ")
	if text == "" {
		return device, device
	}
	if len(text) <= 32 {
		return strings.ReplaceAll(text, "\x00", ""), device
	}
	hardware := strings.ReplaceAll(strings.TrimRight(text[:32], "\x00 "), "\x00", "")
	software := strings.ReplaceAll(strings.TrimRight(text[32:], "\x00 "), "\x00", "")
	parts := strings.Split(software, "_")
	displaySoftware := software
	if len(parts) > 0 && strings.HasPrefix(strings.ToUpper(parts[0]), "V") {
		displaySoftware = "RM200_" + device + "_" + software
	}
	displayHardware := "RM200_" + device + "_" + hardware
	if strings.HasPrefix(hardware, "RM200_") {
		displayHardware = hardware
	}
	return displaySoftware, displayHardware
}

func buildAboutHCVersion(rssVersion string, fsmSoftware string) string {
	version := strings.TrimPrefix(strings.TrimSpace(rssVersion), "V")
	if version == "" {
		version = "2.0.0"
	}
	parts := strings.Split(strings.TrimSpace(fsmSoftware), "_")
	if len(parts) >= 2 && !strings.HasPrefix(strings.ToUpper(parts[0]), "V") {
		return "RM200_" + parts[1] + "_Linux_FruSort_V" + version
	}
	return "FruitSort200_V" + version + "_48"
}

func formatIPMCompileDate(raw []uint8) string {
	text := numericPrefix(raw)
	if len(text) < 12 {
		return "0000/00/00 00:00:00"
	}
	return fmt.Sprintf("%s/%s/%s %s:%s:00", text[0:4], text[4:6], text[6:8], text[8:10], text[10:12])
}

func formatControllerCompileDate(raw []uint8) string {
	text := strings.TrimSpace(strings.TrimRight(string(raw), "\x00"))
	if len(text) >= 11 {
		month := aboutMonthToNum(strings.ToUpper(text[0:3]))
		day := strings.TrimSpace(text[4:6])
		year := strings.TrimSpace(text[7:11])
		if month != "" && day != "" && year != "" {
			if len(day) == 1 {
				day = "0" + day
			}
			return year + "/" + month + "/" + day
		}
	}
	return "0000/00/00"
}

func numericPrefix(raw []uint8) string {
	var builder strings.Builder
	for _, value := range raw {
		if value < '0' || value > '9' {
			break
		}
		builder.WriteByte(value)
	}
	return builder.String()
}

func aboutMonthToNum(month string) string {
	months := []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}
	for index, item := range months {
		if item == month {
			return fmt.Sprintf("%02d", index+1)
		}
	}
	return ""
}

func aboutAPILog(format string, args ...any) {
	fmt.Printf("%s Web API about %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}
