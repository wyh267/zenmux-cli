package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	evalModel     string
	evalQuestions string
	evalStateFile string
	evalJSON      bool
)

// evalCmd 结构化评估（System One，如 TypeSafe jev 模型）。
var evalCmd = &cobra.Command{
	Use:   "eval <待评估文本>",
	Short: "结构化评估：对内容做是/否判定、选项分类、分级打分",
	Args:  cobra.MaximumNArgs(1),
	Long: `使用结构化评估模型（如 typesafe/jev-latest）对一段内容进行评估。

通过 --questions 指定问题定义（JSON 文件，"-" 表示从标准输入读取），
每个问题是三种类型之一：
  noul    是/否判定，返回为"是"的概率
  choice  从自定义选项中选择，返回选中项及概率分布
  score   按自定义分级打分，返回概率加权分值

待评估内容默认是命令行文本参数；结构化数据（JSON object/array）用 --state-file 传入。

示例：
  zenmux eval "救命！我的账户回款已经连续 3 天失败了。" -q questions.json
  cat questions.json | zenmux eval "用户对话记录……" -q -
  zenmux eval --state-file dialog.json -q questions.json
  zenmux eval "这条评论很满意" -q q.json --json    # 输出原始 JSON`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 组装 state：位置参数（纯文本）或 --state-file（JSON 结构化数据）。
		var state any
		if evalStateFile != "" {
			data, err := readInput(evalStateFile)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, &state); err != nil {
				return fmt.Errorf("--state-file 内容不是合法 JSON: %w", err)
			}
		} else if len(args) == 1 {
			state = args[0]
		} else {
			return fmt.Errorf("请提供待评估文本参数，或用 --state-file 指定 JSON 文件")
		}

		if evalQuestions == "" {
			return fmt.Errorf("请用 -q/--questions 指定问题定义 JSON 文件（\"-\" 表示标准输入）")
		}
		qData, err := readInput(evalQuestions)
		if err != nil {
			return err
		}
		var questions map[string]api.EvalQuestion
		if err := json.Unmarshal(qData, &questions); err != nil {
			return fmt.Errorf("解析问题定义失败: %w", err)
		}
		if len(questions) == 0 {
			return fmt.Errorf("问题定义为空")
		}
		for key, q := range questions {
			switch q.Type {
			case "noul", "choice", "score":
			default:
				return fmt.Errorf("问题 %q 的 type 无效: %q（应为 noul/choice/score）", key, q.Type)
			}
		}

		model := evalModel
		if model == "" {
			model = cfg.Models.Eval
		}

		resp, err := api.Eval(newClient(), api.EvalRequest{
			Model:     model,
			State:     state,
			Questions: questions,
		})
		if err != nil {
			return err
		}

		if evalJSON {
			data, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			printEvalAnswers(resp)
		}
		if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "\n[用量] input=%d output=%d\n", resp.Usage.InputTokens, resp.Usage.OutputTokens)
		}
		return nil
	},
}

// readInput 读取文件内容；path 为 "-" 时读取标准输入。
func readInput(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件 %s 失败: %w", path, err)
	}
	return data, nil
}

// printEvalAnswers 以人类可读形式打印评估结果（按问题 key 排序，输出稳定）。
func printEvalAnswers(resp *api.EvalResponse) {
	keys := make([]string, 0, len(resp.Answers))
	for k := range resp.Answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		a := resp.Answers[key]
		switch a.Type {
		case "noul":
			fmt.Printf("%s [noul]: %s\n", key, formatFloat(a.Noul))
		case "choice":
			fmt.Printf("%s [choice]: %s（置信度 %s）\n", key, a.Choice, formatFloat(a.Confidence))
			fmt.Printf("  概率分布: %s\n", formatProbs(a.Probabilities))
		case "score":
			fmt.Printf("%s [score]: %s（置信度 %s）\n", key, formatFloat(a.Score), formatFloat(a.Confidence))
			if len(a.Legend) > 0 {
				fmt.Printf("  分级: %s\n", formatLegend(a.Legend))
			}
			fmt.Printf("  概率分布: %s\n", formatProbs(a.Probabilities))
		default:
			fmt.Printf("%s [%s]: %+v\n", key, a.Type, a)
		}
	}
}

// formatFloat 格式化可选浮点数，保留两位小数。
func formatFloat(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *v)
}

// formatProbs 按 key 排序格式化概率分布。
func formatProbs(probs map[string]float64) string {
	keys := make([]string, 0, len(probs))
	for k := range probs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%.2f", k, probs[k]))
	}
	return strings.Join(parts, " ")
}

// formatLegend 按分级序号排序格式化 score 的 legend。
func formatLegend(legend map[string]string) string {
	keys := make([]string, 0, len(legend))
	for k := range legend {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, legend[k]))
	}
	return strings.Join(parts, " ")
}

func init() {
	evalCmd.Flags().StringVarP(&evalModel, "model", "m", "", "模型 slug（默认用配置中的 default_model.eval）")
	evalCmd.Flags().StringVarP(&evalQuestions, "questions", "q", "", "问题定义 JSON 文件路径，\"-\" 表示标准输入")
	evalCmd.Flags().StringVar(&evalStateFile, "state-file", "", "待评估内容所在的 JSON 文件（用于结构化 state）")
	evalCmd.Flags().BoolVar(&evalJSON, "json", false, "输出原始 JSON 响应")
	rootCmd.AddCommand(evalCmd)
}
