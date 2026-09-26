# zenmux CLI 使用手册

本文档介绍如何使用 zenmux CLI 进行 **文本对话、语音、图片、视频** 四类多模态生成。

> 📌 所有命令共用一份配置，首次使用请先执行：
> ```bash
> zenmux config set api_key           sk-xxxxxxxxxxxxxxxx
> zenmux config set management_api_key sk-mg-xxxxxxxxxxxxxx
> zenmux config set proxy             http://127.0.0.1:7897   # 国内需要代理
> ```

---

## 选择模型的三种方式

| 方式 | 命令 | 生效范围 |
| --- | --- | --- |
| 临时指定 | 命令加 `-m <模型>` | 仅本次 |
| 改默认值 | `zenmux config set default_model.<类型> <模型>` | 长期 |
| 不指定 | 用内置默认模型 | 长期 |

查看平台支持的所有模型：

```bash
zenmux models                       # 全部模型
zenmux models --modality image      # 按输出模态过滤
zenmux models --provider openai     # 按供应商过滤
zenmux models --search gpt          # 关键字搜索
```

---

## 一、文本对话（chat）

与各大语言模型对话，**默认流式输出**（逐字打印），支持设定系统角色。

### 基本用法

```bash
zenmux chat "用 Go 实现一个快速排序"
```

### 指定模型

```bash
zenmux chat "分析这段代码" -m anthropic/claude-opus-5
zenmux chat "翻译成英文" -m google/gemini-3.1-pro-preview
```

### 设定系统角色（让模型扮演特定身份）

```bash
zenmux chat "审查一下这个函数的安全性" --system "你是一位资深安全工程师，用中文回答"
```

### 禁用流式（整段输出，适合管道处理）

```bash
zenmux chat "总结要点" --no-stream
```

### 命令参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-m, --model` | 模型 slug | `default_model.chat`（默认 `openai/gpt-5`） |
| `--system` | 系统提示词，设定模型角色 | 无 |
| `--no-stream` | 禁用流式输出，整段返回 | 关闭（即启用流式） |

### 输出示例

```
快速排序的核心思想是分治……（逐字流式打印）

[用量] prompt=12 completion=128 total=140
```

---

## 二、语音（voice）

语音功能分两个子命令：**文本转语音（TTS）** 和 **语音转文本（STT）**。

### 2.1 文本转语音（TTS）

把文字读出来，生成音频文件。

```bash
# 基本用法：默认输出 output.pcm
zenmux voice "你好丰坚，欢迎使用 zenmux 命令行工具"

# 指定音色和输出文件
zenmux voice "Hello world" --voice Kore -o hello.pcm

# 指定 TTS 模型
zenmux voice "今天天气不错" -m google/gemini-3.1-flash-tts-preview
```

#### 参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-m, --model` | TTS 模型 slug | `default_model.tts`（默认 `google/gemini-3.1-flash-tts-preview`） |
| `--voice` | 音色标识符（因模型而异） | 无 |
| `--format` | 音频编码格式 | `pcm` |
| `-o, --output` | 输出文件名 | `output.<格式>` |

> 💡 **关于播放**：默认生成的 PCM 是裸音频格式。如需用播放器直接播放，可用 ffmpeg 转换：
> ```bash
> ffmpeg -f s16le -ar 24000 -ac 1 -i output.pcm output.wav
> ```

### 2.2 语音转文本（STT）

把音频文件转写成文字。

```bash
# 基本用法
zenmux voice transcribe speech.wav

# 指定语言（提高识别准确率）
zenmux voice transcribe recording.mp3 --language zh

# 指定 STT 模型
zenmux voice transcribe voice.m4a -m qwen/qwen3-asr-flash
```

#### 参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-m, --model` | STT 模型 slug | `default_model.stt`（默认 `qwen/qwen3-asr-flash`） |
| `--language` | 音频语言（ISO-639-1，如 `zh`/`en`/`ja`） | 自动检测 |

#### 支持的音频格式

`wav` / `mp3` / `flac` / `m4a` / `ogg` / `webm` / `aac`

---

## 三、图片生成（image）

根据文字描述生成图片，默认输出 PNG。

### 基本用法

```bash
zenmux image "一只在夕阳下的海滩上奔跑的金毛，电影质感"
# → 生成 output.png
```

### 控制尺寸、质量、格式

```bash
# 横版高清
zenmux image "赛博朋克城市夜景" -s 1536x1024 -q high -o cyber

# WebP 格式（体积更小）
zenmux image "简约 logo 设计" --format webp -o logo
```

### 指定模型

```bash
zenmux image "水彩画风格的山景" -m gpt-image-2
```

### 命令参数

| 参数 | 说明 | 可选值 | 默认值 |
| --- | --- | --- | --- |
| `-m, --model` | 图片模型 slug | 见 `zenmux models --modality image` | `gpt-image-2` |
| `-s, --size` | 图片尺寸 | `1024x1024` / `1536x1024` / `1024x1536` / `auto` | `1024x1024` |
| `-q, --quality` | 质量 | `low` / `medium` / `high` / `auto` | `auto` |
| `-n, --number` | 生成数量 | 1–10 | 1 |
| `--format` | 输出格式 | `png` / `jpeg` / `webp` | `png` |
| `-o, --output` | 输出文件名（不含扩展名） | — | `output` |

> 💡 多张图片时会自动追加序号：`cyber_1.png`、`cyber_2.png`。

---

## 四、视频生成（video）

根据文字描述生成视频。视频生成是**异步任务**，CLI 会自动：提交 → 每 15 秒轮询 → 下载 MP4。

### 4.1 文生视频

```bash
zenmux video "一只金毛在夕阳下的海滩上奔跑，电影质感"
# → 生成 output.mp4
```

### 4.2 控制视频参数

```bash
zenmux video "猫咪在暖光下读书" \
  --resolution 1080p \
  --ratio 16:9 \
  --duration 8 \
  -o my_video
```

### 4.3 图生视频（用一张图作为起始帧）

```bash
zenmux video "小狗站起来跑向海浪" --image first_frame.png -o dog_running
```

### 4.4 生成带音频的视频

```bash
zenmux video "歌手在舞台上表演" --audio
```

### 命令参数

| 参数 | 说明 | 可选值 | 默认值 |
| --- | --- | --- | --- |
| `-m, --model` | 视频模型 slug | 见官方文档 | `bytedance/doubao-seedance-2.0` |
| `--resolution` | 分辨率 | `480p` / `720p` / `1080p` | 模型默认 |
| `--ratio` | 宽高比 | `16:9` / `9:16` / `1:1` / `adaptive` | 模型默认 |
| `--duration` | 时长（秒） | 如 `5` / `8` / `10` | 模型默认 |
| `--image` | 首帧图片路径（图生视频） | 本地文件路径 | 无 |
| `--audio` | 是否生成音频轨道 | — | 关闭 |
| `-o, --output` | 输出文件名（不含扩展名） | — | `output` |

### 执行流程示例

```
提交视频生成任务（模型 bytedance/doubao-seedance-2.0）...
任务已提交，ID: 451f6e18...，状态: queued
[01:27:55] 状态: queued（已等待 1s）
[01:28:11] 状态: running（已等待 17s）
[01:32:28] 状态: succeeded（已等待 4m34s）
下载视频到 my_video.mp4 ...
完成！已保存到 my_video.mp4（3.11 MB，耗时 4m34s）
```

> ⏱ **耗时提示**：视频生成通常需要 30 秒到 3 分钟，1080p 或较长时长会更久。CLI 会自动轮询，期间可去做别的事。

---

## 五、结构化评估（eval）

使用 System One 评估模型（默认 `typesafe/jev-latest`）对一段内容做**类型化评估**，直接返回概率化的结构化答案。适合内容分类、情感打分、紧急度判断、工单路由等场景。

每个问题是三种类型之一：

| 类型 | 作用 | 返回 |
| --- | --- | --- |
| `noul` | 是/否判定 | 为"是"的概率（0~1） |
| `choice` | 从自定义选项中选一个 | 选中项 + 概率分布 + 置信度 |
| `score` | 按自定义分级打分 | 概率加权分值 + 分级说明 + 置信度 |

### 准备问题定义（JSON 文件）

```json
{
  "is_urgent": {
    "type": "noul",
    "instructions": "这段内容是否传达出紧迫感？"
  },
  "department": {
    "type": "choice",
    "instructions": "应由哪个团队来处理？",
    "criteria": {
      "billing": "支付、开票、退款",
      "technical": "缺陷、故障、集成",
      "sales": "定价、升级、新开户"
    }
  },
  "frustration": {
    "type": "score",
    "instructions": "客户的不满程度如何？",
    "criteria": ["平静", "不满", "非常愤怒"]
  }
}
```

### 基本用法

```bash
# 评估一段文本
zenmux eval "救命！我的账户回款已经连续 3 天失败了。" -q questions.json

# 从标准输入读问题定义
cat questions.json | zenmux eval "用户对话记录……" -q -

# 评估结构化数据（对话记录、业务状态等 JSON）
zenmux eval --state-file dialog.json -q questions.json

# 输出原始 JSON（方便管道处理）
zenmux eval "这条评论很满意" -q questions.json --json
```

### 输出示例

```
department [choice]: billing（置信度 0.90）
  概率分布: billing=0.93 sales=0.00 technical=0.07
frustration [score]: 1.74（置信度 0.62）
  分级: 0=平静 1=不满 2=非常愤怒
  概率分布: 0=0.00 1=0.26 2=0.74
is_urgent [noul]: 0.97

[用量] input=431 output=73
```

### 命令参数

| 参数 | 说明 | 默认值 |
| --- | --- | --- |
| `-m, --model` | 评估模型 slug | `default_model.eval`（默认 `typesafe/jev-latest`） |
| `-q, --questions` | 问题定义 JSON 文件，`-` 表示标准输入 | 必填 |
| `--state-file` | 待评估的结构化 JSON 文件（替代文本参数） | 无 |
| `--json` | 输出原始 JSON 响应 | 关闭 |

---

## 附录

### 全部命令一览

| 命令 | 说明 |
| --- | --- |
| `zenmux chat` | 文本对话（流式） |
| `zenmux voice` | 文本转语音（TTS） |
| `zenmux voice transcribe` | 语音转文本（STT） |
| `zenmux image` | 图片生成 |
| `zenmux video` | 视频生成（异步） |
| `zenmux eval` | 结构化评估（noul/choice/score） |
| `zenmux models` | 列出所有支持的模型 |
| `zenmux status` | 查询套餐状态与用量 |
| `zenmux config` | 查看与修改配置 |

### 配置项参考

```bash
zenmux config set api_key                <普通Key>      # 调用模型
zenmux config set management_api_key     <ManagementKey> # 查询套餐
zenmux config set proxy                  http://127.0.0.1:7897
zenmux config set default_model.chat     openai/gpt-5
zenmux config set default_model.image    gpt-image-2
zenmux config set default_model.video    minimax/minimax-h3
zenmux config set default_model.tts      google/gemini-3.1-flash-tts-preview
zenmux config set default_model.stt      qwen/qwen3-asr-flash
zenmux config set default_model.eval     typesafe/jev-latest
zenmux config show                                      # 查看当前配置
```

### 优先级

命令行 `-m` 参数 > 环境变量 > 配置文件 `default_model.*` > 内置默认值

### 相关文档

- [ZenMux 官方文档](https://docs.zenmux.ai/zh/)
- [模型列表页](https://zenmux.ai/models)
