// Package cmd 实现 zenmux-cli 的全部命令行子命令。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/client"
	"github.com/yinghaowu/zenmux-cli/internal/config"
)

// 全局配置与运行时状态，在 PersistentPreRun 中初始化。
var (
	cfg        *config.Config
	flagProxy  string // --proxy 覆盖
	flagConfig string // --config 指定配置文件路径
)

// rootCmd 根命令。
var rootCmd = &cobra.Command{
	Use:   "zenmux",
	Short: "zenmux.ai 大模型聚合 API 的命令行工具",
	Long: `zenmux 是 zenmux.ai 大模型聚合服务的命令行客户端。

支持多模态大模型调用：文本对话、图片生成、视频生成、语音合成与识别，
以及套餐状态查询、模型列表查询等功能。

首次使用请先配置 API Key：
  zenmux config set api_key <你的Key>
  zenmux config set management_api_key <你的Management Key>
  zenmux config set proxy http://127.0.0.1:7897

完整文档：https://docs.zenmux.ai`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// config / help 命令不需要加载配置。
		if cmd.Name() == "config" || cmd.Name() == "help" || cmd.Name() == "completion" {
			return nil
		}
		return loadConfig()
	},
}

// loadConfig 加载配置并应用 flag/环境变量覆盖。
func loadConfig() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	c.ApplyEnv(nil)
	// --proxy 覆盖。
	if flagProxy != "" {
		c.Proxy = flagProxy
	}
	cfg = c
	return nil
}

// newClient 用当前配置构造 API 客户端。
func newClient() *client.Client {
	return client.New(cfg.BaseURL, cfg.APIKey, cfg.ManagementAPIKey, cfg.Proxy)
}

// Execute 启动命令行。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// cobra 已打印错误，这里仅设置退出码。
		os.Exit(1)
	}
}

// init 注册全局 flag。
func init() {
	rootCmd.PersistentFlags().StringVar(&flagProxy, "proxy", "", "HTTP 代理地址（覆盖配置文件）")
}

// die 打印错误并退出。供子命令使用。
func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "错误: "+format+"\n", a...)
	os.Exit(1)
}
