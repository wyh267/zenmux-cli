package api

import (
	"encoding/json"
	"fmt"

	"github.com/yinghaowu/zenmux-cli/internal/client"
)

// ChatMessage 单条对话消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string 或 []ContentPart
}

// ChatRequest 对应 POST /v1/chat/completions 的请求体。
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
	System   string        `json:"-"` // 仅本地使用，在调用前合并进 messages
}

// ChatResponse 非流式返回结构。
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Choices []struct {
		Index        int `json:"index"`
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// chatStreamDelta 流式分块结构（仅解析需要的字段）。
type chatStreamDelta struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason any `json:"finish_reason"`
	} `json:"choices"`
}

// Chat 发起对话。stream=true 时通过 onDelta 回调增量文本；stream=false 时返回完整回复。
// 当 stream=true，返回的 ChatResponse 仅填充了 Model/Usage（如服务端提供）。
func Chat(c *client.Client, req ChatRequest, onDelta func(text string)) (*ChatResponse, error) {
	// 合并 system 提示词。
	if req.System != "" {
		req.Messages = append([]ChatMessage{{Role: "system", Content: req.System}}, req.Messages...)
	}

	if !req.Stream {
		var resp ChatResponse
		if err := c.Do("POST", "/chat/completions", req, false, &resp); err != nil {
			return nil, err
		}
		return &resp, nil
	}

	// 流式：手动解析 SSE。
	var result ChatResponse
	err := c.StreamSSE("/chat/completions", req, func(data []byte) error {
		var delta chatStreamDelta
		if err := json.Unmarshal(data, &delta); err != nil {
			return nil // 跳过无法解析的帧
		}
		if len(delta.Choices) > 0 && delta.Choices[0].Delta.Content != "" {
			if onDelta != nil {
				onDelta(delta.Choices[0].Delta.Content)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// String 便捷获取非流式响应的文本内容。
func (r *ChatResponse) String() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

// UsageString 便捷获取用量摘要。
func (r *ChatResponse) UsageString() string {
	return fmt.Sprintf("prompt=%d completion=%d total=%d", r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens)
}
