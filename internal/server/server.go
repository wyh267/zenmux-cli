// Package server 把 zenmux-cli 的 API 封装暴露为常驻 HTTP 网关服务。
//
// 设计要点：
//   - 完全复用 internal/api 与 internal/client，不修改任何 CLI 逻辑
//   - API Key 留在服务端配置，调用方仅需可选的访问令牌
//   - 仅使用标准库 net/http，无新增依赖
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/yinghaowu/zenmux-cli/internal/api"
	"github.com/yinghaowu/zenmux-cli/internal/client"
	"github.com/yinghaowu/zenmux-cli/internal/config"
)

// Server zenmux HTTP 网关。
type Server struct {
	c        *client.Client
	defaults config.Models // 各模态默认模型
	token    string        // 访问令牌，空表示不鉴权
	mux      *http.ServeMux
}

// New 构造网关服务。token 为空时不对调用方鉴权。
func New(c *client.Client, defaults config.Models, token string) *Server {
	s := &Server{c: c, defaults: defaults, token: token, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("GET /v1/status", s.handleStatus)
	s.mux.HandleFunc("POST /v1/chat", s.handleChat)
	s.mux.HandleFunc("POST /v1/eval", s.handleEval)
	return s
}

// ListenAndServe 在 addr 上启动 HTTP 服务。
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.auth(s.mux))
}

// auth 访问令牌鉴权中间件；令牌为空时放行。
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && r.URL.Path != "/healthz" {
			if r.Header.Get("Authorization") != "Bearer "+s.token {
				writeErr(w, http.StatusUnauthorized, "unauthorized", "缺少或错误的访问令牌")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// handleHealthz 健康检查。
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleModels 模型列表。
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	resp, err := api.ListModels(s.c)
	if err != nil {
		writeAPIErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleStatus 套餐状态：订阅详情 + PAYG 余额 + Flow 汇率合并返回。
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	sub, err := api.GetSubscription(s.c)
	if err != nil {
		writeAPIErr(w, err)
		return
	}
	balance, err := api.GetPAYGBalance(s.c)
	if err != nil {
		writeAPIErr(w, err)
		return
	}
	flow, err := api.GetFlowRate(s.c)
	if err != nil {
		writeAPIErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscription": sub,
		"payg_balance": balance,
		"flow_rate":    flow,
	})
}

// chatRequest POST /v1/chat 请求体。model 缺省时用服务端配置的默认模型。
type chatRequest struct {
	Model    string            `json:"model"`
	System   string            `json:"system"`
	Messages []api.ChatMessage `json:"messages"`
	Stream   bool              `json:"stream"`
}

// handleChat 对话。stream=true 时以 SSE 逐帧返回 {"content": "..."}，以 data: [DONE] 结束。
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "请求体不是合法 JSON: "+err.Error())
		return
	}
	if len(req.Messages) == 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "messages 不能为空")
		return
	}
	if req.Model == "" {
		req.Model = s.defaults.Chat
	}

	if !req.Stream {
		resp, err := api.Chat(s.c, api.ChatRequest{
			Model:    req.Model,
			System:   req.System,
			Messages: req.Messages,
		}, nil)
		if err != nil {
			writeAPIErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// 流式：SSE 转发增量文本。
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "stream_unsupported", "响应不支持流式输出")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	_, err := api.Chat(s.c, api.ChatRequest{
		Model:    req.Model,
		System:   req.System,
		Messages: req.Messages,
		Stream:   true,
	}, func(text string) {
		chunk, _ := json.Marshal(map[string]string{"content": text})
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
	})
	if err != nil {
		// 头已写出，错误以事件形式告知调用方。
		chunk, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", chunk)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// evalRequest POST /v1/eval 请求体。model 缺省时用服务端配置的默认评估模型。
type evalRequest struct {
	Model     string                      `json:"model"`
	State     any                         `json:"state"`
	Questions map[string]api.EvalQuestion `json:"questions"`
}

// handleEval 结构化评估（System One，如 typesafe/jev-latest）。
func (s *Server) handleEval(w http.ResponseWriter, r *http.Request) {
	var req evalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "请求体不是合法 JSON: "+err.Error())
		return
	}
	if req.State == nil || len(req.Questions) == 0 {
		writeErr(w, http.StatusBadRequest, "bad_request", "state 与 questions 均不能为空")
		return
	}
	if req.Model == "" {
		req.Model = s.defaults.Eval
	}
	resp, err := api.Eval(s.c, api.EvalRequest{
		Model:     req.Model,
		State:     req.State,
		Questions: req.Questions,
	})
	if err != nil {
		writeAPIErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeErr 写统一错误结构。
func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// writeAPIErr 把 zenmux 的 APIError 转成 HTTP 响应；
// Code 形如 "http_400" 时透传对应状态码，否则按 502 网关错误处理。
func writeAPIErr(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(*client.APIError); ok {
		status := http.StatusBadGateway
		if strings.HasPrefix(apiErr.Code, "http_") {
			var n int
			if _, err := fmt.Sscanf(apiErr.Code, "http_%d", &n); err == nil && n >= 400 && n < 600 {
				status = n
			}
		}
		writeErr(w, status, apiErr.Code, apiErr.Message)
		return
	}
	writeErr(w, http.StatusInternalServerError, "internal", err.Error())
}
