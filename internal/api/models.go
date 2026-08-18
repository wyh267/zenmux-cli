// Package api 封装 zenmux 各业务接口的类型定义与调用。
package api

import "github.com/yinghaowu/zenmux-cli/internal/client"

// Model 描述 /v1/models 返回的单个模型条目。
type Model struct {
	ID               string   `json:"id"`
	OwnedBy          string   `json:"owned_by"`
	DisplayName      string   `json:"display_name"`
	ContextLength    int      `json:"context_length"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Capabilities     struct {
		Reasoning bool `json:"reasoning"`
	} `json:"capabilities"`
	Created int64 `json:"created"`
}

// ModelsResponse /v1/models 返回结构。
type ModelsResponse struct {
	Data   []Model `json:"data"`
	Object string  `json:"object"`
}

// ListModels 获取平台支持的模型列表。
func ListModels(c *client.Client) (*ModelsResponse, error) {
	var resp ModelsResponse
	if err := c.Do("GET", "/models", nil, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
