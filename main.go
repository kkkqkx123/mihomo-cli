package main

import (
	"github.com/kkkqkx123/mihomo-cli/cmd"
	internalerrors "github.com/kkkqkx123/mihomo-cli/internal/errors"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if err := cmd.Execute(version, commit); err != nil {
		// 统一错误收口：打印错误信息和建议，并按 CLIError.Code（2~8）退出
		internalerrors.ExitWithError(err, false)
	}
}
