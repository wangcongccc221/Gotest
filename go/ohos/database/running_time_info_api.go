package database

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SortLogQueryRequest struct {
	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`
	StartTime string `json:"StartTime"`
	EndTime   string `json:"EndTime"`
}

type SortRunningStateRequest struct {
	SortSpeedPerMinute float64 `json:"SortSpeedPerMinute"`
	IsEndProcess       bool    `json:"IsEndProcess"`
}

type SortRunningStateResult struct {
	Action         string `json:"Action"`
	OpenRecordID   int    `json:"OpenRecordID"`
	ClosedRecordID int    `json:"ClosedRecordID"`
}

type SortLogSegment struct {
	ID          int    `json:"ID"`
	StartTime   string `json:"StartTime"`
	StopTime    string `json:"StopTime"`
	StartMinute int    `json:"StartMinute"`
	StopMinute  int    `json:"StopMinute"`
	Minutes     int    `json:"Minutes"`
}

type SortLogDayItem struct {
	RunningDate  string           `json:"RunningDate"`
	TotalMinutes int              `json:"TotalMinutes"`
	Segments     []SortLogSegment `json:"Segments"`
}

type SortLogQueryResult struct {
	Items        []SortLogDayItem `json:"Items"`
	TotalMinutes int              `json:"TotalMinutes"`
}

func registerRunningTimeInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/FruitInfo")
	group.POST("/GetSortLog", handleSortLogGet)
	group.POST("/ReportSortRunningState", handleSortRunningStateReport)
}

func handleSortLogGet(ctx *gin.Context) {
	var request SortLogQueryRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	result, err := QuerySortLog(request)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func QuerySortLog(request SortLogQueryRequest) (SortLogQueryResult, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return SortLogQueryResult{}, err
	}

	startDate := normalizeSortLogDate(firstNonEmpty(request.StartDate, request.StartTime))
	endDate := normalizeSortLogDate(firstNonEmpty(request.EndDate, request.EndTime))
	query := db.Model(&TbRunningTimeInfo{})
	if startDate != "" {
		query = query.Where("RunningDate >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("RunningDate <= ?", endDate)
	}

	rows := make([]TbRunningTimeInfo, 0)
	if err := query.Order("RunningDate asc, StartTime asc, ID asc").Find(&rows).Error; err != nil {
		return SortLogQueryResult{}, err
	}
	return buildSortLogResult(rows), nil
}

func handleSortRunningStateReport(ctx *gin.Context) {
	var request SortRunningStateRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}
	result, err := ReportSortRunningState(request)
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}
	fruitInfoAPIOK(ctx, result)
}

func ReportSortRunningState(request SortRunningStateRequest) (SortRunningStateResult, error) {
	return ReportSortRunningStateAt(request, databaseNow())
}

func ReportSortRunningStateAt(request SortRunningStateRequest, now time.Time) (SortRunningStateResult, error) {
	db, err := getInitializedFileORMDB()
	if err != nil {
		return SortRunningStateResult{}, err
	}
	now = databaseLocalTime(now)
	runningDate := now.Format("2006-01-02")
	currentTime := now.Format("15:04:05")
	isRunning := request.SortSpeedPerMinute > 0 && !request.IsEndProcess

	var result SortRunningStateResult
	err = db.Transaction(func(tx *gorm.DB) error {
		openRow, ok, err := findOpenRunningTimeRow(tx)
		if err != nil {
			return err
		}

		if ok && strings.TrimSpace(openRow.RunningDate) != runningDate {
			if err := closeRunningTimeRow(tx, openRow.ID, "23:59:59"); err != nil {
				return err
			}
			result.ClosedRecordID = openRow.ID
			ok = false
			if isRunning {
				id, err := createRunningTimeRow(tx, runningDate, currentTime)
				if err != nil {
					return err
				}
				result.OpenRecordID = id
				result.Action = "closedAndOpened"
				return nil
			}
			result.Action = "closed"
			return nil
		}

		if ok {
			if isRunning {
				result.OpenRecordID = openRow.ID
				result.Action = "keptOpen"
				return nil
			}
			if err := closeRunningTimeRow(tx, openRow.ID, currentTime); err != nil {
				return err
			}
			result.ClosedRecordID = openRow.ID
			result.Action = "closed"
			return nil
		}

		if isRunning {
			id, err := createRunningTimeRow(tx, runningDate, currentTime)
			if err != nil {
				return err
			}
			result.OpenRecordID = id
			result.Action = "opened"
			return nil
		}

		result.Action = "idle"
		return nil
	})
	return result, err
}

func findOpenRunningTimeRow(tx *gorm.DB) (TbRunningTimeInfo, bool, error) {
	var row TbRunningTimeInfo
	err := tx.Where("StopTime = ?", "").Order("ID desc").First(&row).Error
	if err == nil {
		return row, true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TbRunningTimeInfo{}, false, nil
	}
	return TbRunningTimeInfo{}, false, err
}

func createRunningTimeRow(tx *gorm.DB, runningDate string, startTime string) (int, error) {
	row := TbRunningTimeInfo{
		RunningDate: runningDate,
		StartTime:   startTime,
		StopTime:    "",
	}
	if err := tx.Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func closeRunningTimeRow(tx *gorm.DB, id int, stopTime string) error {
	return tx.Model(&TbRunningTimeInfo{}).Where("ID = ?", id).Update("StopTime", stopTime).Error
}

func buildSortLogResult(rows []TbRunningTimeInfo) SortLogQueryResult {
	items := make([]SortLogDayItem, 0)
	indexByDate := make(map[string]int)
	totalMinutes := 0

	for _, row := range rows {
		runningDate := strings.TrimSpace(row.RunningDate)
		if runningDate == "" {
			continue
		}
		index, ok := indexByDate[runningDate]
		if !ok {
			index = len(items)
			indexByDate[runningDate] = index
			items = append(items, SortLogDayItem{
				RunningDate: runningDate,
				Segments:    make([]SortLogSegment, 0),
			})
		}

		segment, ok := buildSortLogSegment(row)
		if !ok {
			continue
		}
		items[index].Segments = append(items[index].Segments, segment)
		items[index].TotalMinutes += segment.Minutes
		totalMinutes += segment.Minutes
	}

	return SortLogQueryResult{
		Items:        items,
		TotalMinutes: totalMinutes,
	}
}

func buildSortLogSegment(row TbRunningTimeInfo) (SortLogSegment, bool) {
	startMinute, ok := sortLogMinuteOfDay(row.StartTime)
	if !ok {
		return SortLogSegment{}, false
	}
	stopMinute, ok := sortLogMinuteOfDay(row.StopTime)
	if !ok || stopMinute <= startMinute {
		return SortLogSegment{}, false
	}
	return SortLogSegment{
		ID:          row.ID,
		StartTime:   strings.TrimSpace(row.StartTime),
		StopTime:    strings.TrimSpace(row.StopTime),
		StartMinute: startMinute,
		StopMinute:  stopMinute,
		Minutes:     stopMinute - startMinute,
	}, true
}

func sortLogMinuteOfDay(value string) (int, bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, false
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err := time.ParseInLocation(layout, text, databaseChinaLocation)
		if err == nil {
			return parsed.Hour()*60 + parsed.Minute(), true
		}
	}
	return 0, false
}

func normalizeSortLogDate(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006-01-02",
		"2006/01/02",
	} {
		parsed, err := time.ParseInLocation(layout, text, databaseChinaLocation)
		if err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
