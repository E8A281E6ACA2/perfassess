// Package main 是Perfassess的主入口点
// 负责初始化应用程序并启动命令行界面
package main

import (
	"fmt"
	"os"

	"github.com/E8A281E6ACA2/perfassess/internal/cli"
)

// Version 应用程序版本号，可在发布构建时通过 -ldflags 注入。
var Version = "1.0.0"

// main 函数是程序的入口点
// 初始化CLI并执行命令
func main() {
	// 创建CLI实例
	cliApp := cli.NewCLIWithVersion(Version)

	// 执行CLI
	if err := cliApp.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// 正常退出
	os.Exit(0)
}
