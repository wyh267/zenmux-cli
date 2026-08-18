package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yinghaowu/zenmux-cli/internal/api"
)

var (
	ttsModel  string
	ttsVoice  string
	ttsFormat string
	ttsOutput string
	sttModel  string
	sttLanguage string
)

// voiceCmd 语音相关命令。
//   zenmux voice "文本"          → 文本转语音 (TTS)，默认行为
//   zenmux voice transcribe <f>  → 语音转文本 (STT)，子命令
var voiceCmd = &cobra.Command{
	Use:   "voice [text]",
	Short: "语音合成 (TTS)；用 `voice transcribe` 做语音识别 (STT)",
	Long: `语音命令。默认行为是文本转语音（TTS）；转写文本用 transcribe 子命令。

文本转语音：
  zenmux voice "你好，欢迎使用 zenmux"
  zenmux voice "Hello world" --voice Kore -o hello.pcm

语音转文本：
  zenmux voice transcribe speech.wav
  zenmux voice transcribe recording.mp3 --language zh`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 默认行为：TTS。参数整体作为要合成的文本。
		return runTTS(strings.Join(args, " "))
	},
}

// runTTS 执行文本转语音。
func runTTS(text string) error {
	model := ttsModel
	if model == "" {
		model = cfg.Models.TTS
	}
	req := api.TTSRequest{
		Model:          model,
		Input:          text,
		Voice:          ttsVoice,
		ResponseFormat: ttsFormat,
	}
	c := newClient()
	resp, err := api.CreateSpeech(c, req)
	if err != nil {
		return err
	}

	data, err := base64.StdEncoding.DecodeString(resp.Audio)
	if err != nil {
		return fmt.Errorf("解码音频失败: %w", err)
	}

	ext := ttsFormat
	if ext == "" {
		ext = "pcm"
	}
	out := ttsOutput
	if out == "" {
		out = "output." + ext
	} else if !strings.HasSuffix(out, "."+ext) {
		out = out + "." + ext
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", out)
	if resp.Usage.TotalTokens > 0 {
		fmt.Fprintf(os.Stderr, "[用量] input=%d output=%d total=%d tokens\n",
			resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
	}
	return nil
}

// voiceTranscribeCmd 语音转文本。
var voiceTranscribeCmd = &cobra.Command{
	Use:   "transcribe <audio-file>",
	Short: "语音转文本（Speech-to-Text）",
	Args:  cobra.ExactArgs(1),
	Long: `将音频文件转写为文本。

示例：
  zenmux voice transcribe speech.wav
  zenmux voice transcribe recording.mp3 --language zh
  zenmux voice transcribe voice.m4a -m qwen/qwen3-asr-flash`,
	RunE: func(cmd *cobra.Command, args []string) error {
		audioPath := args[0]
		if _, err := os.Stat(audioPath); err != nil {
			return fmt.Errorf("音频文件不存在: %w", err)
		}
		model := sttModel
		if model == "" {
			model = cfg.Models.STT
		}
		c := newClient()
		resp, err := api.CreateTranscription(c, model, audioPath, sttLanguage)
		if err != nil {
			return err
		}
		fmt.Println(resp.Text)
		if resp.Usage.TotalTokens > 0 {
			fmt.Fprintf(os.Stderr, "\n[用量] input=%d output=%d total=%d tokens",
				resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.Usage.TotalTokens)
			if resp.Usage.Seconds > 0 {
				fmt.Fprintf(os.Stderr, "（音频 %d 秒）", resp.Usage.Seconds)
			}
			fmt.Fprintln(os.Stderr)
		}
		return nil
	},
}

func init() {
	voiceCmd.Flags().StringVarP(&ttsModel, "model", "m", "", "TTS 模型 slug")
	voiceCmd.Flags().StringVar(&ttsVoice, "voice", "", "音色标识符（因模型而异）")
	voiceCmd.Flags().StringVar(&ttsFormat, "format", "pcm", "音频编码格式")
	voiceCmd.Flags().StringVarP(&ttsOutput, "output", "o", "", "输出文件名")

	// 注意：tts 的 flag 也挂载在 voiceCmd 上，transcribe 需要自己的 flag。
	// 但由于 transcribe 是 voiceCmd 的子命令，它的 -m/--language 会覆盖。
	voiceTranscribeCmd.Flags().StringVarP(&sttModel, "model", "m", "", "STT 模型 slug")
	voiceTranscribeCmd.Flags().StringVar(&sttLanguage, "language", "", "音频语言 (ISO-639-1，如 zh/en)")

	voiceCmd.AddCommand(voiceTranscribeCmd)
	rootCmd.AddCommand(voiceCmd)
}
