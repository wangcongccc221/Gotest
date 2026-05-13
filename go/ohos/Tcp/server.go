package tcp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	serverHost = "127.0.0.1"
	serverPort = 18080
	serverAddr = "127.0.0.1:18080"
)

var (
	serverMu      sync.Mutex
	activeServer  *http.Server
	serverRunning bool
)

func StartServer() int {
	return startServer()
}

func StopServer() int {
	return stopServer()
}

func newRouter() http.Handler {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"framework": "gin",
			"message":   "pong from Go",
		})
	})
	return router
}

func startServer() int {
	serverMu.Lock()
	defer serverMu.Unlock()

	if serverRunning {
		return serverPort
	}

	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return -1
	}

	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           newRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	activeServer = srv
	serverRunning = true

	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverMu.Lock()
			if activeServer == srv {
				activeServer = nil
				serverRunning = false
			}
			serverMu.Unlock()
		}
	}()

	return serverPort
}

func stopServer() int {
	serverMu.Lock()
	srv := activeServer
	activeServer = nil
	serverRunning = false
	serverMu.Unlock()

	if srv == nil {
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return -1
	}
	return 0
}
