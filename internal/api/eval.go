package api

import (
	"github.com/yinghaowu/zenmux-cli/internal/client"
)

// EvalQuestion 结构化评估的类型化问题。
// Type 取值为 noul（是/否判定）、choice（选项选择）、score（分级打分）。
type EvalQuestion struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

// EvalRequest 对应 POST /v1/systemone 的请求体。
// State 可以是纯文本 string，也可以是结构化数据（object / array）。
type EvalRequest struct {
	Model     string                  `json:"model"`
	State     any                     `json:"state"`
	Questions map[string]EvalQuestion `json:"questions"`
}

// EvalAnswer 单个问题的结构化答案，字段随 Type 不同而填充。
type EvalAnswer struct {
	Type          string             `json:"type"`                    // noul / choice / score
	Noul          *float64           `json:"noul,omitempty"`          // noul：为"是"的概率（0~1）
	Choice        string             `json:"choice,omitempty"`        // choice：概率最高的选项
	Score         *float64           `json:"score,omitempty"`         // score：跨分级的概率加权分值
	Probabilities map[string]float64 `json:"probabilities,omitempty"` // choice/score：概率分布
	Legend        map[string]string  `json:"legend,omitempty"`        // score：分级序号到描述的映射
	Confidence    *float64           `json:"confidence,omitempty"`    // choice/score：确信度（0~1）
}

// EvalResponse System One 接口的返回结构。
type EvalResponse struct {
	Model   string                `json:"model"`
	Answers map[string]EvalAnswer `json:"answers"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Eval 发起结构化评估（System One API，如 typesafe/jev-latest）。
func Eval(c *client.Client, req EvalRequest) (*EvalResponse, error) {
	var resp EvalResponse
	if err := c.Do("POST", "/systemone", req, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
