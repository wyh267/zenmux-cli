// Package client 封装 zenmux API 的 HTTP 通信：代理、鉴权、SSE 流式解析与统一错误处理。
package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIError 表示 zenmux 返回的标准错误结构 { "error": {code,type,message} }。
type APIError struct {
	Code    string `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("[%s] %s", e.Code, e.Message)
	}
	return e.Message
}

// Client zenmux API 客户端。
type Client struct {
	BaseURL  string // 例如 https://zenmux.ai/api/v1
	APIKey   string // 普通 API Key（调模型）
	MgmtKey  string // Management API Key（查套餐/用量）
	Proxy    string // HTTP/HTTPS 代理地址，空则不走代理
	httpc    *http.Client
}

// New 根据参数构造客户端，自动配置代理。
func New(baseURL, apiKey, mgmtKey, proxy string) *Client {
	transport := &http.Transport{}
	if proxy != "" {
		if u, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	} else {
		// 未显式配置代理时，尊重标准环境变量 HTTPS_PROXY/HTTP_PROXY。
		transport.Proxy = http.ProxyFromEnvironment
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		MgmtKey: mgmtKey,
		Proxy:   proxy,
		httpc: &http.Client{
			Transport: transport,
			Timeout:   10 * time.Minute, // 视频等长任务需要较长超时
		},
	}
}

// pickKey 根据 useMgmt 选择对应的 Key。
func (c *Client) pickKey(useMgmt bool) string {
	if useMgmt {
		return c.MgmtKey
	}
	return c.APIKey
}

// url 拼接完整请求地址。
func (c *Client) url(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path // 已经是绝对地址（如视频下载）
	}
	return c.BaseURL + path
}

// Do 发送 JSON 请求并把响应体解析到 out（out 为 nil 时丢弃）。
// useMgmt 为 true 时使用 Management Key 鉴权。
func (c *Client) Do(method, path string, body any, useMgmt bool, out any) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.url(path), bodyReader)
	if err != nil {
		return err
	}
	key := c.pickKey(useMgmt)
	if key == "" {
		return fmt.Errorf("缺少 API Key，请用 `zenmux config set` 配置（%s）", map[bool]string{true: "management_api_key", false: "api_key"}[useMgmt])
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 非 2xx 统一解析错误。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		// 尝试 { "error": {...} } 结构。
		var wrapper struct {
			Error APIError `json:"error"`
		}
		if json.Unmarshal(data, &wrapper) == nil && wrapper.Error.Message != "" {
			apiErr = wrapper.Error
		} else {
			apiErr = APIError{
				Code:    fmt.Sprintf("http_%d", resp.StatusCode),
				Message: strings.TrimSpace(string(data)),
			}
		}
		return &apiErr
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("解析响应 JSON 失败: %w", err)
		}
	}
	return nil
}

// SSEEvent 表示一个 Server-Sent Events 数据帧。
type SSEEvent struct {
	Raw []byte // 原始 data: 行内容（去掉前缀）
}

// StreamSSE 发送 POST 请求并以 SSE 方式逐帧回调 onEvent。
// 当收到 [DONE] 时停止。常用于 chat 流式输出。
func (c *Client) StreamSSE(path string, body any, onEvent func(data []byte) error) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, c.url(path), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	if c.APIKey == "" {
		return fmt.Errorf("缺少 API Key，请用 `zenmux config set api_key <key>` 配置")
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		var wrapper struct {
			Error APIError `json:"error"`
		}
		if json.Unmarshal(data, &wrapper) == nil && wrapper.Error.Message != "" {
			return &wrapper.Error
		}
		return &APIError{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: strings.TrimSpace(string(data))}
	}

	scanner := bufio.NewScanner(resp.Body)
	// 单行可能较大（如 base64 音频块），调大 buffer。
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // 空行分隔符
		}
		// SSE 规范：以 "data: " 开头。
		const prefix = "data: "
		if !bytes.HasPrefix(line, []byte(prefix)) {
			continue
		}
		payload := bytes.TrimPrefix(line, []byte(prefix))
		// [DONE] 标志流结束。
		if bytes.Equal(bytes.TrimSpace(payload), []byte("[DONE]")) {
			break
		}
		if err := onEvent(payload); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// DownloadFile 下载 url 到本地文件路径，返回写入字节数。
func (c *Client) DownloadFile(rawURL, dest string) (int64, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	if strings.HasPrefix(rawURL, c.BaseURL) {
		// 仅 zenmux 域内地址需要鉴权。
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return 0, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return 0, &APIError{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: strings.TrimSpace(string(data))}
	}
	out, err := openFile(dest)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	return io.Copy(out, resp.Body)
}
