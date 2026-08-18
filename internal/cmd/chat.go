package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	chatModel   string
	chatSystem  string
	chatNoStream bool
)

// chatCmd 文本对话（单次问答）。
var chatCmd = &cobra.Command{
	Use:   "chat <prompt>",
	Short: "与大语言模型对话（单次问答，默认流式输出）",
	Args:  cobra.ExactArgs(1),
	Long: `向大模型发送一条消息并输出回复。默认采用流式输出（逐字打印）。

示例：
  zenmux chat "用 Go 实现一个快速排序"
  zenmux chat "翻译这句话" -m openai/gpt-5
  zenmux chat "分析这段代码" --system "你是一位资深代码审查员"
  zenmux chat "详细解释一下" --no-stream    # 整段输出，不流式`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model := chatModel
		if model == "" {
			model = cfg.Models.Chat
		}

		req := api.ChatRequest{
			Model:    model,
			Stream:   !chatNoStream,
			System:   chatSystem,
			Messages: []api.ChatMessage{{Role: "user", Content: args[0]}},
		}

		c := newClient()
		if !chatNoStream {
			// 流式：实时打印增量。
			resp, err := api.Chat(c, req, func(text string) {
				fmt.Print(text)
			})
			if err != nil {
				return err
			}
			fmt.Println()
			if resp.Usage.TotalTokens > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "\n[用量] %s\n", resp.UsageString())
			}
			return nil
		}

		// 非流式：整段输出。
		resp, err := api.Chat(c, req, nil)
		if err != nil {
			return err
		}
		fmt.Println(resp.String())
		if resp.Usage.TotalTokens > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "\n[用量] %s\n", resp.UsageString())
		}
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVarP(&chatModel, "model", "m", "", "模型 slug（默认用配置中的 default_model.chat）")
	chatCmd.Flags().StringVar(&chatSystem, "system", "", "系统提示词，设定模型角色")
	chatCmd.Flags().BoolVar(&chatNoStream, "no-stream", false, "禁用流式输出，整段返回")
	rootCmd.AddCommand(chatCmd)
}
