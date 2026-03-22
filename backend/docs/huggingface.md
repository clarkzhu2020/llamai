# HuggingFace 集成指南

HuggingFace 提供云端推理 API，支持 Image、Speech、Video 等多模态模型。

## 获取 API Token

1. 访问 [HuggingFace Settings](https://huggingface.co/settings/tokens)
2. 点击 "New Token"
3. 选择角色 "Read" 或 "Write"
4. 复制生成的 Token

## LocalMAI 配置

在 `.env` 文件中配置：

```env
HF_TOKEN=hf_your_token_here
```

## 安装 HuggingFace Hub (可选)

如果需要本地运行模型：

```bash
pip install huggingface-hub transformers accelerate
```

## 支持的模型类型

### Image 生成模型

| 模型 | 描述 | VRAM 需求 |
|------|------|----------|
| stabilityai/stable-diffusion-xl-base-1.0 | SDXL 1.0 | 8GB |
| stabilityai/stable-diffusion-2-1 | SD 2.1 | 5GB |
| stabilityai/stable-diffusion-3-medium | SD 3 Medium | 12GB |
| black-forest-labs/FLUX.1-dev | FLUX.1 Dev | 16GB |
| black-forest-labs/FLUX.1-schnell | FLUX.1 Schnell | 16GB |
| ByteDance/SDXL-Lightning | SDXL-Lightning (超快) | 8GB |
| playgroundai/playground-v2.5-1024px-aesthetic | Playground v2.5 | 8GB |

### Image-to-Image 模型

| 模型 | 描述 | VRAM 需求 |
|------|------|----------|
| stabilityai/stable-diffusion-img2img | SD Image-to-Image | 5GB |

### Speech/TTS 模型

| 模型 | 描述 | VRAM 需求 |
|------|------|----------|
| facebook/fastspeech2-en-ljspeech | FastSpeech 2 | 2GB |
| microsoft/speecht5_tts | SpeechT5 | 2GB |
| suno/bark-small | Bark (小模型) | 2GB |
| suno/bark | Bark (完整模型) | 4GB |

### Speech Recognition 模型

| 模型 | 描述 | VRAM 需求 |
|------|------|----------|
| openai/whisper-large | Whisper Large | 3GB |
| openai/whisper-medium | Whisper Medium | 2GB |
| openai/whisper-small | Whisper Small | 1GB |

### Video 生成模型

| 模型 | 描述 | VRAM 需求 |
|------|------|----------|
| zeroscope_v2_576w | ZeroScope 576w | 8GB |
| damo-vilab/image-to-video | Image-to-Video | 8GB |

## API 使用

### 图片生成

```bash
curl -X POST http://localhost:8080/api/image/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stabilityai/stable-diffusion-xl-base-1.0",
    "prompt": "a beautiful sunset over the ocean, realistic",
    "negative_prompt": "blurry, low quality",
    "width": 1024,
    "height": 1024,
    "steps": 30,
    "cfg_scale": 7.5
  }'
```

### 图片编辑

```bash
curl -X POST http://localhost:8080/api/image/edit \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stabilityai/stable-diffusion-img2img",
    "prompt": "a golden retriever puppy",
    "input_image": "data:image/png;base64,..."
  }'
```

### 语音合成 (TTS)

```bash
curl -X POST http://localhost:8080/api/speech/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "suno/bark",
    "text": "Hello, this is a test of the text to speech system."
  }'
```

### 语音识别

```bash
curl -X POST http://localhost:8080/api/speech/transcribe \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/whisper-large",
    "audio": "data:audio/wav;base64,..."
  }'
```

### 视频生成

```bash
curl -X POST http://localhost:8080/api/video/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "zeroscope_v2_576w",
    "prompt": "a drone flying over mountains"
  }'
```

## Python 示例

```python
import requests
import base64

# 图片生成
response = requests.post("http://localhost:8080/api/image/generate", json={
    "model": "stabilityai/stable-diffusion-xl-base-1.0",
    "prompt": "a cat sitting on a windowsill",
    "width": 512,
    "height": 512
})
task_id = response.json()["task_id"]

# 轮询结果
import time
while True:
    result = requests.get(f"http://localhost:8080/api/tasks/{task_id}").json()
    if result["state"] == "completed":
        print(result["result"]["output"])
        break
    time.sleep(1)
```

## Inference API vs Inference Endpoint

### Inference API (免费托管)

HuggingFace 提供免费的 Inference API，直接调用：

```python
from huggingface_hub import InferenceClient

client = InferenceClient("stabilityai/stable-diffusion-xl-base-1.0", token="hf_xxx")

image = client.text_to_image("a beautiful landscape")
```

### Local 部署 (自托管)

使用 `text-generation-inference` (TGI) 部署：

```bash
# 安装 TGI
docker pull ghcr.io/huggingface/text-generation-inference:latest

# 启动
docker run --gpus all -p 8080:80 \
  -v $HF_HOME:/data \
  ghcr.io/huggingface/text-generation-inference:latest \
  --model-id meta-llama/Llama-2-7b-hf
```

## 速率限制

免费 Tier:
- 推理 API: 每分钟有限请求数
- 付费 Tier 可获得更高配额

建议：
1. 使用 `HF_TOKEN` 配置个人 Token
2. 对于生产环境，考虑部署自己的推理服务端点

## 常见问题

### Q: API 请求超时

HuggingFace 推理 API 有 10 分钟超时限制。大型模型可能需要更长时间。

### Q: 模型不在服务时间

某些模型可能因为 Cold Start 而暂时不可用，稍后重试即可。

### Q: 如何加速推理

```bash
# 使用更小的模型
stabilityai/stable-diffusion-2-1

# 使用量化模型
ByteDance/SDXL-Lightning-4step
```

### Q: 如何使用自己的模型

将模型上传到 HuggingFace：

```bash
# 安装 Git LFS
git lfs install

# 克隆并上传
git clone https://huggingface.co/your-username/your-model
# 添加你的文件
git add .
git push
```

## 模型推荐

### 最佳质量图片
- `stabilityai/stable-diffusion-xl-base-1.0` + Refiner
- `black-forest-labs/FLUX.1-dev` (需要较长生成时间)

### 快速生成
- `ByteDance/SDXL-Lightning-4step` - 4步完成
- `sdxl-turbo` - 实时生成

### 中文支持
- `深深开源/anywidget` 系列
- `腾讯/混元` 系列

## 下一步

- [SD WebUI 集成](./sd-webui.md) - 本地 SD 部署
- [Llama.cpp 集成](./llamacpp.md) - GGUF 模型支持
- [vLLM 集成](./vllm.md) - 高性能推理
- [API 使用](./api.md) - 完整 API 文档
