package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	videoModel      string
	videoResolution string
	videoRatio      string
	videoDuration   int
	videoImage      string
	videoOutput     string
	videoAudio      bool
)

// videoCmd 视频生成（异步轮询）。
var videoCmd = &cobra.Command{
	Use:   "video <prompt>",
	Short: "根据文本提示词生成视频（异步）",
	Args:  cobra.ExactArgs(1),
	Long: `调用视频生成模型。视频生成是异步的，提交后会每 15 秒轮询一次状态，
生成完成（通常 30 秒到 3 分钟）后自动下载 mp4 文件。

示例：
  zenmux video "一只金毛在夕阳下的海滩上奔跑"
  zenmux video "猫咪在暖光下读书" --resolution 1080p --ratio 16:9 --duration 8
  zenmux video "小狗站起来跑向海浪" --image first_frame.png -o my_video
  zenmux video "歌手在舞台上表演" --audio   # 生成带音频的视频`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model := videoModel
		if model == "" {
			model = cfg.Models.Video
		}

		content, err := api.BuildVideoContent(args[0], videoImage)
		if err != nil {
			return err
		}

		req := api.VideoRequest{
			Model:         model,
			Content:       content,
			Resolution:    videoResolution,
			Ratio:         videoRatio,
			Duration:      videoDuration,
			GenerateAudio: videoAudio,
		}

		c := newClient()

		// Step 1: 提交任务。
		fmt.Printf("提交视频生成任务（模型 %s）...\n", model)
		task, err := api.SubmitVideo(c, req)
		if err != nil {
			return err
		}
		fmt.Printf("任务已提交，ID: %s，状态: %s\n", task.ID, task.Status)

		// Step 2: 轮询。
		start := time.Now()
		finalTask, err := api.PollVideo(c, task.ID, 15*time.Second, func(t *api.VideoTask) {
			fmt.Printf("[%s] 状态: %s（已等待 %s）\n", time.Now().Format("15:04:05"), t.Status, time.Since(start).Truncate(time.Second))
		})
		if err != nil {
			return err
		}

		if finalTask.Content.VideoURL == "" {
			return fmt.Errorf("任务完成但未返回视频地址")
		}

		// Step 3: 下载。
		out := videoOutput
		if out == "" {
			out = "output"
		}
		dest := out + ".mp4"
		fmt.Printf("下载视频到 %s ...\n", dest)
		n, err := c.DownloadFile(finalTask.Content.VideoURL, dest)
		if err != nil {
			return err
		}
		fmt.Printf("完成！已保存到 %s（%.2f MB，耗时 %s）\n",
			dest, float64(n)/1024/1024, time.Since(start).Truncate(time.Second))
		return nil
	},
}

func init() {
	videoCmd.Flags().StringVarP(&videoModel, "model", "m", "", "视频模型 slug")
	videoCmd.Flags().StringVar(&videoResolution, "resolution", "", "分辨率 (480p/720p/1080p)")
	videoCmd.Flags().StringVar(&videoRatio, "ratio", "", "宽高比 (16:9/9:16/1:1/adaptive)")
	videoCmd.Flags().IntVar(&videoDuration, "duration", 0, "时长（秒）")
	videoCmd.Flags().StringVar(&videoImage, "image", "", "首帧图片路径（图生视频）")
	videoCmd.Flags().StringVarP(&videoOutput, "output", "o", "output", "输出文件名（不含扩展名）")
	videoCmd.Flags().BoolVar(&videoAudio, "audio", false, "是否生成音频轨道")
	rootCmd.AddCommand(videoCmd)
}
