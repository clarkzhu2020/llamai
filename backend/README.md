# LocalMAI - 多模态 AI 模型服务平台

一个类似 Ollama 的多模态 AI 模型管理平台，支持分布式 Worker、GPU 调度和多种模型后端。

## 功能特性

- **多模态支持**: 文本生成、图像生成、语音合成、语音识别、视频生成
- **多种后端支持**: Ollama、Llama.cpp、vLLM、HuggingFace、Stable Diffusion WebUI、LTX-Video
- **分布式架构**: Worker 节点支持分布式任务执行
- **GPU 调度**: 智能 GPU 内存分配和负载均衡
- **RESTful API**: 易于与应用集成
- **Web UI**: 内置可视化界面

---

## 目录

1. [快速开始](#快速开始)
2. [配置指南](./docs/configuration.md)
3. [后端集成指南](./docs/)
4. [API 使用文档](./docs/api.md)
5. [故障排除](./docs/troubleshooting.md)

---

## 快速开始

### 1. 安装依赖

```bash
# 安装 Go (1.21+)
# https://golang.org/doc/install

# 克隆项目
cd backend
go mod tidy
```

### 2. 配置环境

复制配置示例文件：

```bash
cp .env.example .env
```

编辑 `.env` 文件，配置必要的参数：

```env
# HuggingFace Token (必需，用于访问 HF 模型)
HF_TOKEN=your_hf_token_here

# API 认证密钥 (可选)
LOCALMAI_API_KEY=

# Ollama 地址 (默认已配置)
OLLAMA_BASE_URL=http://localhost:11434

# Llama.cpp 地址
LLAMA_CPP_URL=http://localhost:8080

# vLLM 地址
VLLM_URL=http://localhost:8000

# Stable Diffusion WebUI 地址
SD_WEBUI_URL=http://localhost:7860
```

### 3. 启动后端服务

根据你需要使用的模型，启动相应的服务：

```bash
# 启动 LocalMAI
go run .
```

服务运行在: http://localhost:8080

### 4. 验证安装

```bash
curl http://localhost:8080/api/health
```

---

## 后端集成指南

详细的后端配置和使用指南：

| 后端 | 文档 | 适用场景 |
|------|------|----------|
| **Ollama** | [快速上手](./docs/ollama.md) | LLM/Vision 本地部署，最简单 |
| **Llama.cpp** | [集成指南](./docs/llamacpp.md) | GGUF 格式模型，CPU/GPU 灵活 |
| **vLLM** | [部署指南](./docs/vllm.md) | 高性能生产环境 |
| **HuggingFace** | [API 使用](./docs/huggingface.md) | Image/Speech/Video 云端推理 |
| **SD WebUI** | [本地部署](./docs/sd-webui.md) | 本地图片生成，自定义强 |

---

## 模型列表

### 文本生成模型 (LLM)

#### Ollama (26 个模型)

| 模型 | VRAM | 描述 |
|------|------|------|
| llama3 | 4GB | Meta Llama 3 8B |
| llama3.1 | 4GB | Meta Llama 3.1 8B |
| llama3.2 | 4GB | Meta Llama 3.2 |
| mistral | 4GB | Mistral 7B |
| qwen2.5 | 4GB | Qwen 2.5 7B |
| phi3 | 2GB | Microsoft Phi-3 Mini |
| ... | ... | [全部模型](./docs/ollama.md) |

#### Llama.cpp (12 个模型)

| 模型 | VRAM | 描述 |
|------|------|------|
| llama-7b | 4GB | LLaMA 7B GGUF |
| llama-70b | 16GB | LLaMA 70B GGUF |
| mistral-7b | 4GB | Mistral 7B GGUF |
| mixtral-8x7b | 8GB | Mixtral 8x7B GGUF |
| qwen2-7b | 4GB | Qwen2 7B GGUF |
| ... | ... | [全部模型](./docs/llamacpp.md) |

#### vLLM (12 个模型)

| 模型 | VRAM | 描述 |
|------|------|------|
| meta-llama/Llama-2-7b-hf | 4GB | LLaMA-2 7B |
| meta-llama/Meta-Llama-3-8B | 4GB | LLaMA-3 8B |
| mistralai/Mistral-7B-v0.1 | 4GB | Mistral 7B |
| ... | ... | [全部模型](./docs/vllm.md) |

### 图像生成模型

| 模型 | VRAM | 后端 | 描述 |
|------|------|------|------|
| stable-diffusion-xl-base-1.0 | 8GB | HF/SD WebUI | SDXL 1.0 |
| SDXL-Lightning | 8GB | HF | ⚡ 4步完成 |
| FLUX.1-dev | 16GB | HF | 高质量 |
| ... | ... | ... | [更多](./docs/huggingface.md) |

### 语音模型

| 模型 | 类型 | 后端 |
|------|------|------|
| suno/bark | TTS | HuggingFace |
| openai/whisper-large | ASR | HuggingFace |

### 视频生成模型

| 模型 | 类型 | 后端 |
|------|------|------|
| zeroscope_v2_576w | Text-to-Video | HuggingFace |
| damo-vilab/image-to-video | Image-to-Video | HuggingFace |

---

## API 使用

完整 API 文档：[API 使用文档](./docs/api.md)

### 文本生成

```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model": "llama3", "prompt": "Hello"}'
```

### 对话

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### 图片生成

```bash
curl -X POST http://localhost:8080/api/image/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stabilityai/stable-diffusion-xl-base-1.0",
    "prompt": "a beautiful sunset"
  }'
```

---

## Web UI

打开浏览器访问: http://localhost:8080

支持：
- 模型浏览
- 文本生成/聊天
- 图片生成
- 任务状态监控

---

## 项目结构

```
backend/
├── main.go              # 主入口
├── api.go               # HTTP API 处理器
├── manager.go           # 模型管理器
├── scheduler.go         # 任务调度器
├── gpu.go               # GPU 管理器
├── ollama.go            # Ollama 客户端
├── llamacpp.go         # Llama.cpp 客户端
├── vllm.go             # vLLM 客户端
├── huggingface.go       # HuggingFace 客户端
├── types.go            # 类型定义
├── docs/                # 文档目录
│   ├── ollama.md
│   ├── llamacpp.md
│   ├── vllm.md
│   ├── huggingface.md
│   ├── sd-webui.md
│   ├── api.md
│   └── configuration.md
├── frontend/
│   └── index.html       # Web UI
└── worker/              # Worker 节点
```

---

## 技术栈

- **后端**: Go 1.21+
- **API**: net/http (标准库)
- **前端**: HTML5 + CSS3 + JavaScript
- **模型后端**:
  - Ollama (LLM/Vision)
  - Llama.cpp (GGUF)
  - vLLM (高性能)
  - HuggingFace Inference API
  - Stable Diffusion WebUI API

---

## 扩展开发

### 添加新模型

在 `manager.go` 的 `registerAllModels()` 函数中添加：

```go
{Name: "新模型名", Type: ModelTypeXXX, Backend: BackendXXX, ...}
```

### 添加新后端

1. 在 `types.go` 添加新的 `ModelBackend` 常量
2. 创建新的客户端文件（如 `mybackend.go`）
3. 在 `manager.go` 添加客户端初始化
4. 在 `scheduler.go` 添加任务路由

---

## 许可证

MIT License
