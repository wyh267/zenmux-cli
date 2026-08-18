package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

// statusCmd 查询套餐状态。
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询套餐状态、余额与 Flow 汇率",
	Long: `查询当前账号的订阅详情、PAYG 余额和 Flow 汇率。

需要先配置 Management API Key：
  zenmux config set management_api_key <你的Management Key>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()

		// 三个 management 接口分别尝试，单个失败不阻断其它展示。
		sub, subErr := api.GetSubscription(c)
		bal, balErr := api.GetPAYGBalance(c)
		rate, rateErr := api.GetFlowRate(c)

		// 至少要有一个成功。
		if subErr != nil && balErr != nil && rateErr != nil {
			return fmt.Errorf("查询失败（订阅: %v / 余额: %v / 汇率: %v）\n请确认已配置 management_api_key", subErr, balErr, rateErr)
		}

		fmt.Println("════════════════════════════════════════════")
		fmt.Println("  ZenMux 套餐状态")
		fmt.Println("════════════════════════════════════════════")

		// 订阅信息
		if subErr != nil {
			fmt.Printf("\n❌ 订阅信息查询失败: %v\n\n", subErr)
		} else {
			fmt.Printf("\n【订阅套餐】\n")
			fmt.Printf("  等级:     %s  ($%g/%s)\n", sub.Plan.Tier, sub.Plan.AmountUSD, sub.Plan.Interval)
			if sub.Plan.ExpiresAt != "" {
				fmt.Printf("  到期时间: %s\n", formatTime(sub.Plan.ExpiresAt))
			}
			fmt.Printf("  账号状态: %s\n", statusLabel(sub.AccountStatus))
			fmt.Printf("  Flow 汇率: $%.4f / Flow\n", sub.EffectiveUSDPerFlow)

			fmt.Printf("\n【用量配额】\n")
			printQuota("5 小时", sub.Quota5Hour)
			printQuota("7 天  ", sub.Quota7Day)
			if sub.QuotaMonthly.MaxFlows > 0 {
				fmt.Printf("  当月总额: %.0f Flows ($%.2f)\n", sub.QuotaMonthly.MaxFlows, sub.QuotaMonthly.MaxValueUSD)
			}
		}

		// PAYG 余额
		if balErr != nil {
			fmt.Printf("\n❌ PAYG 余额查询失败: %v\n", balErr)
		} else {
			fmt.Printf("\n【PAYG 按量付费余额】\n")
			fmt.Printf("  总余额:   $%.2f\n", bal.TotalCredits)
			fmt.Printf("  充值余额: $%.2f\n", bal.TopUpCredits)
			fmt.Printf("  赠送余额: $%.2f\n", bal.BonusCredits)
		}

		// Flow 汇率（独立接口）
		if rateErr == nil && rate != nil {
			fmt.Printf("\n【Flow 汇率】\n")
			fmt.Printf("  基准汇率: $%.4f / Flow\n", rate.BaseUSDPerFlow)
			fmt.Printf("  实际汇率: $%.4f / Flow\n", rate.EffectiveUSDPerFlow)
		}

		fmt.Println("\n════════════════════════════════════════════")
		return nil
	},
}

// printQuota 打印一个滚动窗口配额。
func printQuota(name string, q api.Quota) {
	pct := q.UsagePercentage * 100
	fmt.Printf("  %s窗口: %.1f / %.0f Flows (%.1f%%)", name, q.UsedFlows, q.MaxFlows, pct)
	if q.RemainingFlows > 0 {
		fmt.Printf("，剩余 %.1f", q.RemainingFlows)
	}
	if q.ResetsAt != "" {
		fmt.Printf("，重置于 %s", formatTime(q.ResetsAt))
	}
	fmt.Println()
	if q.MaxValueUSD > 0 {
		fmt.Printf("           已用 $%.2f / $%.2f\n", q.UsedValueUSD, q.MaxValueUSD)
	}
}

// statusLabel 把英文账号状态翻译成中文 + 状态符号。
func statusLabel(s string) string {
	switch s {
	case "healthy":
		return "✅ 正常 (healthy)"
	case "monitored":
		return "⚠️ 监控中 (monitored)"
	case "abusive":
		return "⚠️ 异常用量 (abusive)"
	case "suspended":
		return "⛔ 已暂停 (suspended)"
	case "banned":
		return "⛔ 已封禁 (banned)"
	default:
		return s
	}
}

// formatTime 把 ISO 8601 时间格式化成本地可读形式。
func formatTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
