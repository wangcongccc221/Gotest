package tcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerCTCPHTTPRoutes(router *gin.Engine) {
	router.GET("/tcp/config", handleCTCPConfig)
	router.GET("/tcp/stglobal", handleCTCPStGlobal)
	router.GET("/tcp/statistics", handleCTCPStatistics)
}

func handleCTCPConfig(ctx *gin.Context) {
	snapshot, ok := LastCTCPConfigSnapshot()
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "CTCP config not received yet",
		})
		return
	}

	configJSON, err := formatCTCPConfigSnapshotJSON(snapshot)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(configJSON))
}

// 返回最近一次 0x1000 强转后序列化的完整 StGlobal JSON
func handleCTCPStGlobal(ctx *gin.Context) {
	body := LastCTCPStGlobalFullJSON()
	if body == "" {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "StGlobal not received yet; need FSM_CMD_CONFIG (0x1000) once",
		})
		return
	}
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(body))
}

func handleCTCPStatistics(ctx *gin.Context) {
	snapshot, ok := LastCTCPStatisticsSnapshot()
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "CTCP statistics not received yet",
		})
		return
	}

	statsJSON, err := formatCTCPStatisticsSnapshotJSON(snapshot)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.Data(http.StatusOK, "application/json; charset=utf-8", []byte(statsJSON))
}
