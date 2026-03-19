package output

import (
	"errors"
	"os"
	"path/filepath"
)

// LogOutput 日志输出管理器
type LogOutput struct {
	file   *os.File
	writer Writer
}

// InitLogOutput 初始化日志输出
// mode: console - 仅终端, file - 仅文件, both - 终端+文件
func InitLogOutput(filePath, mode string, append bool) (*LogOutput, error) {
	lo := &LogOutput{}

	switch mode {
	case "console", "":
		// 仅终端
		lo.writer = os.Stdout
		return lo, nil

	case "file":
		// 仅文件
		if filePath == "" {
			return nil, errors.New("log file path is required")
		}

		file, err := lo.openFile(filePath, append)
		if err != nil {
			return nil, err
		}
		lo.file = file
		lo.writer = file
		return lo, nil

	case "both":
		// 终端+文件
		if filePath == "" {
			return nil, errors.New("log file path is required")
		}

		file, err := lo.openFile(filePath, append)
		if err != nil {
			return nil, err
		}
		lo.file = file
		lo.writer = NewMultiWriter(os.Stdout, file)
		return lo, nil

	default:
		return nil, errors.New("invalid log mode: " + mode)
	}
}

// openFile 打开日志文件
func (lo *LogOutput) openFile(filePath string, append bool) (*os.File, error) {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// 打开文件
	if append {
		return os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}
	return os.Create(filePath)
}

// GetWriter 获取写入器
func (lo *LogOutput) GetWriter() Writer {
	return lo.writer
}

// Close 关闭日志输出
func (lo *LogOutput) Close() error {
	if lo.file != nil {
		return lo.file.Close()
	}
	return nil
}
