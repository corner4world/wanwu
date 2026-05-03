package mcp_util

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/ThinkInAIXYZ/go-mcp/client"
	"github.com/ThinkInAIXYZ/go-mcp/protocol"
	"github.com/ThinkInAIXYZ/go-mcp/transport"
	"github.com/UnicomAI/wanwu/pkg/constant"
	"github.com/UnicomAI/wanwu/pkg/log"
)

// httpClient 创建共享的 HTTP 客户端，跳过证书验证
var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

// ListTools 根据 transport 类型获取 MCP 工具列表
// transport: "sse" 或 "streamable"
func ListTools(ctx context.Context, url string, transportType string) ([]*protocol.Tool, error) {
	var transportClient transport.ClientTransport
	var err error
	if transportType == "" || url == "" {
		return nil, fmt.Errorf("transport type or url is empty")
	}
	switch transportType {
	case constant.MCPTransportStreamable:
		// 创建 StreamableHTTP 传输客户端
		transportClient, err = transport.NewStreamableHTTPClientTransport(url,
			transport.WithStreamableHTTPClientOptionLogger(log.Log()),
			transport.WithStreamableHTTPClientOptionHTTPClient(httpClient),
		)
		if err != nil {
			return nil, fmt.Errorf("mcp list tools (%v) init streamable transport err: %v", url, err)
		}
	case constant.MCPTransportSSE:
		// 默认使用 SSE 传输客户端
		transportClient, err = transport.NewSSEClientTransport(url,
			transport.WithSSEClientOptionReceiveTimeout(time.Minute*2),
			transport.WithSSEClientOptionLogger(log.Log()),
			transport.WithSSEClientOptionHTTPClient(httpClient),
		)
		if err != nil {
			return nil, fmt.Errorf("mcp list tools (%v) init sse transport err: %v", url, err)
		}
	default:
		return nil, fmt.Errorf("mcp list tools (%v) init transport err: %v", url, err)
	}

	// 初始化 MCP 客户端
	mcpClient, err := client.NewClient(transportClient)
	if err != nil {
		return nil, fmt.Errorf("mcp list tools (%v) init client err: %v", url, err)
	}
	defer func() { _ = mcpClient.Close() }()

	// 获取可用工具列表
	resp, err := mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("mcp list tools (%v) err: %v", url, err)
	}
	return resp.Tools, nil
}
