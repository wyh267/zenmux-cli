package api

import "github.com/yinghaowu/zenmux-cli/internal/client"

// ImageRequest 对应 POST /v1/images/generations 的请求体。
type ImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
	Size   string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// ImageResponse 图片生成返回结构。
type ImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json"`
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Usage struct {
		TotalTokens  int `json:"total_tokens"`
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GenerateImage 生成图片。
func GenerateImage(c *client.Client, req ImageRequest) (*ImageResponse, error) {
	var resp ImageResponse
	if err := c.Do("POST", "/images/generations", req, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
