package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// WebSocketClient WebSocket 客户端封装
type WebSocketClient struct {
	secret    string
	dialer    *websocket.Dialer
	timeout   time.Duration
}

// NewWebSocketClient 创建新的 WebSocket 客户端
func NewWebSocketClient(secret string, timeout time.Duration) *WebSocketClient {
	return &WebSocketClient{
		secret:  secret,
		timeout: timeout,
		dialer: &websocket.Dialer{
			HandshakeTimeout: timeout,
		},
	}
}

// buildWebSocketURL 构建 WebSocket URL
func buildWebSocketURL(baseURL, endpoint string) (string, error) {
	// 将 http/https 转换为 ws/wss
	if len(baseURL) >= 4 {
		if baseURL[:4] == "http" {
			if baseURL[:5] == "https" {
				baseURL = "wss" + baseURL[5:]
			} else {
				baseURL = "ws" + baseURL[4:]
			}
		}
	}

	return baseURL + endpoint, nil
}

// connectWebSocket 建立 WebSocket 连接
func (ws *WebSocketClient) connectWebSocket(ctx context.Context, baseURL, endpoint string) (*websocket.Conn, error) {
	// 构建 WebSocket URL
	wsURL, err := buildWebSocketURL(baseURL, endpoint)
	if err != nil {
		return nil, NewConnectionError(err)
	}

	// 准备请求头
	header := http.Header{}
	if ws.secret != "" {
		header.Set("Authorization", "Bearer "+ws.secret)
	}

	// 建立 WebSocket 连接
	conn, _, err := ws.dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, NewConnectionError(err)
	}

	return conn, nil
}

// LogStream 日志流处理器
type LogStream struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	done     chan struct{}
	messages chan *types.LogInfo
	err      error
}

// Messages 返回日志消息通道
func (ls *LogStream) Messages() <-chan *types.LogInfo {
	return ls.messages
}

// Err 返回错误信息
func (ls *LogStream) Err() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	return ls.err
}

// Close 关闭日志流
func (ls *LogStream) Close() error {
	close(ls.done)
	return ls.conn.Close()
}

// readMessages 读取日志消息
func (ls *LogStream) readMessages() {
	defer close(ls.messages)

	for {
		select {
		case <-ls.done:
			return
		default:
			_, message, err := ls.conn.ReadMessage()
			if err != nil {
				ls.mu.Lock()
				ls.err = NewAPIError(ErrAPIError, "读取日志流失败", err)
				ls.mu.Unlock()
				return
			}

			var logInfo types.LogInfo
			if err := json.Unmarshal(message, &logInfo); err != nil {
				// 如果解析失败，尝试作为纯文本处理
				logInfo = types.LogInfo{
					LogType: "info",
					Payload: string(message),
				}
			}

			select {
			case ls.messages <- &logInfo:
			case <-ls.done:
				return
			}
		}
	}
}

// StreamLogs 获取日志流（WebSocket）
func (c *Client) StreamLogs(ctx context.Context) (*LogStream, error) {
	// 创建 WebSocket 客户端
	wsClient := NewWebSocketClient(c.secret, c.timeout)

	// 建立 WebSocket 连接
	conn, err := wsClient.connectWebSocket(ctx, c.baseURL, "/logs")
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "建立日志流连接失败", err)
	}

	// 创建日志流处理器
	stream := &LogStream{
		conn:     conn,
		done:     make(chan struct{}),
		messages: make(chan *types.LogInfo, 100),
	}

	// 启动消息读取协程
	go stream.readMessages()

	return stream, nil
}

// TrafficStream 流量统计流处理器
type TrafficStream struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	done     chan struct{}
	messages chan *types.TrafficInfo
	err      error
}

// Messages 返回流量统计消息通道
func (ts *TrafficStream) Messages() <-chan *types.TrafficInfo {
	return ts.messages
}

// Err 返回错误信息
func (ts *TrafficStream) Err() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.err
}

// Close 关闭流量统计流
func (ts *TrafficStream) Close() error {
	close(ts.done)
	return ts.conn.Close()
}

// readMessages 读取流量统计消息
func (ts *TrafficStream) readMessages() {
	defer close(ts.messages)

	for {
		select {
		case <-ts.done:
			return
		default:
			_, message, err := ts.conn.ReadMessage()
			if err != nil {
				ts.mu.Lock()
				ts.err = NewAPIError(ErrAPIError, "读取流量统计流失败", err)
				ts.mu.Unlock()
				return
			}

			var trafficInfo types.TrafficInfo
			if err := json.Unmarshal(message, &trafficInfo); err != nil {
				ts.mu.Lock()
				ts.err = NewAPIError(ErrAPIError, "解析流量统计数据失败", err)
				ts.mu.Unlock()
				return
			}

			select {
			case ts.messages <- &trafficInfo:
			case <-ts.done:
				return
			}
		}
	}
}

// StreamTraffic 获取流量统计流（WebSocket）
func (c *Client) StreamTraffic(ctx context.Context) (*TrafficStream, error) {
	// 创建 WebSocket 客户端
	wsClient := NewWebSocketClient(c.secret, c.timeout)

	// 建立 WebSocket 连接
	conn, err := wsClient.connectWebSocket(ctx, c.baseURL, "/traffic")
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "建立流量统计流连接失败", err)
	}

	// 创建流量统计流处理器
	stream := &TrafficStream{
		conn:     conn,
		done:     make(chan struct{}),
		messages: make(chan *types.TrafficInfo, 100),
	}

	// 启动消息读取协程
	go stream.readMessages()

	return stream, nil
}

// MemoryStream 内存使用流处理器
type MemoryStream struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	done     chan struct{}
	messages chan *types.MemoryInfo
	err      error
}

// Messages 返回内存使用消息通道
func (ms *MemoryStream) Messages() <-chan *types.MemoryInfo {
	return ms.messages
}

// Err 返回错误信息
func (ms *MemoryStream) Err() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.err
}

// Close 关闭内存使用流
func (ms *MemoryStream) Close() error {
	close(ms.done)
	return ms.conn.Close()
}

// readMessages 读取内存使用消息
func (ms *MemoryStream) readMessages() {
	defer close(ms.messages)

	for {
		select {
		case <-ms.done:
			return
		default:
			_, message, err := ms.conn.ReadMessage()
			if err != nil {
				ms.mu.Lock()
				ms.err = NewAPIError(ErrAPIError, "读取内存使用流失败", err)
				ms.mu.Unlock()
				return
			}

			var memoryInfo types.MemoryInfo
			if err := json.Unmarshal(message, &memoryInfo); err != nil {
				ms.mu.Lock()
				ms.err = NewAPIError(ErrAPIError, "解析内存使用数据失败", err)
				ms.mu.Unlock()
				return
			}

			select {
			case ms.messages <- &memoryInfo:
			case <-ms.done:
				return
			}
		}
	}
}

// StreamMemory 获取内存使用流（WebSocket）
func (c *Client) StreamMemory(ctx context.Context) (*MemoryStream, error) {
	// 创建 WebSocket 客户端
	wsClient := NewWebSocketClient(c.secret, c.timeout)

	// 建立 WebSocket 连接
	conn, err := wsClient.connectWebSocket(ctx, c.baseURL, "/memory")
	if err != nil {
		return nil, NewAPIError(ErrAPIError, "建立内存使用流连接失败", err)
	}

	// 创建内存使用流处理器
	stream := &MemoryStream{
		conn:     conn,
		done:     make(chan struct{}),
		messages: make(chan *types.MemoryInfo, 100),
	}

	// 启动消息读取协程
	go stream.readMessages()

	return stream, nil
}

// ReadAllLogs 读取所有日志直到流结束或出错
func ReadAllLogs(stream *LogStream) ([]*types.LogInfo, error) {
	var logs []*types.LogInfo
	for log := range stream.Messages() {
		logs = append(logs, log)
	}
	return logs, stream.Err()
}

// ReadAllTraffic 读取所有流量统计直到流结束或出错
func ReadAllTraffic(stream *TrafficStream) ([]*types.TrafficInfo, error) {
	var traffic []*types.TrafficInfo
	for t := range stream.Messages() {
		traffic = append(traffic, t)
	}
	return traffic, stream.Err()
}

// ReadAllMemory 读取所有内存使用直到流结束或出错
func ReadAllMemory(stream *MemoryStream) ([]*types.MemoryInfo, error) {
	var memory []*types.MemoryInfo
	for m := range stream.Messages() {
		memory = append(memory, m)
	}
	return memory, stream.Err()
}

// StreamLogsToWriter 将日志流写入到 Writer
func StreamLogsToWriter(stream *LogStream, writer io.Writer) error {
	for log := range stream.Messages() {
		line := fmt.Sprintf("[%s] %s\n", log.LogType, log.Payload)
		if _, err := writer.Write([]byte(line)); err != nil {
			return err
		}
	}
	return stream.Err()
}
