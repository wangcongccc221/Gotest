package tcp

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var webSocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func registerWebSocketRoutes(router *gin.Engine) {
	router.GET("/ws", handleWebSocketEcho)
}

func handleWebSocketEcho(ctx *gin.Context) {
	conn, err := webSocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType == websocket.TextMessage {
			payload = append([]byte("已发送"), payload...)
		}
		if err := conn.WriteMessage(messageType, payload); err != nil {
			return
		}
	}
}
