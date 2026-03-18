package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

func TestListProviders(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path != "/providers/proxies" {
			t.Errorf("Expected path /providers/proxies, got %s", r.URL.Path)
		}

		// 验证请求方法
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// 返回模拟数据
		response := types.ProvidersResponse{
			Providers: map[string]*types.ProviderInfo{
				"test-provider": {
					Name:        "test-provider",
					Type:        "file",
					VehicleType: "HTTP",
					Proxies: []types.ProviderProxyInfo{
						{
							Name:  "proxy1",
							Type:  "ss",
							UDP:   true,
							XUDP:  true,
							Alive: true,
						},
					},
					UpdatedAt: "2024-01-01T00:00:00.000Z",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL, "test-secret")

	// 测试 ListProviders
	providers, err := client.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}

	// 验证结果
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	provider, exists := providers["test-provider"]
	if !exists {
		t.Fatal("Expected provider 'test-provider' not found")
	}

	if provider.Name != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got %s", provider.Name)
	}

	if provider.Type != "file" {
		t.Errorf("Expected provider type 'file', got %s", provider.Type)
	}

	if provider.VehicleType != "HTTP" {
		t.Errorf("Expected vehicle type 'HTTP', got %s", provider.VehicleType)
	}

	if len(provider.Proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(provider.Proxies))
	}
}

func TestUpdateProvider(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		expectedPath := "/providers/proxies/test-provider"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// 验证请求方法
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		// 返回成功状态
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL, "test-secret")

	// 测试 UpdateProvider
	err := client.UpdateProvider(context.Background(), "test-provider")
	if err != nil {
		t.Fatalf("UpdateProvider failed: %v", err)
	}
}

func TestUpdateProviderWithSpecialChars(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径（应该被 URL 编码）
		expectedPath := "/providers/proxies/test%20provider"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// 验证请求方法
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		// 返回成功状态
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL, "test-secret")

	// 测试 UpdateProvider（带特殊字符的名称）
	err := client.UpdateProvider(context.Background(), "test provider")
	if err != nil {
		t.Fatalf("UpdateProvider failed: %v", err)
	}
}
