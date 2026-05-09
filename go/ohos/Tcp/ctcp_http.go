package tcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerCTCPHTTPRoutes(router *gin.Engine) {
	router.GET("/tcp/config", handleCTCPConfig)
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
