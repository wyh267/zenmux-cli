package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/config"
)

// configCmd config 子命令。
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "查看与修改配置",
	Long: `管理 zenmux-cli 的配置（位于 ~/.config/zenmux/config.yaml）。

可用配置项：
  api_key              普通 API Key（调用模型）
  management_api_key   Management API Key（查询套餐/用量）
  base_url             API 基础地址（默认 https://zenmux.ai/api/v1）
  proxy                HTTP 代理地址（如 http://127.0.0.1:7897）
  default_model.chat   chat 子命令的默认模型
  default_model.image  image 子命令的默认模型
  default_model.video  video 子命令的默认模型
  default_model.tts    voice 子命令的默认 TTS 模型
  default_model.stt    voice transcribe 的默认 STT 模型`,
}

// configShowCmd 显示当前配置（Key 脱敏）。
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("配置文件: %s\n", cfg.ConfigPath)
		fmt.Println("---")
		fmt.Printf("api_key:              %s\n", maskKey(cfg.APIKey))
		fmt.Printf("management_api_key:   %s\n", maskKey(cfg.ManagementAPIKey))
		fmt.Printf("base_url:             %s\n", cfg.BaseURL)
		fmt.Printf("proxy:                %s\n", orNA(cfg.Proxy))
		fmt.Println("default_model:")
		fmt.Printf("  chat:               %s\n", cfg.Models.Chat)
		fmt.Printf("  image:              %s\n", cfg.Models.Image)
		fmt.Printf("  video:              %s\n", cfg.Models.Video)
		fmt.Printf("  tts:                %s\n", cfg.Models.TTS)
		fmt.Printf("  stt:                %s\n", cfg.Models.STT)
		return nil
	},
}

// configSetCmd 设置某个配置项。
//
// 语法：zenmux config set <key> <value>
// key 支持点号路径，如 default_model.chat。
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "设置某个配置项",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if err := setConfigField(cfg, key, value); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("已设置 %s = %s\n", key, maskIfNeeded(key, value))
		fmt.Printf("配置文件：%s\n", cfg.ConfigPath)
		return nil
	},
}

// setConfigField 根据点号路径设置配置字段。
func setConfigField(cfg *config.Config, key, value string) error {
	switch key {
	case "api_key":
		cfg.APIKey = value
	case "management_api_key":
		cfg.ManagementAPIKey = value
	case "base_url":
		cfg.BaseURL = value
	case "proxy":
		cfg.Proxy = value
	case "default_model.chat":
		cfg.Models.Chat = value
	case "default_model.image":
		cfg.Models.Image = value
	case "default_model.video":
		cfg.Models.Video = value
	case "default_model.tts":
		cfg.Models.TTS = value
	case "default_model.stt":
		cfg.Models.STT = value
	default:
		return fmt.Errorf("未知的配置项 %q，可用项见 `zenmux config --help`", key)
	}
	return nil
}

// maskKey 对 Key 做脱敏，仅保留前4后4。
func maskKey(key string) string {
	if key == "" {
		return "(未设置)"
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// maskIfNeeded 仅对包含 key 的配置项脱敏。
func maskIfNeeded(key, value string) string {
	if strings.Contains(key, "api_key") || strings.Contains(key, "secret") {
		return maskKey(value)
	}
	return value
}

// orNA 空值显示为 N/A。
func orNA(s string) string {
	if s == "" {
		return "(未设置)"
	}
	return s
}

func init() {
	configCmd.AddCommand(configShowCmd, configSetCmd)
	rootCmd.AddCommand(configCmd)
}
