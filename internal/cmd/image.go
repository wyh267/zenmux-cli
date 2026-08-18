package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	imageModel   string
	imageSize    string
	imageQuality string
	imageN       int
	imageFormat  string
	imageOutput  string
)

// imageCmd 图片生成。
var imageCmd = &cobra.Command{
	Use:   "image <prompt>",
	Short: "根据文本提示词生成图片",
	Args:  cobra.ExactArgs(1),
	Long: `调用图片生成模型，根据文字描述生成图片。

示例：
  zenmux image "一只在夕阳下的海滩上奔跑的金毛"
  zenmux image "赛博朋克城市夜景" -s 1536x1024 -q high -o cyber.png
  zenmux image "logo 设计" -m gpt-image-2 --format webp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model := imageModel
		if model == "" {
			model = cfg.Models.Image
		}

		req := api.ImageRequest{
			Model:        model,
			Prompt:       args[0],
			N:            imageN,
			Size:         imageSize,
			Quality:      imageQuality,
			OutputFormat: imageFormat,
		}

		c := newClient()
		resp, err := api.GenerateImage(c, req)
		if err != nil {
			return err
		}

		outDir := filepath.Dir(imageOutput)
		baseName := strings.TrimSuffix(imageOutput, filepath.Ext(imageOutput))

		for i, item := range resp.Data {
			// 优先 b64_json，其次 url。
			if item.B64JSON != "" {
				data, err := base64.StdEncoding.DecodeString(item.B64JSON)
				if err != nil {
					return fmt.Errorf("解码图片失败: %w", err)
				}
				path := outputPath(outDir, baseName, i, len(resp.Data), imageFormat)
				if err := os.WriteFile(path, data, 0o644); err != nil {
					return fmt.Errorf("写入文件失败: %w", err)
				}
				fmt.Printf("已生成: %s\n", path)
			} else if item.URL != "" {
				fmt.Printf("图片 URL: %s\n", item.URL)
			}
		}
		if resp.Usage.TotalTokens > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "[用量] input=%d output=%d total=%d tokens\n",
				resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
		}
		return nil
	},
}

// outputPath 生成第 i 张图片的输出路径。
func outputPath(dir, baseName string, i, total int, format string) string {
	ext := "." + format
	if ext == "." {
		ext = ".png"
	}
	if total == 1 {
		return filepath.Join(dir, baseName+ext)
	}
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", baseName, i+1, ext))
}

func init() {
	imageCmd.Flags().StringVarP(&imageModel, "model", "m", "", "图片模型 slug")
	imageCmd.Flags().StringVarP(&imageSize, "size", "s", "1024x1024", "图片尺寸")
	imageCmd.Flags().StringVarP(&imageQuality, "quality", "q", "auto", "质量 (low/medium/high/auto)")
	imageCmd.Flags().IntVarP(&imageN, "number", "n", 1, "生成数量 (1-10)")
	imageCmd.Flags().StringVar(&imageFormat, "format", "png", "输出格式 (png/jpeg/webp)")
	imageCmd.Flags().StringVarP(&imageOutput, "output", "o", "output", "输出文件名（不含扩展名）")
	rootCmd.AddCommand(imageCmd)
}
