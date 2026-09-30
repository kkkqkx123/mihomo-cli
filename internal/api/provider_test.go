package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

func TestListRuleProviders(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if r.URL.Path != "/providers/rules" {
			t.Errorf("Expected path /providers/rules, got %s", r.URL.Path)
		}

		// 验证请求方法
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// 返回模拟数据（使用正确的 API 格式）
		response := types.RuleProvidersResponse{
			Providers: map[string]*types.RuleProviderInfo{
				"cn_domain": {
					Name:        "cn_domain",
					Type:        "RuleSet",
					Behavior:    "domain",
					Format:      "mrs",
					VehicleType: "HTTP",
					RuleCount:   1000,
					UpdatedAt:   "2024-01-01T00:00:00.000Z",
				},
				"github_domain": {
					Name:        "github_domain",
					Type:        "RuleSet",
					Behavior:    "domain",
					Format:      "mrs",
					VehicleType: "HTTP",
					RuleCount:   50,
					UpdatedAt:   "2024-01-01T00:00:00.000Z",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(server.URL, "test-secret")

	// 测试 ListRuleProviders
	providers, err := client.ListRuleProviders(context.Background())
	if err != nil {
		t.Fatalf("ListRuleProviders failed: %v", err)
	}

	// 验证结果
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// 验证 cn_domain 提供者
	provider, exists := providers["cn_domain"]
	if !exists {
		t.Fatal("Expected provider 'cn_domain' not found")
	}

	if provider.Name != "cn_domain" {
		t.Errorf("Expected provider name 'cn_domain', got %s", provider.Name)
	}

	if provider.Type != "RuleSet" {
		t.Errorf("Expected provider type 'RuleSet', got %s", provider.Type)
	}

	if provider.Behavior != "domain" {
		t.Errorf("Expected behavior 'domain', got %s", provider.Behavior)
	}

	if provider.Format != "mrs" {
		t.Errorf("Expected format 'mrs', got %s", provider.Format)
	}

	if provider.VehicleType != "HTTP" {
		t.Errorf("Expected vehicle type 'HTTP', got %s", provider.VehicleType)
	}

	if provider.RuleCount != 1000 {
		t.Errorf("Expected rule count 1000, got %d", provider.RuleCount)
	}

	// 验证 github_domain 提供者
	provider2, exists := providers["github_domain"]
	if !exists {
		t.Fatal("Expected provider 'github_domain' not found")
	}

	if provider2.RuleCount != 50 {
		t.Errorf("Expected rule count 50, got %d", provider2.RuleCount)
	}
}

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
					Proxies: []*types.ProxyInfo{
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

func TestListProviderProxyMap(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/providers/proxies" {
			t.Errorf("Expected path /providers/proxies, got %s", r.URL.Path)
		}

		response := types.ProvidersResponse{
			Providers: map[string]*types.ProviderInfo{
				"subscription": {
					Name:        "subscription",
					Type:        "http",
					VehicleType: "HTTP",
					Proxies: []*types.ProxyInfo{
						{
							Name:         "node1",
							Type:         "ss",
							ProviderName: "subscription",
							Alive:        true,
							History: []types.DelayHistory{
								{Time: "2024-01-01T00:00:00Z", Delay: 100},
								{Time: "2024-01-01T00:01:00Z", Delay: 245},
							},
						},
						{
							Name:  "node2",
							Type:  "vmess",
							Alive: false,
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-secret")

	index, err := client.ListProviderProxyMap(context.Background())
	if err != nil {
		t.Fatalf("ListProviderProxyMap failed: %v", err)
	}

	if len(index) != 2 {
		t.Fatalf("Expected 2 indexed proxies, got %d", len(index))
	}

	node, exists := index["node1"]
	if !exists {
		t.Fatal("Expected node 'node1' in index")
	}
	if node.ProviderName != "subscription" {
		t.Errorf("Expected provider name 'subscription', got %s", node.ProviderName)
	}
	if node.Delay != 245 {
		t.Errorf("Expected delay backfilled from last history entry 245, got %d", node.Delay)
	}
	if !node.Alive {
		t.Error("Expected node1 to be alive")
	}

	node2, exists := index["node2"]
	if !exists {
		t.Fatal("Expected node 'node2' in index")
	}
	if node2.Delay != 0 {
		t.Errorf("Expected delay 0 for node without history, got %d", node2.Delay)
	}
}
