package database

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type fruitProcessInfoTimeRangeRequest struct {
	StartTime string `json:"StartTime"`
	EndTime   string `json:"EndTime"`
	Year      *int   `json:"Year"`
}

func registerFruitProcessInfoRoutes(router *gin.Engine) {
	group := router.Group("/Api/FruitProcessInfo")
	group.POST("/GetByTimeRange", handleFruitProcessInfoGetByTimeRange)
}

func handleFruitProcessInfoGetByTimeRange(ctx *gin.Context) {
	var request fruitProcessInfoTimeRangeRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		fruitInfoAPIFail(ctx, "invalid request: "+err.Error())
		return
	}

	startTime := strings.TrimSpace(request.StartTime)
	endTime := strings.TrimSpace(request.EndTime)
	if startTime == "" || endTime == "" {
		fruitInfoAPIFail(ctx, "StartTime and EndTime are required")
		return
	}
	queryStartTime := fruitProcessInfoQueryTime(startTime)
	queryEndTime := fruitProcessInfoQueryTime(endTime)

	years := fruitProcessInfoRequestYears(request.Year, startTime, endTime)
	if len(years) == 0 {
		fruitInfoAPIFail(ctx, "invalid year")
		return
	}

	db, err := getInitializedFileORMDB()
	if err != nil {
		fruitInfoAPIFail(ctx, err.Error())
		return
	}

	rows := make([]TbFruitProcessInfo, 0)
	for _, year := range years {
		tableName := fruitProcessInfoTableName(year)
		if !db.Migrator().HasTable(tableName) {
			continue
		}

		var yearRows []TbFruitProcessInfo
		if err := db.Table(tableName).
			Where("RunningDate >= ? AND RunningDate <= ?", queryStartTime, queryEndTime).
			Order("RunningDate asc, FID asc").
			Find(&yearRows).Error; err != nil {
			fruitInfoAPIFail(ctx, err.Error())
			return
		}
		rows = append(rows, yearRows...)
	}
	fruitInfoAPIOK(ctx, rows)
}

func fruitProcessInfoRequestYears(value *int, startTime string, endTime string) []int {
	if value != nil && fruitProcessInfoValidYear(*value) {
		return []int{*value}
	}

	startYear := fruitProcessInfoRequestYear(startTime)
	if !fruitProcessInfoValidYear(startYear) {
		return nil
	}
	endYear := fruitProcessInfoRequestYear(endTime)
	if !fruitProcessInfoValidYear(endYear) || endYear < startYear {
		endYear = startYear
	}

	years := make([]int, 0, endYear-startYear+1)
	for year := startYear; year <= endYear; year++ {
		years = append(years, year)
	}
	return years
}

func fruitProcessInfoRequestYear(value string) int {
	parsed, ok := fruitProcessInfoParseTime(value)
	if !ok {
		return 0
	}
	return parsed.Year()
}

func fruitProcessInfoQueryTime(value string) string {
	parsed, ok := fruitProcessInfoParseTime(value)
	if !ok {
		return strings.TrimSpace(value)
	}
	return parsed.Format("2006-01-02 15:04")
}

func fruitProcessInfoParseTime(value string) (time.Time, bool) {
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
		parsed, err := time.ParseInLocation(layout, strings.TrimSpace(value), databaseChinaLocation)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func fruitProcessInfoValidYear(year int) bool {
	return year >= 2000 && year <= 9999
}
