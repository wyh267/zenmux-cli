package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yinghaowu/zenmux-cli/internal/client"
)

// VideoContentItem 视频生成的 content 数组项。
type VideoContentItem struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	Role     string         `json:"role,omitempty"`
	ImageURL *MediaRef      `json:"image_url,omitempty"`
	VideoURL *MediaRef      `json:"video_url,omitempty"`
	AudioURL *MediaRef      `json:"audio_url,omitempty"`
}

// MediaRef 媒体引用容器 { url: "..." }。
type MediaRef struct {
	URL string `json:"url"`
}

// VideoRequest 对应 POST /v1/videos 的请求体。
type VideoRequest struct {
	Model         string             `json:"model"`
	Content       []VideoContentItem `json:"content"`
	Resolution    string             `json:"resolution,omitempty"`
	Ratio         string             `json:"ratio,omitempty"`
	Duration      int                `json:"duration,omitempty"`
	Seed          int                `json:"seed,omitempty"`
	GenerateAudio bool               `json:"generate_audio,omitempty"`
}

// VideoTask 视频生成任务。
type VideoTask struct {
	ID      string `json:"id"`
	Status  string `json:"status"` // queued | running | succeeded | failed
	Model   string `json:"model"`
	Content struct {
		VideoURL     string `json:"video_url"`
		LastFrameURL string `json:"last_frame_url"`
	} `json:"content"`
	Error *client.APIError `json:"error,omitempty"`
}

// SubmitVideo 提交视频生成任务，返回初始任务对象。
func SubmitVideo(c *client.Client, req VideoRequest) (*VideoTask, error) {
	var task VideoTask
	if err := c.Do("POST", "/videos", req, false, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetVideo 查询视频任务状态。
func GetVideo(c *client.Client, jobID string) (*VideoTask, error) {
	var task VideoTask
	if err := c.Do("GET", "/videos/"+jobID, nil, false, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// PollVideo 轮询任务直到完成，interval 为轮询间隔。
// onStatus 在每次状态变化时回调（status 描述）。
// 返回最终任务对象。失败或超时返回错误。
func PollVideo(c *client.Client, jobID string, interval time.Duration, onStatus func(task *VideoTask)) (*VideoTask, error) {
	lastStatus := ""
	for {
		task, err := GetVideo(c, jobID)
		if err != nil {
			return nil, err
		}
		if task.Status != lastStatus {
			lastStatus = task.Status
			if onStatus != nil {
				onStatus(task)
			}
		}
		switch task.Status {
		case "succeeded":
			return task, nil
		case "failed":
			if task.Error != nil {
				return nil, fmt.Errorf("视频生成失败: %w", task.Error)
			}
			return nil, fmt.Errorf("视频生成失败")
		}
		time.Sleep(interval)
	}
}

// BuildVideoContent 根据 prompt 和可选的首帧图路径构造 content 数组。
// 若 imagePath 非空，读取文件并转为 data URI 作为 first_frame。
func BuildVideoContent(prompt, imagePath string) ([]VideoContentItem, error) {
	items := []VideoContentItem{{Type: "text", Text: prompt}}
	if imagePath == "" {
		return items, nil
	}
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("读取首帧图片失败: %w", err)
	}
	mime := mimeFromExt(filepath.Ext(imagePath))
	uri := fmt.Sprintf("data:%s;base64,%s", mime, base64Encode(data))
	items = append(items, VideoContentItem{
		Type:     "image_url",
		Role:     "first_frame",
		ImageURL: &MediaRef{URL: uri},
	})
	return items, nil
}

// mimeFromExt 根据扩展名推断图片 MIME 类型。
func mimeFromExt(ext string) string {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	default:
		return "image/png"
	}
}
