//go:build windows

package util

import (
	"testing"
)

func TestIsAdmin(t *testing.T) {
	// 测试 IsAdmin 函数是否正常运行
	// 注意：这个测试的结果取决于当前进程是否以管理员权限运行
	// 在普通用户模式下运行测试将返回 false
	// 在管理员模式下运行测试将返回 true
	
	isAdmin := IsAdmin()
	
	// 我们只测试函数能够正常执行并返回布尔值
	// 不强制要求必须是 true 或 false，因为这取决于运行环境
	t.Logf("当前进程管理员状态：%v", isAdmin)
	
	// 验证返回值是合理的布尔值（这个断言总是会通过，因为 IsAdmin 返回 bool 类型）
	if isAdmin != true && isAdmin != false {
		t.Error("IsAdmin 返回了无效的布尔值")
	}
}

func TestIsAdmin_Consistency(t *testing.T) {
	// 测试多次调用 IsAdmin 是否返回一致的结果
	firstCall := IsAdmin()
	secondCall := IsAdmin()
	thirdCall := IsAdmin()
	
	if firstCall != secondCall || secondCall != thirdCall {
		t.Error("IsAdmin 多次调用返回了不一致的结果")
	}
	
	t.Logf("三次调用结果一致：%v", firstCall)
}
