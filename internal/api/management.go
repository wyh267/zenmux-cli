package api

import "github.com/yinghaowu/zenmux-cli/internal/client"

// SubscriptionPlan 套餐信息。
type SubscriptionPlan struct {
	Tier      string  `json:"tier"`
	AmountUSD float64 `json:"amount_usd"`
	Interval  string  `json:"interval"`
	ExpiresAt string  `json:"expires_at"`
}

// Quota 滚动窗口配额。
type Quota struct {
	UsagePercentage float64 `json:"usage_percentage"`
	ResetsAt        string  `json:"resets_at"`
	MaxFlows        float64 `json:"max_flows"`
	UsedFlows       float64 `json:"used_flows"`
	RemainingFlows  float64 `json:"remaining_flows"`
	UsedValueUSD    float64 `json:"used_value_usd"`
	MaxValueUSD     float64 `json:"max_value_usd"`
}

// MonthlyQuota 月度配额（仅含上限）。
type MonthlyQuota struct {
	MaxFlows    float64 `json:"max_flows"`
	MaxValueUSD float64 `json:"max_value_usd"`
}

// SubscriptionDetail 订阅详情响应。
type SubscriptionDetail struct {
	Plan                SubscriptionPlan `json:"plan"`
	Currency            string           `json:"currency"`
	BaseUSDPerFlow      float64          `json:"base_usd_per_flow"`
	EffectiveUSDPerFlow float64          `json:"effective_usd_per_flow"`
	AccountStatus       string           `json:"account_status"`
	Quota5Hour          Quota            `json:"quota_5_hour"`
	Quota7Day           Quota            `json:"quota_7_day"`
	QuotaMonthly        MonthlyQuota     `json:"quota_monthly"`
}

// GetSubscription 查询订阅详情。
func GetSubscription(c *client.Client) (*SubscriptionDetail, error) {
	var wrapper struct {
		Success bool               `json:"success"`
		Data    SubscriptionDetail `json:"data"`
	}
	if err := c.Do("GET", "/management/subscription/detail", nil, true, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// PAYGBalance PAYG 余额响应。
type PAYGBalance struct {
	Currency      string  `json:"currency"`
	TotalCredits  float64 `json:"total_credits"`
	TopUpCredits  float64 `json:"top_up_credits"`
	BonusCredits  float64 `json:"bonus_credits"`
}

// GetPAYGBalance 查询 PAYG 余额。
func GetPAYGBalance(c *client.Client) (*PAYGBalance, error) {
	var wrapper struct {
		Success bool        `json:"success"`
		Data    PAYGBalance `json:"data"`
	}
	if err := c.Do("GET", "/management/payg/balance", nil, true, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// FlowRate Flow 汇率响应。
type FlowRate struct {
	Currency            string  `json:"currency"`
	BaseUSDPerFlow      float64 `json:"base_usd_per_flow"`
	EffectiveUSDPerFlow float64 `json:"effective_usd_per_flow"`
}

// GetFlowRate 查询 Flow 汇率。
func GetFlowRate(c *client.Client) (*FlowRate, error) {
	var wrapper struct {
		Success bool     `json:"success"`
		Data    FlowRate `json:"data"`
	}
	if err := c.Do("GET", "/management/flow_rate", nil, true, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}
