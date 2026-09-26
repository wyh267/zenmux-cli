# zenmux-cli

[zenmux.ai](https://zenmux.ai) 大模型聚合服务的命令行客户端。

用一个二进制调用 100+ 个大模型：文本对话、图片生成、视频生成、语音合成与识别，以及套餐状态查询、模型列表查询。

## 功能

| 命令 | 说明 |
| --- | --- |
| `zenmux chat` | 文本对话（默认流式输出） |
| `zenmux image` | 文生图 |
| `zenmux video` | 文生视频 / 图生视频（异步轮询，自动下载） |
| `zenmux voice` | 文本转语音 (TTS) |
| `zenmux voice transcribe` | 语音转文本 (STT) |
| `zenmux eval` | 结构化评估：是/否判定、选项分类、分级打分 |
| `zenmux models` | 列出所有支持的模型及其模态 |
| `zenmux status` | 查询套餐状态、余额、Flow 汇率 |
| `zenmux config` | 查看与修改配置 |

## 安装

需要 Go 1.21+。

```bash
git clone <repo-url> && cd zenmux-cli
go build -o zenmux .
# 可选：安装到 PATH
sudo mv zenmux /usr/local/bin/
```

## 快速开始

### 1. 配置

首次使用需配置 API Key。zenmux 区分两种 Key：

- **API Key**：调用模型（chat/image/video/voice）
- **Management API Key**：查询套餐/用量（在 [控制台 Management 页面](https://zenmux.ai/platform/management) 创建）

```bash
zenmux config set api_key sk-xxxxxxxxxxxxxxxx
zenmux config set management_api_key sk-mg-xxxxxxxxxxxxxx
# 国内需要代理
zenmux config set proxy http://127.0.0.1:7897

# 检查配置（Key 自动脱敏）
zenmux config show
```

配置文件位于 `~/.config/zenmux/config.yaml`（遵循 XDG）。

> 也可以用环境变量：`ZENMUX_API_KEY`、`ZENMUX_MANAGEMENT_API_KEY`、`ZENMUX_PROXY`、`ZENMUX_BASE_URL`。

### 2. 列出模型

```bash
zenmux models                       # 全部 151+ 个模型
zenmux models --modality image      # 能生成图片的模型
zenmux models --provider anthropic  # 只看 Anthropic
zenmux models --search gpt-5        # 关键字搜索
zenmux models --json                # JSON 格式
```

### 3. 文本对话

```bash
zenmux chat "用 Go 实现快速排序"
zenmux chat "分析这段代码" --system "你是资深代码审查员"
zenmux chat "详细解释" -m openai/gpt-5 --no-stream
```

默认流式输出，实时打印回复。

### 4. 生成图片

```bash
zenmux image "赛博朋克城市夜景"
zenmux image "logo 设计" -s 1536x1024 -q high -o logo --format webp
```

### 5. 生成视频

视频生成是异步任务，CLI 会自动提交→轮询（每 15s）→下载 mp4。

```bash
zenmux video "金毛在夕阳下的海滩奔跑" --resolution 720p --ratio 16:9 --duration 8
# 图生视频
zenmux video "小狗站起来跑向海浪" --image first_frame.png -o my_video
# 带音频
zenmux video "歌手舞台表演" --audio
```

### 6. 语音

```bash
# 文本转语音
zenmux voice "你好，欢迎使用 zenmux"
zenmux voice "Hello world" --voice Kore -o hello.pcm

# 语音转文本
zenmux voice transcribe speech.wav
zenmux voice transcribe recording.mp3 --language zh
```

### 7. 结构化评估

使用 `typesafe/jev-latest` 等 System One 评估模型，对内容做类型化评估：是/否判定（noul）、选项分类（choice）、分级打分（score），返回概率化的结构化答案。

```bash
zenmux eval "救命！我的账户回款已经连续 3 天失败了。" -q questions.json
zenmux eval --state-file dialog.json -q questions.json --json
```

### 8. 套餐状态

```bash
zenmux status
```

展示套餐等级、到期、账号状态、5 小时 / 7 天 / 月度配额、PAYG 余额与 Flow 汇率。

## 配置项参考

| 配置键 | 说明 | 默认值 |
| --- | --- | --- |
| `api_key` | 普通 API Key | — |
| `management_api_key` | Management API Key | — |
| `base_url` | API 基础地址 | `https://zenmux.ai/api/v1` |
| `proxy` | HTTP 代理地址 | — |
| `default_model.chat` | chat 默认模型 | `openai/gpt-5` |
| `default_model.image` | image 默认模型 | `gpt-image-2` |
| `default_model.video` | video 默认模型 | `bytedance/doubao-seedance-2.0` |
| `default_model.tts` | voice (TTS) 默认模型 | `google/gemini-3.1-flash-tts-preview` |
| `default_model.stt` | voice transcribe 默认模型 | `qwen/qwen3-asr-flash` |
| `default_model.eval` | eval 默认模型 | `typesafe/jev-latest` |

优先级：命令行 `--flag` > 环境变量 > 配置文件 > 内置默认值。

## 项目结构

```
zenmux-cli/
├── main.go
└── internal/
    ├── config/   # 配置加载/保存
    ├── client/   # HTTP 客户端（代理、鉴权、SSE、错误解析）
    ├── api/      # 各业务接口封装
    └── cmd/      # cobra 子命令
```

## 参考文档

- [ZenMux 快速开始](https://docs.zenmux.ai/zh/guide/quickstart)
- [Chat Completions API](https://docs.zenmux.ai/zh/api/openai/create-chat-completion)
- [Image Generation API](https://docs.zenmux.ai/zh/api/openai/generate-an-image)
- [Video Generation API](https://docs.zenmux.ai/zh/api/zenmux/generate-videos-native)
- [TTS / STT API](https://docs.zenmux.ai/zh/api/openai/create-audio-speech)
- [结构化评估（System One）API](https://docs.zenmux.ai/zh/api/typesafe/systemone)
- [Platform Management API](https://docs.zenmux.ai/zh/api/platform/subscription-detail)
