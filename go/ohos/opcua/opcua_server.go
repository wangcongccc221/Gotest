package opcua

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	opcuaserver "github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

const (
	opcuaServerHost = "0.0.0.0"
	opcuaServerPort = 4840
)

var (
	opcuaServerMu      sync.Mutex
	activeOPCUAServer  *opcuaserver.Server
	opcuaServerCancel  context.CancelFunc
	opcuaServerRunning bool
)

func StartOPCUAServer() int {
	return startOPCUAServer()
}

func StopOPCUAServer() int {
	return stopOPCUAServer()
}

func startOPCUAServer() int {
	opcuaServerMu.Lock()
	defer opcuaServerMu.Unlock()

	if opcuaServerRunning {
		return opcuaServerPort
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv := newOPCUAServer()
	if err := srv.Start(ctx); err != nil {
		cancel()
		return -1
	}

	activeOPCUAServer = srv
	opcuaServerCancel = cancel
	opcuaServerRunning = true
	return opcuaServerPort
}

func stopOPCUAServer() int {
	opcuaServerMu.Lock()
	srv := activeOPCUAServer
	cancel := opcuaServerCancel
	activeOPCUAServer = nil
	opcuaServerCancel = nil
	opcuaServerRunning = false
	opcuaServerMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if srv == nil {
		return 0
	}
	if err := srv.Close(); err != nil {
		return -1
	}
	return 0
}

func opcuaServerStatus() map[string]any {
	opcuaServerMu.Lock()
	defer opcuaServerMu.Unlock()

	return map[string]any{
		"endpoint": fmt.Sprintf("opc.tcp://%s:%d", opcuaServerHost, opcuaServerPort),
		"port":     opcuaServerPort,
		"running":  opcuaServerRunning,
	}
}

func newOPCUAServer() *opcuaserver.Server {
	opts := []opcuaserver.Option{
		opcuaserver.EnableSecurity("None", ua.MessageSecurityModeNone),
		opcuaserver.EnableAuthMode(ua.UserTokenTypeAnonymous),
		opcuaserver.EndPoint(opcuaServerHost, opcuaServerPort),
		opcuaserver.EndPoint("localhost", opcuaServerPort),
		opcuaserver.ServerName("HarmonyOS Go OPC UA Server"),
		opcuaserver.ProductName("HarmonyOS Go OPC UA POC"),
		opcuaserver.ManufacturerName("goTest"),
	}

	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		opts = append(opts, opcuaserver.EndPoint(hostname, opcuaServerPort))
	}

	srv := opcuaserver.New(opts...)
	addDemoOPCUANamespace(srv)
	return srv
}

func addDemoOPCUANamespace(srv *opcuaserver.Server) {
	ns := opcuaserver.NewMapNamespace(srv, "HarmonyGo")
	ns.Data["Tag1"] = float64(123.45)
	ns.Data["Tag2"] = int32(42)
	ns.Data["Message"] = "Hello OPC UA from HarmonyOS Go"
	ns.Data["Running"] = true
	ns.Data["Timestamp"] = time.Now()

	rootNS, err := srv.Namespace(0)
	if err != nil {
		return
	}
	rootObjects := rootNS.Objects()
	if rootObjects == nil {
		return
	}
	rootObjects.AddRef(ns.Objects(), id.HasComponent, true)
}
