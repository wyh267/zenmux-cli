// Package config 负责 zenmux-cli 的配置加载、保存与优先级合并。
//
// 配置文件默认位于 ~/.config/zenmux/config.yaml（遵循 XDG规范）。
// 优先级：命令行 flag > 环境变量 > 配置文件 > 内置默认值。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultModels 内置的各模态默认模型 slug。
var DefaultModels = Models{
	Chat:  "openai/gpt-5",
	Image: "gpt-image-2",
	Video: "bytedance/doubao-seedance-2.0",
	TTS:   "google/gemini-3.1-flash-tts-preview",
	STT:   "qwen/qwen3-asr-flash",
	Eval:  "typesafe/jev-latest",
}

// Models 各子命令的默认模型配置。
type Models struct {
	Chat  string `yaml:"chat"`
	Image string `yaml:"image"`
	Video string `yaml:"video"`
	TTS   string `yaml:"tts"`
	STT   string `yaml:"stt"`
	Eval  string `yaml:"eval"`
}

// Config zenmux-cli 的完整配置。
type Config struct {
	APIKey           string `yaml:"api_key"`
	ManagementAPIKey string `yaml:"management_api_key"`
	BaseURL          string `yaml:"base_url"`
	Proxy            string `yaml:"proxy"`
	Models           Models `yaml:"default_model"`

	// ConfigPath 当前加载/保存使用的配置文件路径（不写入 yaml）。
	ConfigPath string `yaml:"-"`
}

// Default 返回带内置默认值的配置。
func Default() *Config {
	return &Config{
		BaseURL: "https://zenmux.ai/api/v1",
		Models:  DefaultModels,
	}
}

// configPath 返回配置文件路径，遵循 XDG_CONFIG_HOME。
func configPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zenmux", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zenmux", "config.yaml"), nil
}

// Load 从默认路径加载配置；文件不存在时返回带默认值的空配置。
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	cfg := Default()
	cfg.ConfigPath = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在不算错误，返回默认配置。
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	// yaml 反序列化可能把默认值覆盖为空串，补回空缺。
	if cfg.BaseURL == "" {
		cfg.BaseURL = Default().BaseURL
	}
	if cfg.Models.Chat == "" {
		cfg.Models.Chat = DefaultModels.Chat
	}
	if cfg.Models.Image == "" {
		cfg.Models.Image = DefaultModels.Image
	}
	if cfg.Models.Video == "" {
		cfg.Models.Video = DefaultModels.Video
	}
	if cfg.Models.TTS == "" {
		cfg.Models.TTS = DefaultModels.TTS
	}
	if cfg.Models.STT == "" {
		cfg.Models.STT = DefaultModels.STT
	}
	if cfg.Models.Eval == "" {
		cfg.Models.Eval = DefaultModels.Eval
	}
	return cfg, nil
}

// Save 将配置写入文件（自动创建目录）。
func (c *Config) Save() error {
	path := c.ConfigPath
	if path == "" {
		var err error
		path, err = configPath()
		if err != nil {
			return err
		}
		c.ConfigPath = path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// ApplyEnv 用环境变量覆盖配置中的空缺项。
// 环境变量：ZENMUX_API_KEY、ZENMUX_MANAGEMENT_API_KEY、ZENMUX_BASE_URL、ZENMUX_PROXY。
func (c *Config) ApplyEnv(env func(string) string) {
	if env == nil {
		env = os.Getenv
	}
	if c.APIKey == "" {
		c.APIKey = env("ZENMUX_API_KEY")
	}
	if c.ManagementAPIKey == "" {
		c.ManagementAPIKey = env("ZENMUX_MANAGEMENT_API_KEY")
	}
	if c.BaseURL == "" {
		c.BaseURL = env("ZENMUX_BASE_URL")
	}
	if c.Proxy == "" {
		c.Proxy = env("ZENMUX_PROXY")
	}
}
