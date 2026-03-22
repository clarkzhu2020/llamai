# Ollama 集成指南

Ollama 是本地大模型运行的简化方案，支持 LLM 和 Vision 模型。

## 安装 Ollama

### macOS / Linux
```bash
# 安装
curl -fsSL https://ollama.com/install.sh | sh

# 验证安装
ollama --version
```

### Windows
从 [Ollama 官网](https://ollama.com/download) 下载安装包。

### Docker (可选)
```bash
docker pull ollama/ollama
docker run -d -v ollama:/root/.ollama -p 11434:11434 --name ollama ollama/ollama
```

## 启动 Ollama 服务

```bash
# 启动服务 (默认端口 11434)
ollama serve

# 服务会在后台运行
```

## 下载模型

```bash
# LLM 模型
ollama pull llama3              # Meta LLaMA 3 8B
ollama pull llama3.2             # Meta LLaMA 3.2
ollama pull mistral              # Mistral 7B
ollama pull qwen2.5             # Qwen 2.5 7B
ollama pull codellama            # Code LLaMA
ollama pull deepseek-coder       # DeepSeek Coder

# Vision 模型
ollama pull llava                # LLaVA Vision
ollama pull llava-llama3         # LLaVA with LLaMA 3
ollama pull qwen2-vl             # Qwen2 VL Vision
ollama pull llama3.2-vision      # LLaMA 3.2 Vision

# 其他模型
ollama pull gemma2               # Google Gemma 2
ollama pull phi3                 # Microsoft Phi-3
ollama pull mixtral              # Mixtral 8x7B
ollama pull wizardlm2            # WizardLM 2
```

## 查看已下载模型

```bash
ollama list
```

输出示例：
```
NAME                    ID          SIZE      MODIFIED
llama3:latest           365c0fb9b847    4.7GB     2 days ago
llava:latest            8dd30c8b8f06    7.4GB     3 days ago
mistral:latest          2ae6f0e87ef3    4.1GB     1 week ago
```

## LocalMAI 配置

在 `.env` 文件中配置 Ollama 地址：

```env
OLLAMA_BASE_URL=http://localhost:11434
```

默认地址已经是 `http://localhost:11434`，无需额外配置。

## API 使用

### Chat 对话
```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ]
  }'
```

### 文本生成
```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "prompt": "Write a short story about a robot."
  }'
```

### Python 示例
```python
import requests

response = requests.post("http://localhost:8080/api/chat", json={
    "model": "llama3",
    "messages": [
        {"role": "user", "content": "What is machine learning?"}
    ]
})
print(response.json())
```

## 支持的模型列表

LocalMAI 预注册了以下 Ollama 模型：

| 模型名称 | 类型 | VRAM 需求 | 描述 |
|---------|------|----------|------|
| llama3 | LLM | 4GB | Meta LLaMA 3 8B |
| llama3.2 | LLM | 4GB | Meta LLaMA 3.2 |
| llama3.1 | LLM | 4GB | Meta LLaMA 3.1 8B |
| mistral | LLM | 4GB | Mistral 7B |
| mistral-nemo | LLM | 4GB | Mistral Nemo 12B |
| qwen2.5 | LLM | 4GB | Qwen 2.5 7B |
| qwen2.5-coder | LLM | 4GB | Qwen 2.5 Coder 7B |
| codellama | LLM | 4GB | Code LLaMA |
| deepseek-coder | LLM | 4GB | DeepSeek Coder |
| gemma2 | LLM | 4GB | Google Gemma 2 |
| phi3 | LLM | 2GB | Microsoft Phi-3 Mini |
| mixtral | LLM | 4GB | Mixtral 8x7B |
| llava | Vision | 6GB | LLaVA Vision Model |
| llava-llama3 | Vision | 8GB | LLaVA with LLaMA 3 |
| qwen2-vl | Vision | 6GB | Qwen2 VL Vision |
| llama3.2-vision | Vision | 8GB | LLaMA 3.2 Vision |
| moondream | Vision | 4GB | Moondream Vision |

## 常见问题

### Q: Ollama 服务无法启动
```bash
# 检查端口是否被占用
lsof -i :11434

# 重启服务
pkill ollama
ollama serve
```

### Q: 模型下载失败
```bash
# 设置代理
export HTTP_PROXY=http://your-proxy:port
export HTTPS_PROXY=http://your-proxy:port

# 重新下载
ollama pull llama3
```

### Q: 如何自定义模型路径
```bash
# 设置模型存储路径
export OLLAMA_MODELS=/path/to/models

# 重启 ollama
```

### Q: GPU 不被使用
确保已安装 CUDA 驱动：
```bash
nvidia-smi
```

## 下一步

- [Llama.cpp 集成](./llamacpp.md) - 支持 GGUF 格式模型
- [vLLM 集成](./vllm.md) - 高性能推理
- [HuggingFace 集成](./huggingface.md) - Image/Speech/Video 模型
- [API 使用](./api.md) - 完整 API 文档
