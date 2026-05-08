package opcua

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

const opcuaRequestTimeout = 5 * time.Second

type opcuaEndpointInfo struct {
	EndpointURL       string `json:"endpointUrl"`
	SecurityMode      string `json:"securityMode"`
	SecurityPolicyURI string `json:"securityPolicyUri"`
	SecurityLevel     uint8  `json:"securityLevel"`
	ServerName        string `json:"serverName,omitempty"`
}

type opcuaReadResult struct {
	Endpoint string `json:"endpoint"`
	NodeID   string `json:"nodeId"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}

func registerOPCUARoutes(router *gin.Engine) { //注册OPC路由
	router.GET("/opcua", handleOPCUAStatus)
	router.GET("/opcua/server", handleOPCUAServerStatus)
	router.POST("/opcua/server/start", handleOPCUAServerStart)
	router.POST("/opcua/server/stop", handleOPCUAServerStop)
	router.GET("/opcua/endpoints", handleOPCUAEndpoints)
	router.GET("/opcua/read", handleOPCUARead)
}

func handleOPCUAStatus(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"package": "github.com/gopcua/opcua",
		"status":  "ready",
	})
}

func handleOPCUAServerStatus(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, opcuaServerStatus()) //返回给前端
}

func handleOPCUAServerStart(ctx *gin.Context) {
	port := startOPCUAServer()
	if port < 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "OPC UA server failed to start"})
		return
	}
	ctx.JSON(http.StatusOK, opcuaServerStatus())
}

func handleOPCUAServerStop(ctx *gin.Context) {
	if code := stopOPCUAServer(); code != 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "OPC UA server failed to stop"})
		return
	}
	ctx.JSON(http.StatusOK, opcuaServerStatus())
}

func handleOPCUAEndpoints(ctx *gin.Context) {
	endpoint := strings.TrimSpace(ctx.Query("endpoint"))
	if endpoint == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing endpoint"})
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), opcuaRequestTimeout)
	defer cancel()

	endpoints, err := opcuaEndpointDescriptions(reqCtx, endpoint)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"endpoints": endpoints})
}

func handleOPCUARead(ctx *gin.Context) {
	endpoint := strings.TrimSpace(ctx.Query("endpoint"))
	nodeID := strings.TrimSpace(ctx.Query("node"))
	if endpoint == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing endpoint"})
		return
	}
	if nodeID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing node"})
		return
	}

	if _, err := ua.ParseNodeID(nodeID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid node id: %v", err)})
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx.Request.Context(), opcuaRequestTimeout)
	defer cancel()

	result, err := opcuaReadNodeValue(reqCtx, endpoint, nodeID)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func opcuaEndpointDescriptions(ctx context.Context, endpoint string) ([]opcuaEndpointInfo, error) {
	endpoints, err := opcua.GetEndpoints(ctx, endpoint, opcuaClientOptions()...)
	if err != nil {
		return nil, err
	}

	result := make([]opcuaEndpointInfo, 0, len(endpoints))
	for _, ep := range endpoints {
		info := opcuaEndpointInfo{
			EndpointURL:       ep.EndpointURL,
			SecurityMode:      ep.SecurityMode.String(),
			SecurityPolicyURI: ep.SecurityPolicyURI,
			SecurityLevel:     ep.SecurityLevel,
		}
		if ep.Server != nil && ep.Server.ApplicationName != nil {
			info.ServerName = ep.Server.ApplicationName.Text
		}
		result = append(result, info)
	}
	return result, nil
}

func opcuaReadNodeValue(ctx context.Context, endpoint string, nodeIDText string) (*opcuaReadResult, error) {
	nodeID, err := ua.ParseNodeID(nodeIDText)
	if err != nil {
		return nil, err
	}

	client, err := opcua.NewClient(endpoint, opcuaClientOptions()...)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	defer client.Close(context.Background())

	value, err := client.Node(nodeID).Value(ctx)
	if err != nil {
		return nil, err
	}

	return &opcuaReadResult{
		Endpoint: endpoint,
		NodeID:   nodeID.String(),
		Type:     value.Type().String(),
		Value:    fmt.Sprint(value.Value()),
	}, nil
}

func opcuaClientOptions() []opcua.Option {
	return []opcua.Option{
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.SecurityPolicy(ua.SecurityPolicyURINone),
		opcua.AuthAnonymous(),
		opcua.DialTimeout(opcuaRequestTimeout),
		opcua.RequestTimeout(opcuaRequestTimeout),
	}
}
