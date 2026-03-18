package api

import (
	"context"
	"fmt"
	"time"
)

// ExampleUsage 演示 API 客户端的使用方法
//
// 这个示例展示了如何创建和使用 Mihomo API 客户端：
//
// 1. 创建客户端
// 2. 使用不同的 HTTP 方法
// 3. 处理错误响应
//
// 示例代码：
//
//	// 创建客户端
//	client := api.NewClient(
//	    "http://127.0.0.1:9090",
//	    "your-secret-here",
//	    api.WithTimeout(15*time.Second),
//	)
//
//	// 使用 context 设置超时
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	// 获取版本信息
//	var version map[string]interface{}
//	err := client.Get(ctx, "/version", nil, &version)
//	if err != nil {
//	    fmt.Printf("Error: %v\n", err)
//	    return
//	}
//	fmt.Printf("Version: %v\n", version)
//
//	// POST 请求示例
//	data := map[string]interface{}{
//	    "mode": "rule",
//	}
//	err = client.Patch(ctx, "/configs", nil, data, nil)
//	if err != nil {
//	    fmt.Printf("Error: %v\n", err)
//	    return
//	}
//	fmt.Println("Config updated successfully")
//
//	// 错误处理示例
//	err = client.Get(ctx, "/proxies/invalid-proxy", nil, nil)
//	if err != nil {
//	    if api.IsNotFoundError(err) {
//	        fmt.Println("Proxy not found")
//	    } else if api.IsAPIConnectionError(err) {
//	        fmt.Println("Failed to connect to API")
//	    } else if api.IsAPIAuthError(err) {
//	        fmt.Println("Authentication failed")
//	    } else {
//	        fmt.Printf("Error: %v\n", err)
//	    }
//	}
func ExampleUsage() {
	fmt.Println("API Client Usage Example:")
	fmt.Println("=========================")
	fmt.Println()
	fmt.Println("1. Creating a client:")
	fmt.Println("   client := api.NewClient(")
	fmt.Println("       \"http://127.0.0.1:9090\",")
	fmt.Println("       \"your-secret-here\",")
	fmt.Println("       api.WithTimeout(15*time.Second),")
	fmt.Println("   )")
	fmt.Println()
	fmt.Println("2. Making GET requests:")
	fmt.Println("   var version map[string]interface{}")
	fmt.Println("   err := client.Get(ctx, \"/version\", nil, &version)")
	fmt.Println()
	fmt.Println("3. Making POST requests:")
	fmt.Println("   data := map[string]interface{}{\"mode\": \"rule\"}")
	fmt.Println("   err := client.Patch(ctx, \"/configs\", nil, data, nil)")
	fmt.Println()
	fmt.Println("4. Error handling:")
	fmt.Println("   if api.IsNotFoundError(err) {")
	fmt.Println("       fmt.Println(\"Resource not found\")")
	fmt.Println("   } else if api.IsAPIConnectionError(err) {")
	fmt.Println("       fmt.Println(\"Connection failed\")")
	fmt.Println("   }")
	fmt.Println()
	fmt.Println("For complete API documentation, see docs/spec/mihono-api.md")
}

// ExampleCreateClient 创建客户端的示例
func ExampleCreateClient() {
	// 创建带有默认超时的客户端
	client := NewClient("http://127.0.0.1:9090", "secret")

	// 创建带有自定义超时的客户端
	client = NewClient(
		"http://127.0.0.1:9090",
		"secret",
		WithTimeout(15*time.Second),
	)

	// 使用兼容旧接口的方法
	client = NewClientWithTimeout("http://127.0.0.1:9090", "secret", 15)

	_ = client
}

// ExampleGetRequest GET 请求示例
func ExampleGetRequest() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// 无查询参数
	var result map[string]interface{}
	err := client.Get(ctx, "/version", nil, &result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 带查询参数
	queryParams := map[string]string{
		"name": "google.com",
		"type": "A",
	}
	err = client.Get(ctx, "/dns/query", queryParams, &result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExamplePostRequest POST 请求示例
func ExamplePostRequest() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// POST 请求
	data := map[string]interface{}{
		"path": "/path/to/config.yaml",
	}
	err := client.Post(ctx, "/configs", nil, data, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExamplePutRequest PUT 请求示例
func ExamplePutRequest() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// PUT 请求 - 切换代理
	data := map[string]interface{}{
		"name": "proxy-name",
	}
	err := client.Put(ctx, "/proxies/Proxy", nil, data, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExamplePatchRequest PATCH 请求示例
func ExamplePatchRequest() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// PATCH 请求 - 更新配置
	data := map[string]interface{}{
		"mode":     "rule",
		"log-level": "info",
	}
	err := client.Patch(ctx, "/configs", nil, data, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExampleDeleteRequest DELETE 请求示例
func ExampleDeleteRequest() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// DELETE 请求 - 关闭连接
	err := client.Delete(ctx, "/connections/conn-id", nil, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

// ExampleErrorHandling 错误处理示例
func ExampleErrorHandling() {
	client := NewClient("http://127.0.0.1:9090", "secret")
	ctx := context.Background()

	// 请求可能失败
	var result map[string]interface{}
	err := client.Get(ctx, "/proxies/nonexistent", nil, &result)
	if err != nil {
		// 检查错误类型
		switch {
		case IsAPIConnectionError(err):
			fmt.Println("Failed to connect to API server")
		case IsAPIAuthError(err):
			fmt.Println("Authentication failed. Check your secret.")
		case IsTimeoutError(err):
			fmt.Println("Request timeout. Try increasing timeout.")
		case IsNotFoundError(err):
			fmt.Println("Resource not found")
		default:
			// 获取 APIError 以访问更多详情
			if apiErr, ok := err.(*APIError); ok {
				fmt.Printf("API Error [%d]: %s\n", apiErr.Code, apiErr.Message)
				if apiErr.Cause != nil {
					fmt.Printf("Caused by: %v\n", apiErr.Cause)
				}
			} else {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}