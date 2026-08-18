package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	modelsModality string // --modality 过滤输出模态
	modelsProvider string // --provider 过滤供应商
	modelsSearch   string // --search 关键字
	modelsJSON     bool   // --json 输出原始 JSON
)

// modelsCmd 列出所有支持的模型。
var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "列出所有支持的模型及其模态",
	Long: `从 zenmux 平台拉取所有可用模型，展示其输入/输出模态、上下文长度、能力等。

过滤示例：
  zenmux models --modality image          # 列出能生成图片的模型
  zenmux models --provider openai         # 只看 OpenAI 的模型
  zenmux models --search gpt-5            # 关键字搜索`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		resp, err := api.ListModels(c)
		if err != nil {
			return err
		}

		filtered := filterModels(resp.Data)

		if modelsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(filtered)
		}

		if len(filtered) == 0 {
			fmt.Println("没有匹配的模型。")
			return nil
		}

		printModelsTable(filtered)
		fmt.Printf("\n共 %d 个模型（全平台 %d 个）。\n", len(filtered), len(resp.Data))
		return nil
	},
}

// filterModels 按供应商/模态/关键字过滤。
func filterModels(models []api.Model) []api.Model {
	var out []api.Model
	for _, m := range models {
		if modelsProvider != "" && m.OwnedBy != modelsProvider {
			continue
		}
		if modelsModality != "" && !hasModality(m.OutputModalities, modelsModality) {
			continue
		}
		if modelsSearch != "" && !strings.Contains(strings.ToLower(m.ID), strings.ToLower(modelsSearch)) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// hasModality 判断模型是否支持指定模态。
func hasModality(modalities []string, target string) bool {
	for _, m := range modalities {
		if m == target {
			return true
		}
	}
	return false
}

// printModelsTable 以表格形式打印模型列表。
func printModelsTable(models []api.Model) {
	fmt.Printf("%-40s %-14s %-18s %-18s %-10s\n", "MODEL", "PROVIDER", "INPUT", "OUTPUT", "REASONING")
	fmt.Println(strings.Repeat("-", 104))
	for _, m := range models {
		reasoning := "-"
		if m.Capabilities.Reasoning {
			reasoning = "✓"
		}
		fmt.Printf("%-40s %-14s %-18s %-18s %-10s\n",
			truncate(m.ID, 40),
			m.OwnedBy,
			joinMod(m.InputModalities),
			joinMod(m.OutputModalities),
			reasoning,
		)
	}
}

// joinMod 用 + 连接模态列表。
func joinMod(mods []string) string {
	if len(mods) == 0 {
		return "-"
	}
	return strings.Join(mods, "+")
}

// truncate 截断字符串到指定长度。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func init() {
	modelsCmd.Flags().StringVar(&modelsModality, "modality", "", "按输出模态过滤 (text/image/video/audio)")
	modelsCmd.Flags().StringVar(&modelsProvider, "provider", "", "按供应商过滤 (如 openai/anthropic/google)")
	modelsCmd.Flags().StringVar(&modelsSearch, "search", "", "按模型 ID 关键字搜索")
	modelsCmd.Flags().BoolVar(&modelsJSON, "json", false, "以 JSON 格式输出")
	rootCmd.AddCommand(modelsCmd)
}
