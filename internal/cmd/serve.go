package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/server"
)

var (
	serveAddr  string
	serveToken string
)

// serveCmd 常驻 HTTP 网关服务。
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "以 HTTP 服务方式常驻运行，供其他服务调用 zenmux",
	Long: `启动一个常驻 HTTP 网关，把 zenmux 的能力暴露给其他服务调用。
API Key 留在服务端配置中，调用方只需携带 --token 指定的访问令牌。

端点：
  GET  /healthz          健康检查
  GET  /v1/models        模型列表
  GET  /v1/status        套餐状态/余额/汇率
  POST /v1/chat          文本对话（"stream":true 时 SSE 流式返回）
  POST /v1/eval          结构化评估（jev）

示例：
  zenmux serve                          # 监听 127.0.0.1:8310
  zenmux serve --addr :8310 --token xxx # 监听所有网卡并启用访问令牌

调用示例：
  curl -H "Authorization: Bearer xxx" http://127.0.0.1:8310/v1/models
  curl -X POST http://127.0.0.1:8310/v1/chat \
    -H "Authorization: Bearer xxx" -H "Content-Type: application/json" \
    -d '{"messages":[{"role":"user","content":"你好"}]}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 未配置令牌时只允许监听回环地址，避免把未鉴权服务暴露到局域网。
		host, _, err := net.SplitHostPort(serveAddr)
		if err != nil {
			return fmt.Errorf("--addr 格式无效: %w", err)
		}
		if serveToken == "" && !isLoopbackHost(host) {
			return fmt.Errorf("未设置 --token 时只能监听回环地址（当前 %q）；对外监听请配置访问令牌", serveAddr)
		}

		srv := server.New(newClient(), cfg.Models, serveToken)
		fmt.Printf("zenmux 网关已启动: http://%s\n", serveAddr)
		if serveToken == "" {
			fmt.Println("提示：未配置访问令牌，仅监听回环地址")
		}
		fmt.Println("端点: GET /healthz, GET /v1/models, GET /v1/status, POST /v1/chat, POST /v1/eval")
		return srv.ListenAndServe(serveAddr)
	},
}

// isLoopbackHost 判断监听地址是否回环（空 host 视为监听所有网卡，不算回环）。
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:8310", "HTTP 服务监听地址")
	serveCmd.Flags().StringVar(&serveToken, "token", "", "访问令牌（调用方需带 Authorization: Bearer <token>）")
	rootCmd.AddCommand(serveCmd)
}
