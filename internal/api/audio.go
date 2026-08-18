package api

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yinghaowu/zenmux-cli/internal/client"
)

// base64Encode 将字节编码为标准 base64 字符串。
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// --- 文本转语音 (TTS) ---

// TTSRequest 对应 POST /v1/audio/speech 的请求体。
type TTSRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Speed          int    `json:"speed,omitempty"`
}

// TTSResponse 文本转语音返回结构。
type TTSResponse struct {
	Audio   string `json:"audio"`
	MimeType string `json:"mime_type"`
	Model   string `json:"model"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateSpeech 文本转语音。
func CreateSpeech(c *client.Client, req TTSRequest) (*TTSResponse, error) {
	var resp TTSResponse
	if err := c.Do("POST", "/audio/speech", req, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- 语音转文本 (STT) ---

// STTRequest 对应 POST /v1/audio/transcriptions 的请求体。
type STTRequest struct {
	Model      string        `json:"model"`
	InputAudio InputAudio    `json:"input_audio"`
	Language   string        `json:"language,omitempty"`
}

// InputAudio 音频输入，data 与 url 二选一。
type InputAudio struct {
	Data   string `json:"data,omitempty"`
	URL    string `json:"url,omitempty"`
	Format string `json:"format,omitempty"`
}

// STTResponse 语音转文本返回结构。
type STTResponse struct {
	Text  string `json:"text"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
		Seconds      int `json:"seconds"`
	} `json:"usage"`
}

// CreateTranscription 语音转文本。从本地文件读取并编码为 base64。
func CreateTranscription(c *client.Client, model, audioPath, language string) (*STTResponse, error) {
	data, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("读取音频文件失败: %w", err)
	}
	req := STTRequest{
		Model:      model,
		Language:   language,
		InputAudio: InputAudio{
			Data:   base64Encode(data),
			Format: audioFormatFromExt(filepath.Ext(audioPath)),
		},
	}
	var resp STTResponse
	if err := c.Do("POST", "/audio/transcriptions", req, false, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// audioFormatFromExt 根据扩展名推断音频容器格式。
func audioFormatFromExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "wav":
		return "wav"
	case "mp3":
		return "mp3"
	case "flac":
		return "flac"
	case "m4a":
		return "m4a"
	case "ogg":
		return "ogg"
	case "webm":
		return "webm"
	case "aac":
		return "aac"
	default:
		return "wav"
	}
}
