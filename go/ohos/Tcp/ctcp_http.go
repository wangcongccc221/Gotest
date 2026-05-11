package tcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerCTCPHTTPRoutes(router *gin.Engine) {
	router.GET("/tcp/config", handleCTCPConfig)
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
