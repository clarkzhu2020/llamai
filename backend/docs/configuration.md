# 配置指南

LocalMAI 支持通过 `.env` 文件或环境变量进行配置。

## 配置文件

### 创建配置文件

```bash
# 复制示例配置
cp .env.example .env

# 编辑配置
nano .env
```

### 环境变量优先级

1. 环境变量 > `.env` 文件 > 默认值

## 配置项说明

### 必需配置

#### HF_TOKEN

HuggingFace API Token，用于访问 HuggingFace 推理 API。

```env
HF_TOKEN=hf_your_token_here
```

获取方式：
1. 访问 https://huggingface.co/settings/tokens
2. 创建新 Token (需要 "Read" 权限)
3. 复制 Token 值

### 可选配置

#### LOCALMAI_API_KEY

API 认证密钥，用于保护 API 端点。

```env
LOCALMAI_API_KEY=your-secret-key
```

使用方式：
```bash
curl -H "X-API-Key: your-secret-key" http://localhost:8080/api/health
```

留空表示禁用认证。

#### OLLAMA_BASE_URL

Ollama 服务器地址。

```env
OLLAMA_BASE_URL=http://localhost:11434
```

默认：`http://localhost:11434`

#### LLAMA_CPP_URL

Llama.cpp Server 地址。

```env
LLAMA_CPP_URL=http://localhost:8080
```

默认：`http://localhost:8080`

#### VLLM_URL

vLLM 服务器地址。

```env
VLLM_URL=http://localhost:8000
```

默认：`http://localhost:8000`

#### SD_WEBUI_URL

Stable Diffusion WebUI 地址。

```env
SD_WEBUI_URL=http://localhost:7860
```

默认：`http://localhost:7860`

#### PORT

LocalMAI 服务端口。

```env
PORT=8080
```

默认：`8080`

#### GPU_MEMORY_MB

模拟 GPU 内存大小（MB）。仅在未检测到 NVIDIA GPU 时使用。

```env
GPU_MEMORY_MB=16000
```

默认：`16000`（自动检测真实 GPU，无需配置）

## 完整配置示例

```env
# HuggingFace
HF_TOKEN=hf_example_token_here

# API 认证
LOCALMAI_API_KEY=my-secret-key

# 服务地址
OLLAMA_BASE_URL=http://localhost:11434
LLAMA_CPP_URL=http://localhost:8080
VLLM_URL=http://localhost:8000
SD_WEBUI_URL=http://localhost:7860

# 服务器
PORT=8080

# GPU
GPU_MEMORY_MB=16000
```

## 后端服务配置

### 仅使用 Ollama

如果只需要文本生成，最小配置：

```env
HF_TOKEN=  # 留空
OLLAMA_BASE_URL=http://localhost:11434
```

### Ollama + SD WebUI (本地图片生成)

```env
HF_TOKEN=  # 留空
OLLAMA_BASE_URL=http://localhost:11434
SD_WEBUI_URL=http://localhost:7860
```

### 完整配置 (所有功能)

```env
HF_TOKEN=hf_your_token
OLLAMA_BASE_URL=http://localhost:11434
LLAMA_CPP_URL=http://localhost:8080
VLLM_URL=http://localhost:8000
SD_WEBUI_URL=http://localhost:7860
PORT=8080
GPU_MEMORY_MB=16000
```

## 配置检查

启动时 LocalMAI 会检查各后端的连接状态：

```
✅ Loaded configuration from .env
✅ Connected to Ollama server
⚠️  HF_TOKEN not set (HuggingFace inference disabled)
⚠️  Llama.cpp not available at http://localhost:8080
⚠️  vLLM not available at http://localhost:8000
⚠️  SD WebUI not available at http://localhost:7860
```

确保所需后端服务在启动 LocalMAI 之前已经运行。

## 下一步

- [Ollama 集成](./ollama.md)
- [Llama.cpp 集成](./llamacpp.md)
- [vLLM 集成](./vllm.md)
- [HuggingFace 集成](./huggingface.md)
- [SD WebUI 集成](./sd-webui.md)
