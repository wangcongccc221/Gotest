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

func registerWebSocketRoutes(router *gin.Engine) { //注册websocket 路由
	router.GET("/ws", handleWebSocketEcho)
}

func handleWebSocketEcho(ctx *gin.Context) { //处理回显
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

func handleWebSocketData(ctx *gin.Context)
{
	conn, err := webSocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)	// 升级为websocket连接
	if err != nil {
		return
	}
	//每秒给这个客户端发送一次这个ststatics字符串数据  数据转换之后 发送
	
}
