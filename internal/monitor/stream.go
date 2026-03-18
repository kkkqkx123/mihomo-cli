package monitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// StreamClient WebSocket 流客户端
type StreamClient struct {
	baseURL string
	secret  string
	conn    *websocket.Conn
	mu      sync.Mutex
}

// NewStreamClient 创建新的流客户端
func NewStreamClient(baseURL, secret string) *StreamClient {
	return &StreamClient{
		baseURL: baseURL,
		secret:  secret,
	}
}

// convertToWebSocketURL 将 HTTP URL 转换为 WebSocket URL
func (s *StreamClient) convertToWebSocketURL(endpoint string) (string, error) {
	// 解析基础 URL
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// 替换协议
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		// 如果已经是 ws 或 wss，保持不变
	}

	// 设置路径
	u.Path = endpoint

	return u.String(), nil
}

// StreamTraffic 流式获取流量数据
// 返回一个 channel，持续推送流量数据，直到 context 被取消或连接出错
func (s *StreamClient) StreamTraffic(ctx context.Context) (<-chan *types.TrafficInfo, error) {
	// 转换 URL
	wsURL, err := s.convertToWebSocketURL("/traffic")
	if err != nil {
		return nil, err
	}

	// 创建 WebSocket 配置
	config, err := websocket.NewConfig(wsURL, "http://localhost/")
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket config: %w", err)
	}

	// 设置请求头
	if s.secret != "" {
		config.Header = make(http.Header)
		config.Header.Set("Authorization", "Bearer "+s.secret)
	}

	// 建立 WebSocket 连接
	conn, err := websocket.DialConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// 创建数据 channel
	dataChan := make(chan *types.TrafficInfo, 100)

	// 启动 goroutine 读取数据
	go func() {
		defer close(dataChan)
		defer s.closeConnection()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 设置读取超时
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))

				var data types.TrafficInfo
				err := websocket.JSON.Receive(conn, &data)
				if err != nil {
					// 检查是否是超时
					if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
						continue
					}
					// 检查是否是连接关闭
					if strings.Contains(err.Error(), "use of closed network connection") ||
						strings.Contains(err.Error(), "EOF") {
						return
					}
					// 其他错误，继续尝试读取
					continue
				}

				select {
				case dataChan <- &data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return dataChan, nil
}

// StreamMemory 流式获取内存数据
// 返回一个 channel，持续推送内存数据，直到 context 被取消或连接出错
func (s *StreamClient) StreamMemory(ctx context.Context) (<-chan *types.MemoryInfo, error) {
	// 转换 URL
	wsURL, err := s.convertToWebSocketURL("/memory")
	if err != nil {
		return nil, err
	}

	// 创建 WebSocket 配置
	config, err := websocket.NewConfig(wsURL, "http://localhost/")
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket config: %w", err)
	}

	// 设置请求头
	if s.secret != "" {
		config.Header = make(http.Header)
		config.Header.Set("Authorization", "Bearer "+s.secret)
	}

	// 建立 WebSocket 连接
	conn, err := websocket.DialConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to websocket: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// 创建数据 channel
	dataChan := make(chan *types.MemoryInfo, 100)

	// 启动 goroutine 读取数据
	go func() {
		defer close(dataChan)
		defer s.closeConnection()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// 设置读取超时
				conn.SetReadDeadline(time.Now().Add(5 * time.Second))

				var data types.MemoryInfo
				err := websocket.JSON.Receive(conn, &data)
				if err != nil {
					// 检查是否是超时
					if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
						continue
					}
					// 检查是否是连接关闭
					if strings.Contains(err.Error(), "use of closed network connection") ||
						strings.Contains(err.Error(), "EOF") {
						return
					}
					// 其他错误，继续尝试读取
					continue
				}

				select {
				case dataChan <- &data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return dataChan, nil
}

// Close 关闭 WebSocket 连接
func (s *StreamClient) Close() error {
	return s.closeConnection()
}

// closeConnection 内部关闭连接方法
func (s *StreamClient) closeConnection() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// WatchTraffic 使用 HTTP 轮询方式监控流量（备用方案）
// 当 WebSocket 不可用时使用
func WatchTraffic(ctx context.Context, getTraffic func(ctx context.Context) (*types.TrafficInfo, error), interval time.Duration) <-chan *types.TrafficInfo {
	dataChan := make(chan *types.TrafficInfo, 10)

	go func() {
		defer close(dataChan)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := getTraffic(ctx)
				if err != nil {
					continue
				}
				select {
				case dataChan <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return dataChan
}

// WatchMemory 使用 HTTP 轮询方式监控内存（备用方案）
// 当 WebSocket 不可用时使用
func WatchMemory(ctx context.Context, getMemory func(ctx context.Context) (*types.MemoryInfo, error), interval time.Duration) <-chan *types.MemoryInfo {
	dataChan := make(chan *types.MemoryInfo, 10)

	go func() {
		defer close(dataChan)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := getMemory(ctx)
				if err != nil {
					continue
				}
				select {
				case dataChan <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return dataChan
}
