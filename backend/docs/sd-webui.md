# Stable Diffusion WebUI 集成指南

AUTOMATIC1111's Stable Diffusion WebUI 是最流行的本地图片生成工具，提供丰富的功能和优秀的用户体验。

## 安装

### 基本要求

- **操作系统**: Windows 10/11, Linux, macOS
- **GPU**: NVIDIA GPU (至少 4GB VRAM，8GB 推荐)
- **RAM**: 16GB+
- **磁盘空间**: 50GB+

### Windows 安装

```bash
# 1. 安装 Python 3.10.x (不要使用 3.11+)
# 下载地址: https://www.python.org/downloads/

# 2. 安装 Git
# 下载地址: https://git-scm.com/download/win

# 3. 克隆仓库
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# 4. 下载基础模型
# 将 .safetensors 模型文件放入 models/Stable-diffusion 目录

# 5. 启动 (首次会自动安装依赖)
./webui-user.bat
```

### Linux/macOS 安装

```bash
# 1. 安装依赖
# Ubuntu/Debian
sudo apt update
sudo apt install python3-venv python3-pip

# 2. 克隆仓库
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# 3. 下载模型
mkdir -p models/Stable-diffusion
# 将 .safetensors 文件放入该目录

# 4. 启动
./webui.sh
```

### Docker 安装 (推荐)

```bash
# 使用 docker-compose
version: '3'
services:
  sd-webui:
    image: ghcr.io/automatic1111/stable-diffusion-webui:latest
    ports:
      - "7860:7860"
    volumes:
      - ./models:/app/models
      - ./outputs:/app/outputs
    environment:
      - COMMANDLINE_ARGS=--api --listen 0.0.0.0
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]
```

```bash
# 启动
docker-compose up -d
```

## 启动 API 服务

### 基本 API 启动

```bash
# Windows
set COMMANDLINE_ARGS=--api --listen
./webui-user.bat

# Linux/macOS
export COMMANDLINE_ARGS="--api --listen"
./webui.sh
```

### 推荐参数

```bash
# 完整参数示例
./webui.sh \
  --api \
  --listen 0.0.0.0 \
  --port 7860 \
  --enable-insecure-extension-access \
  --gradio-debug \
  --no-half-vae \
  --xformers
```

### 常用参数说明

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `--api` | 启用 API | 必须 |
| `--listen` | 允许远程访问 | 0.0.0.0 |
| `--port` | 端口 | 7860 |
| `--xformers` | 启用 xformers (加速) | 推荐 |
| `--medvram` | 节省显存 | 低配置必选 |
| `--lowvram` | 更低显存 | 低配置必选 |
| `--opt-sdp-attention` | 更好的 attention | 实验性 |

## LocalMAI 配置

在 `.env` 文件中配置：

```env
SD_WEBUI_URL=http://localhost:7860
```

## API 使用

### 图片生成 (txt2img)

```bash
curl -X POST http://localhost:7860/sdapi/v1/txt2img \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "a beautiful sunset over ocean, realistic",
    "negative_prompt": "blurry, low quality, watermark",
    "width": 512,
    "height": 512,
    "steps": 30,
    "cfg_scale": 7,
    "sampler_name": "Euler a"
  }'
```

### 图片编辑 (img2img)

```bash
curl -X POST http://localhost:7860/sdapi/v1/img2img \
  -H "Content-Type: application/json" \
  -d '{
    "init_images": ["data:image/png;base64,..."],
    "prompt": "make it into anime style",
    "denoising_strength": 0.75
  }'
```

### 获取模型列表

```bash
# Stable Diffusion 模型
curl http://localhost:7860/sdapi/v1/sd-models

# VAE 模型
curl http://localhost:7860/sdapi/v1/sd-vae

# Lora 模型
curl http://localhost:7860/sdapi/v1/loras
```

###  ControlNet

```bash
curl -X POST http://localhost:7860/sdapi/v1/ControlNet/img2img \
  -H "Content-Type: application/json" \
  -d '{
    "input_image": "data:image/png;base64,...",
    "control_units": [
      {
        "model": "canny",
        "input_image": "data:image/png;base64,..."
      }
    ],
    "prompt": "..."
  }'
```

## LocalMAI API 使用

```bash
# 通过 LocalMAI 代理访问 SD WebUI
curl -X POST http://localhost:8080/api/image/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd-xl",
    "prompt": "a beautiful landscape",
    "width": 1024,
    "height": 1024
  }'
```

## 支持的模型列表

LocalMAI 预注册了以下 SD WebUI 模型：

| 模型名称 | 路径 | VRAM 需求 | 描述 |
|---------|------|----------|------|
| sd-xl | SDXL | 8GB | Stable Diffusion XL |
| sd-2.1 | v2-1_768-ema-pruned | 5GB | SD 2.1 |
| sdxl-turbo | sdxl-turbo | 4GB | SDXL Turbo (快速) |
| sd-turbo | sd-turbo | 4GB | SD Turbo (快速) |
| majicmix | majicmix | 4GB | MajicMix (动漫风格) |
| anything-v5 | anything-v5 | 4GB | Anything V5 (动漫风格) |
| counterfeit | counterfeit | 4GB | Counterfeit (动漫风格) |

## 模型安装

### 下载模型

1. 从 [Civitai](https://civitai.com/) 下载模型 (.safetensors)
2. 从 [HuggingFace](https://huggingface.co/models?other=stable-diffusion) 下载

### 模型存放位置

```
stable-diffusion-webui/
├── models/
│   ├── Stable-diffusion/     # SD 主模型
│   │   ├── sd_xl_base_1.0.safetensors
│   │   ├── v1-5-pruned.safetensors
│   │   └── ...
│   ├── VAE/                  # VAE 模型
│   ├── Lora/                 # LoRA 模型
│   └── ControlNet/           # ControlNet 模型
```

### 模型格式转换

```bash
# 将 ckpt 转换为 safetensors
python scripts/checkpoint_converter.py \
  --input model.ckpt \
  --output model.safetensors
```

## 常用采样器

| 采样器 | 速度 | 质量 | 适用场景 |
|--------|------|------|----------|
| Euler | 快 | 良好 | 快速测试 |
| Euler a | 中 | 良好 | 精细控制 |
| DPM++ 2M Karras | 中 | 优秀 | 平衡之选 |
| DPM++ SDE Karras | 慢 | 优秀 | 高质量 |
| DDIM | 慢 | 优秀 | 精细控制 |
| PLMS | 中 | 良好 | 旧版兼容 |

## 常见问题

### Q: 显存不足 (CUDA out of memory)

```bash
# 启动时添加参数
./webui.sh --medvram --lowvram

# 或者在 webui-user.bat 中设置
set COMMANDLINE_ARGS=--medvram --lowvram
```

### Q: 如何提升生成速度

```bash
# 启用 xformers
./webui.sh --xformers

# 使用更快的采样器
# Euler a, DPM++ 2M Karras

# 减小图片尺寸
```

### Q: 模型加载失败

```bash
# 检查模型文件完整性
sha256sum model.safetensors

# 检查文件名和路径
# 文件名不要有特殊字符
```

### Q: API 访问被拒绝

```bash
# 确保启动时添加了 --api --listen
export COMMANDLINE_ARGS="--api --listen 0.0.0.0"
```

### Q: 如何使用 VAE

```bash
# 下载 VAE 文件放入 models/VAE/
# 在生成时指定
curl -X POST http://localhost:7860/sdapi/v1/txt2img \
  -d '{"prompt": "...", "sd_vae": "vae-ft-mse-840000-ema-pruned"}'
```

## 进阶技巧

### 批量生成

```bash
# 使用 API 批量生成
for i in {1..5}; do
  curl -X POST http://localhost:7860/sdapi/v1/txt2img \
    -d "{\"prompt\": \"artwork $i\"}" > "output_$i.json"
done
```

### 模型切换

```bash
# 通过 API 切换模型
curl -X POST http://localhost:7860/sdapi/v1/options \
  -H "Content-Type: application/json" \
  -d '{"sd_model_checkpoint": "model_name.safetensors"}'
```

## 下一步

- [HuggingFace 集成](./huggingface.md) - 云端图片生成
- [Llama.cpp 集成](./llamacpp.md) - LLM 推理
- [vLLM 集成](./vllm.md) - 高性能推理
- [API 使用](./api.md) - 完整 API 文档
