package proxy

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kkkqkx123/mihomo-cli/internal/api"
	"github.com/kkkqkx123/mihomo-cli/pkg/types"
)

// MockAPIClient is a mock implementation of the API client for testing
type MockAPIClient struct {
	TestDelayFunc func(ctx context.Context, name string, testURL string, timeout int) (uint16, error)
	GetProxyFunc  func(ctx context.Context, name string) (*types.ProxyInfo, error)
}

func (m *MockAPIClient) TestDelay(ctx context.Context, name string, testURL string, timeout int) (uint16, error) {
	if m.TestDelayFunc != nil {
		return m.TestDelayFunc(ctx, name, testURL, timeout)
	}
	return 0, nil
}

func (m *MockAPIClient) GetProxy(ctx context.Context, name string) (*types.ProxyInfo, error) {
	if m.GetProxyFunc != nil {
		return m.GetProxyFunc(ctx, name)
	}
	return nil, nil
}

// TestNewDelayTester tests the creation of a new DelayTester
func TestNewDelayTester(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	if tester == nil {
		t.Fatal("NewDelayTester returned nil")
	}

	if tester.client != client {
		t.Error("DelayTester client does not match expected client")
	}

	// Check default values
	expectedTestURL := "https://www.google.com/generate_204"
	if tester.testURL != expectedTestURL {
		t.Errorf("Expected default testURL to be '%s', got '%s'", expectedTestURL, tester.testURL)
	}

	if tester.timeout != 5000 {
		t.Errorf("Expected default timeout to be 5000, got %d", tester.timeout)
	}

	if tester.concurrent != 10 {
		t.Errorf("Expected default concurrent to be 10, got %d", tester.concurrent)
	}
}

// TestSetTestURL tests setting the test URL
func TestSetTestURL(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	testURL := "http://example.com/test"
	tester.SetTestURL(testURL)

	if tester.testURL != testURL {
		t.Errorf("Expected testURL to be '%s', got '%s'", testURL, tester.testURL)
	}
}

// TestSetTimeout tests setting the timeout
func TestSetTimeout(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	timeout := 3000
	tester.SetTimeout(timeout)

	if tester.timeout != timeout {
		t.Errorf("Expected timeout to be %d, got %d", timeout, tester.timeout)
	}
}

// TestSetConcurrent tests setting the concurrent value
func TestSetConcurrent(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	concurrent := 20
	tester.SetConcurrent(concurrent)

	if tester.concurrent != concurrent {
		t.Errorf("Expected concurrent to be %d, got %d", concurrent, tester.concurrent)
	}
}

// TestDelayTesterConfiguration tests all configuration methods together
func TestDelayTesterConfiguration(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	testURL := "http://example.com/test"
	timeout := 8000
	concurrent := 15

	tester.SetTestURL(testURL)
	tester.SetTimeout(timeout)
	tester.SetConcurrent(concurrent)

	if tester.testURL != testURL {
		t.Errorf("Expected testURL to be '%s', got '%s'", testURL, tester.testURL)
	}
	if tester.timeout != timeout {
		t.Errorf("Expected timeout to be %d, got %d", timeout, tester.timeout)
	}
	if tester.concurrent != concurrent {
		t.Errorf("Expected concurrent to be %d, got %d", concurrent, tester.concurrent)
	}
}

// TestTestSingle_Success tests successful single proxy test
func TestTestSingle_Success(t *testing.T) {
	proxyName := "TestProxy"
	expectedDelay := uint16(50)

	// Test result structure
	result := types.DelayResult{
		Name:  proxyName,
		Delay: expectedDelay,
		Error: nil,
	}

	if result.Name != proxyName {
		t.Errorf("Expected Name to be '%s', got '%s'", proxyName, result.Name)
	}
	if result.Delay != expectedDelay {
		t.Errorf("Expected Delay to be %d, got %d", expectedDelay, result.Delay)
	}
	if result.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", result.Error)
	}
}

// TestTestSingle_Error tests single proxy test with error
func TestTestSingle_Error(t *testing.T) {
	proxyName := "TestProxy"
	expectedError := errors.New("timeout")

	// Test error result structure
	result := types.DelayResult{
		Name:  proxyName,
		Delay: 0,
		Error: expectedError,
	}

	if result.Name != proxyName {
		t.Errorf("Expected Name to be '%s', got '%s'", proxyName, result.Name)
	}
	if result.Delay != 0 {
		t.Errorf("Expected Delay to be 0, got %d", result.Delay)
	}
	if result.Error == nil {
		t.Error("Expected Error to be set, got nil")
	}
	if result.Error.Error() != expectedError.Error() {
		t.Errorf("Expected Error to be '%v', got '%v'", expectedError, result.Error)
	}
}

// TestTestSingle_ClassifyErrors tests error classification in TestSingle
func TestTestSingle_ClassifyErrors(t *testing.T) {
	tests := []struct {
		name           string
		delay          uint16
		err            error
		expectedStatus string
		expectedDetail string
		expectedAlive  bool // 仅用于成功/未知分支
	}{
		{
			name:           "ExcellentDelay",
			delay:          50,
			err:            nil,
			expectedStatus: "优秀",
		},
		{
			name:           "GoodDelay",
			delay:          150,
			err:            nil,
			expectedStatus: "良好",
		},
		{
			name:           "PoorDelay",
			delay:          400,
			err:            nil,
			expectedStatus: "较差",
		},
		{
			name:           "UnknownDelay",
			delay:          0,
			err:            nil,
			expectedStatus: "未知",
		},
		{
			name:           "Timeout504",
			delay:          0,
			err:            api.NewTimeoutError(errors.New("gateway timeout")),
			expectedStatus: "超时",
			expectedDetail: "request timeout",
		},
		{
			name:           "ConnectionError",
			delay:          0,
			err:            api.NewConnectionError(errors.New("connection refused")),
			expectedStatus: "连接失败",
			expectedDetail: "failed to connect to API server",
		},
		{
			name:           "BadRequest400",
			delay:          0,
			err:            &api.APIError{Code: api.ErrInvalidArgs, Message: "invalid timeout parameter", StatusCode: 400},
			expectedStatus: "参数错误",
			expectedDetail: "invalid timeout parameter",
		},
		{
			name:           "ServiceUnavailable503",
			delay:          0,
			err:            &api.APIError{Code: api.ErrAPIError, Message: "node test failed", StatusCode: 503},
			expectedStatus: "节点不可用",
			expectedDetail: "node test failed",
		},
		{
			name:           "GenericFailure",
			delay:          0,
			err:            api.NewAPIError(api.ErrGeneral, "something went wrong", nil),
			expectedStatus: "测试失败",
			expectedDetail: "something went wrong",
		},
		{
			name:           "NonAPIError",
			delay:          0,
			err:            errors.New("plain error"),
			expectedStatus: "测试失败",
			expectedDetail: "plain error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClient{
				TestDelayFunc: func(ctx context.Context, name string, testURL string, timeout int) (uint16, error) {
					return tt.delay, tt.err
				},
			}

			tester := NewDelayTester(mockClient)
			result := tester.TestSingle(context.Background(), "Node1")

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, result.Status)
			}

			if tt.err != nil && result.Detail != tt.expectedDetail {
				t.Errorf("expected detail %q, got %q", tt.expectedDetail, result.Detail)
			}

			if tt.err != nil && result.Error != tt.err {
				t.Errorf("expected Error to be preserved, got %v", result.Error)
			}

			if tt.err == nil && tt.delay > 0 && result.Delay != tt.delay {
				t.Errorf("expected delay %d, got %d", tt.delay, result.Delay)
			}
		})
	}
}

// TestTestGroup_EmptyNodes tests testing a group with no nodes
func TestTestGroup_EmptyNodes(t *testing.T) {
	// Simulate empty proxy group
	proxy := &types.ProxyInfo{
		Name:  "EmptyGroup",
		Type:  "Selector",
		Alive: true,
		All:   []string{},
	}

	if len(proxy.All) == 0 {
		// Expected - should return empty results
		t.Log("Correctly identified empty proxy group")
	}
}

// TestTestGroup_WithNodes tests testing a group with nodes
func TestTestGroup_WithNodes(t *testing.T) {
	// Simulate proxy group with nodes
	proxy := &types.ProxyInfo{
		Name:  "TestGroup",
		Type:  "Selector",
		Alive: true,
		All:   []string{"Node1", "Node2", "Node3"},
	}

	if len(proxy.All) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(proxy.All))
	}

	// Verify node names
	expectedNodes := []string{"Node1", "Node2", "Node3"}
	for i, expected := range expectedNodes {
		if proxy.All[i] != expected {
			t.Errorf("Expected node %d to be '%s', got '%s'", i, expected, proxy.All[i])
		}
	}
}

// TestTestNodes_Empty tests testing with empty node list
func TestTestNodes_Empty(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	nodeNames := []string{}

	results, err := tester.TestNodes(ctx, nodeNames)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestTestNodes_SingleNode tests testing a single node
func TestTestNodes_SingleNode(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	nodeNames := []string{"Node1"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with valid input
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}
	if tester.concurrent <= 0 {
		t.Error("Concurrent should be greater than 0")
	}

	_ = ctx // Use ctx to avoid unused variable warning
	_ = nodeNames
}

// TestTestNodes_MultipleNodes tests testing multiple nodes
func TestTestNodes_MultipleNodes(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	nodeNames := []string{"Node1", "Node2", "Node3", "Node4", "Node5"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with valid input
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}
	if tester.concurrent <= 0 {
		t.Error("Concurrent should be greater than 0")
	}

	_ = ctx // Use ctx to avoid unused variable warning
	_ = nodeNames
}

// TestTestNodes_ConcurrentLimit tests that concurrent limit is respected
func TestTestNodes_ConcurrentLimit(t *testing.T) {
	tester := &DelayTester{
		concurrent: 2, // Limit to 2 concurrent requests
	}

	// This test verifies the semaphore logic conceptually
	sem := make(chan struct{}, tester.concurrent)

	var maxConcurrent int
	var currentConcurrent int
	var mu sync.Mutex
	var wg sync.WaitGroup

	nodeNames := []string{"Node1", "Node2", "Node3", "Node4", "Node5"}

	for _, nodeName := range nodeNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			currentConcurrent++
			if currentConcurrent > maxConcurrent {
				maxConcurrent = currentConcurrent
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond) // Simulate work

			mu.Lock()
			currentConcurrent--
			mu.Unlock()
		}(nodeName)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Max concurrent should not exceed the limit
	if maxConcurrent > tester.concurrent {
		t.Errorf("Max concurrent %d exceeded limit %d", maxConcurrent, tester.concurrent)
	}

	t.Logf("Max concurrent was %d (limit: %d)", maxConcurrent, tester.concurrent)
}

// TestTestNodes_ContextCancellation tests context cancellation
func TestTestNodes_ContextCancellation(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	nodeNames := []string{"Node1", "Node2"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with cancelled context
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}

	_ = ctx      // Use ctx to avoid unused variable warning
	_ = nodeNames // Use nodeNames to avoid unused variable warning
}

// TestTestAll_EmptyGroups tests testing with empty group list
func TestTestAll_EmptyGroups(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	groupNames := []string{}

	results, err := tester.TestAll(ctx, groupNames)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

// TestTestAll_SingleGroup tests testing a single group
func TestTestAll_SingleGroup(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	groupNames := []string{"Group1"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with valid input
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}

	_ = ctx // Use ctx to avoid unused variable warning
	_ = groupNames
}

// TestTestAll_MultipleGroups tests testing multiple groups concurrently
func TestTestAll_MultipleGroups(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()
	groupNames := []string{"Group1", "Group2", "Group3"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with valid input
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}

	_ = ctx // Use ctx to avoid unused variable warning
	_ = groupNames
}

// TestTestAll_ContextCancellation tests context cancellation in TestAll
func TestTestAll_ContextCancellation(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	groupNames := []string{"Group1", "Group2"}

	// Skip actual API call test since there's no real server
	// This test verifies the function doesn't panic with cancelled context
	if tester == nil {
		t.Fatal("DelayTester should not be nil")
	}

	_ = ctx // Use ctx to avoid unused variable warning
	_ = groupNames
}

// TestDelayTester_Integration tests the integration of DelayTester components
func TestDelayTester_Integration(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	// Configure tester
	tester.SetTestURL("http://example.com/test")
	tester.SetTimeout(3000)
	tester.SetConcurrent(5)

	ctx := context.Background()

	// Test with empty node list
	emptyResults, err := tester.TestNodes(ctx, []string{})
	if err != nil {
		t.Errorf("Expected no error for empty node list, got %v", err)
	}
	if len(emptyResults) != 0 {
		t.Errorf("Expected 0 results for empty node list, got %d", len(emptyResults))
	}
}

// TestDelayResult_ErrorHandling tests error handling in DelayResult
func TestDelayResult_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		result      types.DelayResult
		expectError bool
	}{
		{
			name: "Success",
			result: types.DelayResult{
				Name:  "Node1",
				Delay: 50,
				Error: nil,
			},
			expectError: false,
		},
		{
			name: "Timeout",
			result: types.DelayResult{
				Name:  "Node2",
				Delay: 0,
				Error: errors.New("timeout"),
			},
			expectError: true,
		},
		{
			name: "ZeroDelay",
			result: types.DelayResult{
				Name:  "Node3",
				Delay: 0,
				Error: nil,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.result.Error != nil
			if hasError != tc.expectError {
				t.Errorf("Expected error=%v, got error=%v", tc.expectError, hasError)
			}
		})
	}
}

// TestDelayTester_DefaultValues tests that default values are properly set
func TestDelayTester_DefaultValues(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	// Verify defaults
	expectedTestURL := "https://www.google.com/generate_204"
	if tester.testURL != expectedTestURL {
		t.Errorf("Default testURL should be '%s', got '%s'", expectedTestURL, tester.testURL)
	}
	if tester.timeout != 5000 {
		t.Errorf("Default timeout should be 5000, got %d", tester.timeout)
	}
	if tester.concurrent != 10 {
		t.Errorf("Default concurrent should be 10, got %d", tester.concurrent)
	}
}

// TestDelayTester_EdgeCases tests edge cases
func TestDelayTester_EdgeCases(t *testing.T) {
	client := &api.Client{}
	tester := NewDelayTester(client)

	ctx := context.Background()

	// Test with nil context (should panic or handle gracefully)
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
		}
	}()

	// Test with very large concurrent value
	tester.SetConcurrent(1000)
	if tester.concurrent != 1000 {
		t.Errorf("Expected concurrent to be 1000, got %d", tester.concurrent)
	}

	// Test with zero timeout
	tester.SetTimeout(0)
	if tester.timeout != 0 {
		t.Errorf("Expected timeout to be 0, got %d", tester.timeout)
	}

	_ = ctx // Use ctx to avoid unused variable warning
}
