package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newRuleTestServer 创建用于规则测试的 mock 服务器
func newRuleTestServer(t *testing.T, wantMethod string, wantPayload map[int]bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径：核心仅提供 /rules/disable 端点
		if r.URL.Path != "/rules/disable" {
			t.Errorf("Expected path /rules/disable, got %s", r.URL.Path)
		}

		// 验证请求方法
		if r.Method != wantMethod {
			t.Errorf("Expected method %s, got %s", wantMethod, r.Method)
		}

		// 验证请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}
		var payload map[int]bool
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Failed to decode request body %q: %v", string(body), err)
		}
		for k, v := range wantPayload {
			got, ok := payload[k]
			if !ok {
				t.Errorf("Expected index %d in payload, got %v", k, payload)
				continue
			}
			if got != v {
				t.Errorf("Expected index %d disabled=%v, got %v", k, v, got)
			}
		}
		if len(payload) != len(wantPayload) {
			t.Errorf("Expected payload length %d, got %d (%v)", len(wantPayload), len(payload), payload)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestDisableRules(t *testing.T) {
	server := newRuleTestServer(t, "PATCH", map[int]bool{0: true, 1: true, 5: true})
	defer server.Close()

	client := NewClient(server.URL, "test-secret")

	err := client.DisableRules(context.Background(), []int{0, 1, 5})
	if err != nil {
		t.Fatalf("DisableRules failed: %v", err)
	}
}

func TestEnableRules(t *testing.T) {
	// 核心无 /rules/enable 端点，启用通过 /rules/disable 的 value=false 实现
	server := newRuleTestServer(t, "PATCH", map[int]bool{0: false, 2: false})
	defer server.Close()

	client := NewClient(server.URL, "test-secret")

	err := client.EnableRules(context.Background(), []int{0, 2})
	if err != nil {
		t.Fatalf("EnableRules failed: %v", err)
	}
}

func TestDisableRulesEmptyList(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-secret")

	err := client.DisableRules(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for empty rule list, got nil")
	}
}

func TestEnableRulesEmptyList(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-secret")

	err := client.EnableRules(context.Background(), []int{})
	if err == nil {
		t.Fatal("Expected error for empty rule list, got nil")
	}
}
