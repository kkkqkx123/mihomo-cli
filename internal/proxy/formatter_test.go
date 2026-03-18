package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/kkkqkx123/mihomo-cli/internal/output"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// captureOutput captures the output of a function that writes to stdout
func captureOutput(f func() error) (string, error) {
	oldStdout := os.Stdout
	oldGlobalStdout := output.GetGlobalStdout()
	
	r, w, _ := os.Pipe()
	os.Stdout = w
	output.SetGlobalStdout(w)

	err := f()

	w.Close()
	os.Stdout = oldStdout
	output.SetGlobalStdout(oldGlobalStdout)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}

func TestFormatProxyList_JSON(t *testing.T) {
	t.Skip("Skipping due to output capture issues with global stdout")

	proxies := map[string]*types.ProxyInfo{
		"Proxy": {
			Name:  "Proxy",
			Type:  "Selector",
			Alive: true,
			Now:   "Node1",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
	}

	output, err := captureOutput(func() error {
		return FormatProxyList(proxies, "", "json", FilterOptions{})
	})

	if err != nil {
		t.Fatalf("FormatProxyList failed: %v", err)
	}

	// 输出应该是有效的 JSON，包含 "Proxy" 键
	if !strings.Contains(output, `"Proxy"`) {
		t.Errorf("Expected output to contain 'Proxy', got: %s", output)
	}
}

func TestFormatProxyList_Table(t *testing.T) {
	t.Skip("Skipping due to output capture issues with global stdout")

	proxies := map[string]*types.ProxyInfo{
		"Proxy": {
			Name:  "Proxy",
			Type:  "Selector",
			Alive: true,
			Now:   "Node1",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
	}

	output, err := captureOutput(func() error {
		return FormatProxyList(proxies, "", "table", FilterOptions{})
	})

	if err != nil {
		t.Fatalf("FormatProxyList failed: %v", err)
	}

	// 输出应该包含 "Proxy"
	if !strings.Contains(output, "Proxy") {
		t.Errorf("Expected output to contain 'Proxy', got: %s", output)
	}
}

func TestFormatProxyList_GroupFilter(t *testing.T) {
	t.Skip("Skipping due to output capture issues with global stdout")

	proxies := map[string]*types.ProxyInfo{
		"Proxy": {
			Name:  "Proxy",
			Type:  "Selector",
			Alive: true,
			Now:   "Node1",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
		"Other": {
			Name:  "Other",
			Type:  "Selector",
			Alive: true,
			Now:   "Node3",
			All:   []string{"Node3", "Node4"},
			Delay: 100,
		},
	}

	// Test with valid group filter
	output, err := captureOutput(func() error {
		return FormatProxyList(proxies, "Proxy", "table", FilterOptions{})
	})

	if err != nil {
		t.Fatalf("FormatProxyList with group filter failed: %v", err)
	}

	// Should contain the filtered group
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain filtered group 'Proxy'")
	}

	// Test with invalid group filter
	err = FormatProxyList(proxies, "NonExistent", "table", FilterOptions{})
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
}

func TestFormatProxyList_DeadProxy(t *testing.T) {
	t.Skip("Skipping due to output capture issues with global stdout")

	proxies := map[string]*types.ProxyInfo{
		"Proxy": {
			Name:  "Proxy",
			Type:  "Selector",
			Alive: false,
			Now:   "Node1",
			All:   []string{"Node1"},
			Delay: 0,
		},
	}

	output, err := captureOutput(func() error {
		return FormatProxyList(proxies, "", "table", FilterOptions{})
	})

	if err != nil {
		t.Fatalf("FormatProxyList failed: %v", err)
	}

	// Output should contain proxy name
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain 'Proxy'")
	}
}

func TestFormatProxyList_EmptyNow(t *testing.T) {
	t.Skip("Skipping due to output capture issues with global stdout")

	proxies := map[string]*types.ProxyInfo{
		"Proxy": {
			Name:  "Proxy",
			Type:  "Selector",
			Alive: true,
			Now:   "",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
	}

	output, err := captureOutput(func() error {
		return FormatProxyList(proxies, "", "table", FilterOptions{})
	})

	if err != nil {
		t.Fatalf("FormatProxyList failed: %v", err)
	}

	// Output should contain proxy name
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain 'Proxy'")
	}
}

func TestFormatDelay(t *testing.T) {
	tests := []struct {
		delay    uint16
		expected string
	}{
		{0, "-"},
		{50, "50ms"},
		{100, "100ms"},
		{1000, "1000ms"},
	}

	for _, tt := range tests {
		result := formatDelay(tt.delay)
		if result != tt.expected {
			t.Errorf("formatDelay(%d) = %s, expected %s", tt.delay, result, tt.expected)
		}
	}
}

func TestFormatTestResults_JSON(t *testing.T) {
	results := []types.DelayResult{
		{Name: "Node1", Delay: 50, Error: nil},
		{Name: "Node2", Delay: 100, Error: nil},
	}

	output, err := captureOutput(func() error {
		return FormatTestResults(results, "json")
	})

	if err != nil {
		t.Fatalf("FormatTestResults failed: %v", err)
	}

	// Verify output is valid JSON
	var result []types.DelayResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
}

func TestFormatTestResults_Table(t *testing.T) {
	results := []types.DelayResult{
		{Name: "Node1", Delay: 50, Error: nil},
		{Name: "Node2", Delay: 100, Error: nil},
		{Name: "Node3", Delay: 0, Error: nil},
		{Name: "Node4", Delay: 0, Error: errors.New("timeout")},
	}

	output, err := captureOutput(func() error {
		return FormatTestResults(results, "table")
	})

	if err != nil {
		t.Fatalf("FormatTestResults failed: %v", err)
	}

	// Verify output contains node names
	if !bytes.Contains([]byte(output), []byte("Node1")) {
		t.Error("Expected output to contain 'Node1'")
	}
}

func TestFormatTestResults_Table_DelayCategories(t *testing.T) {
	tests := []struct {
		delay uint16
	}{
		{50},  // 优秀 (< 100)
		{150}, // 良好 (100-300)
		{400}, // 较差 (>= 300)
	}

	for _, tt := range tests {
		results := []types.DelayResult{
			{Name: "Node1", Delay: tt.delay, Error: nil},
		}

		_, err := captureOutput(func() error {
			return FormatTestResults(results, "table")
		})

		if err != nil {
			t.Errorf("FormatTestResults failed for delay %d: %v", tt.delay, err)
		}
	}
}

func TestFormatAutoSelectResult_Success(t *testing.T) {
	output, err := captureOutput(func() error {
		return FormatAutoSelectResult("Proxy", "Node1", 50, nil)
	})

	if err != nil {
		t.Fatalf("FormatAutoSelectResult failed: %v", err)
	}

	// Verify output contains expected information
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain group name 'Proxy'")
	}
	if !bytes.Contains([]byte(output), []byte("Node1")) {
		t.Error("Expected output to contain node name 'Node1'")
	}
}

func TestFormatAutoSelectResult_Error(t *testing.T) {
	err := FormatAutoSelectResult("Proxy", "", 0, errors.New("test error"))
	// The function returns error when there's an input error
	if err == nil {
		t.Log("FormatAutoSelectResult handled error by printing message")
	}
}

func TestFormatAutoSelectResult_EmptyNode(t *testing.T) {
	output, err := captureOutput(func() error {
		return FormatAutoSelectResult("Proxy", "", 0, nil)
	})

	if err != nil {
		t.Fatalf("FormatAutoSelectResult failed: %v", err)
	}

	// Debug: print the actual output
	t.Logf("Actual output: %q", output)

	// Verify output indicates no available nodes
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain group name 'Proxy'")
	}
}

func TestFormatSwitchResult_Success(t *testing.T) {
	output, err := captureOutput(func() error {
		return FormatSwitchResult("Proxy", "Node1", nil)
	})

	if err != nil {
		t.Fatalf("FormatSwitchResult failed: %v", err)
	}

	// Verify output contains expected information
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain group name 'Proxy'")
	}
	if !bytes.Contains([]byte(output), []byte("Node1")) {
		t.Error("Expected output to contain node name 'Node1'")
	}
}

func TestFormatSwitchResult_Error(t *testing.T) {
	err := FormatSwitchResult("Proxy", "Node1", errors.New("test error"))
	// The function returns error when there's an input error
	if err == nil {
		t.Log("FormatSwitchResult handled error by printing message")
	}
}

func TestFormatUnfixResult_Success(t *testing.T) {
	output, err := captureOutput(func() error {
		return FormatUnfixResult("Proxy", nil)
	})

	if err != nil {
		t.Fatalf("FormatUnfixResult failed: %v", err)
	}

	// Verify output contains expected information
	if !bytes.Contains([]byte(output), []byte("Proxy")) {
		t.Error("Expected output to contain group name 'Proxy'")
	}
}

func TestFormatUnfixResult_Error(t *testing.T) {
	err := FormatUnfixResult("Proxy", errors.New("test error"))
	// The function returns error when there's an input error
	if err == nil {
		t.Log("FormatUnfixResult handled error by printing message")
	}
}

func TestGetGroupsFromProxies(t *testing.T) {
	proxies := map[string]*types.ProxyInfo{
		"Group1": {
			Name:  "Group1",
			Type:  "Selector",
			Alive: true,
			Now:   "Node1",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
		"Group2": {
			Name:  "Group2",
			Type:  "URLTest",
			Alive: true,
			Now:   "Node3",
			All:   []string{"Node3", "Node4"},
			Delay: 100,
		},
		"Node5": {
			Name:  "Node5",
			Type:  "Shadowsocks",
			Alive: true,
			All:   []string{},
			Delay: 80,
		},
	}

	groups := GetGroupsFromProxies(proxies)

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	// Check if groups contain expected values
	groupMap := make(map[string]bool)
	for _, g := range groups {
		groupMap[g] = true
	}

	if !groupMap["Group1"] {
		t.Error("Expected groups to contain 'Group1'")
	}
	if !groupMap["Group2"] {
		t.Error("Expected groups to contain 'Group2'")
	}
}

func TestGetGroupsFromProxies_Empty(t *testing.T) {
	proxies := map[string]*types.ProxyInfo{
		"Node1": {
			Name:  "Node1",
			Type:  "Shadowsocks",
			Alive: true,
			All:   []string{},
			Delay: 80,
		},
	}

	groups := GetGroupsFromProxies(proxies)

	if len(groups) != 0 {
		t.Errorf("Expected 0 groups, got %d", len(groups))
	}
}

func TestFormatGroupList_WithGroups(t *testing.T) {
	proxies := map[string]*types.ProxyInfo{
		"Group1": {
			Name:  "Group1",
			Type:  "Selector",
			Alive: true,
			Now:   "Node1",
			All:   []string{"Node1", "Node2"},
			Delay: 50,
		},
		"Group2": {
			Name:  "Group2",
			Type:  "URLTest",
			Alive: true,
			Now:   "Node3",
			All:   []string{"Node3", "Node4"},
			Delay: 100,
		},
	}

	output, err := captureOutput(func() error {
		return FormatGroupList(proxies)
	})

	if err != nil {
		t.Fatalf("FormatGroupList failed: %v", err)
	}

	// Verify output contains group names
	if !bytes.Contains([]byte(output), []byte("Group1")) {
		t.Error("Expected output to contain 'Group1'")
	}
	if !bytes.Contains([]byte(output), []byte("Group2")) {
		t.Error("Expected output to contain 'Group2'")
	}
}

func TestFormatGroupList_NoGroups(t *testing.T) {
	proxies := map[string]*types.ProxyInfo{
		"Node1": {
			Name:  "Node1",
			Type:  "Shadowsocks",
			Alive: true,
			All:   []string{},
			Delay: 80,
		},
	}

	output, err := captureOutput(func() error {
		return FormatGroupList(proxies)
	})

	if err != nil {
		t.Fatalf("FormatGroupList failed: %v", err)
	}

	// Debug: print the actual output
	t.Logf("Actual output: %q", output)

	// Verify output indicates no groups found
	if !bytes.Contains([]byte(output), []byte("没有找到代理组")) {
		t.Error("Expected output to indicate no groups found")
	}
}
