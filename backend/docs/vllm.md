# vLLM 集成指南

vLLM 是 NVIDIA GPU 的高性能推理引擎，支持 PagedAttention、量化等技术，提供高吞吐量的 LLM 推理服务。

## 系统要求

- **GPU**: NVIDIA GPU with CUDA support (至少 8GB VRAM)
- **CUDA**: 11.8 或更高版本
- **Python**: 3.8 - 3.11
- **操作系统**: Linux (Ubuntu 20.04/22.04 推荐)

## 安装 vLLM

### 使用 pip 安装 (推荐)

```bash
# 基础安装
pip install vllm

# 或者安装最新版本
pip install vllm --upgrade

# 验证安装
python -c "import vllm; print(vllm.__version__)"
```

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/vllm-project/vllm.git
cd vllm

# 安装依赖
pip install -e .

# 或者使用 Docker
docker build -t vllm .
```

### Docker 安装 (推荐)

```bash
# 拉取镜像 (选择 CUDA 版本)
docker pull nvidia/cuda:12.1.0-runtime-ubuntu22.04

# 运行 vLLM
docker run --gpus all -p 8000:8000 \
  -v /path/to/models:/models \
  vllm/vllm-openai:latest \
  --model meta-llama/Llama-2-7b-hf
```

## 启动 vLLM Server

### 基本命令

```bash
# 使用 HuggingFace 模型
vllm serve meta-llama/Llama-2-7b-hf

# 指定 GPU 内存利用率
vllm serve meta-llama/Llama-2-7b-hf --gpu-memory-utilization 0.9

# 指定端口
vllm serve meta-llama/Llama-2-7b-hf --port 8000

# 多 GPU 并行
vllm serve meta-llama/Llama-2-70b-hf --tensor-parallel-size 2
```

### 完整示例

```bash
vllm serve meta-llama/Llama-2-7b-hf \
  --port 8000 \
  --host 0.0.0.0 \
  --gpu-memory-utilization 0.9 \
  --max-model-len 4096 \
  --tensor-parallel-size 1
```

### 常用参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--port` | 服务端口 | 8000 |
| `--host` | 监听地址 | 127.0.0.1 |
| `--gpu-memory-utilization` | GPU 内存使用率 | 0.9 |
| `--max-model-len` | 最大序列长度 | 模型支持的最大值 |
| `--tensor-parallel-size` | Tensor 并行数 | 1 |
| `--swap-space` | Swap 空间 (GB) | 4 |
| `--dtype` | 数据类型 | auto |

## LocalMAI 配置

在 `.env` 文件中配置：

```env
VLLM_URL=http://localhost:8000
```

## API 使用

vLLM 提供与 OpenAI 兼容的 API。

### Chat 对话 (OpenAI 兼容)

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-2-7b-hf",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Explain quantum computing"}
    ],
    "temperature": 0.7,
    "max_tokens": 500
  }'
```

### 文本补全

```bash
curl -X POST http://localhost:8000/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-2-7b-hf",
    "prompt": "The future of AI is",
    "max_tokens": 100,
    "temperature": 0.7
  }'
```

### Embeddings

```bash
curl -X POST http://localhost:8000/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-2-7b-hf",
    "input": "The quick brown fox"
  }'
```

### Python 示例

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8000/v1",
    api_key="not-needed"
)

# Chat
response = client.chat.completions.create(
    model="meta-llama/Llama-2-7b-hf",
    messages=[
        {"role": "user", "content": "What is Python?"}
    ]
)
print(response.choices[0].message.content)

# Completion
response = client.completions.create(
    model="meta-llama/Llama-2-7b-hf",
    prompt="def hello_world():"
)
print(response.choices[0].text)
```

## 支持的模型列表

LocalMAI 预注册了以下 vLLM 模型：

| 模型名称 | VRAM 需求 | 描述 |
|---------|----------|------|
| meta-llama/Llama-2-7b-hf | 4GB | LLaMA-2 7B |
| meta-llama/Llama-2-13b-hf | 8GB | LLaMA-2 13B |
| meta-llama/Llama-2-70b-hf | 16GB | LLaMA-2 70B |
| meta-llama/Meta-Llama-3-8B | 4GB | LLaMA-3 8B |
| meta-llama/Meta-Llama-3-70B | 16GB | LLaMA-3 70B |
| mistralai/Mistral-7B-v0.1 | 4GB | Mistral 7B |
| mistralai/Mixtral-8x7B-Instruct-v0.1 | 8GB | Mixtral 8x7B |
| Qwen/Qwen2-7B | 4GB | Qwen2 7B |
| Qwen/Qwen2-72B | 16GB | Qwen2 72B |
| deepseek-ai/DeepSeek-V2 | 8GB | DeepSeek V2 |
| 01-ai/Yi-1.5-34B | 8GB | Yi-1.5 34B |
| microsoft/phi-3-medium | 8GB | Phi-3 Medium |

## 量化支持

vLLM 支持多种量化格式减少显存占用：

```bash
# AWQ 量化 (推荐)
vllm serve meta-llama/Llama-2-7b-hf --quantization awq

# GPTQ 量化
vllm serve meta-llama/Llama-2-7b-hf --quantization gptq

# FP8 量化 (H100/H200)
vllm serve meta-llama/Llama-2-7b-hf --dtype fp8
```

## 多 GPU 部署

### Tensor Parallelism

```bash
# 2 GPU
vllm serve meta-llama/Llama-2-70b-hf \
  --tensor-parallel-size 2 \
  --port 8000

# 4 GPU
vllm serve meta-llama/Llama-2-70b-hf \
  --tensor-parallel-size 4 \
  --port 8000
```

### Pipeline Parallelism

```bash
vllm serve meta-llama/Llama-2-70b-hf \
  --pipeline-parallel-size 2 \
  --tensor-parallel-size 2
```

## 常见问题

### Q: CUDA out of memory

```bash
# 减小 GPU 内存使用率
vllm serve model --gpu-memory-utilization 0.8

# 使用更小的模型
vllm serve meta-llama/Llama-2-7b-hf

# 启用量化
vllm serve model --quantization awq
```

### Q: 如何提升吞吐量

```bash
# 增加 batch size
vllm serve model --max-num-batched-tokens 32768

# 增加并行数
vllm serve model --tensor-parallel-size 2

# 使用更好的量化
vllm serve model --quantization fp8
```

### Q: 模型下载慢

```bash
# 设置 HuggingFace 镜像
export HF_ENDPOINT=https://hf-mirror.com

# 或者使用代理
export HTTPS_PROXY=http://proxy:port

# 然后启动
vllm serve meta-llama/Llama-2-7b-hf
```

### Q: 支持 continuous batching 吗

是的，vLLM 默认启用 continuous batching，无需额外配置。

## 性能对比

| 方案 | 吞吐量 (tokens/s) | 延迟 | 适用场景 |
|------|-------------------|------|----------|
| Ollama | 20-50 | 中 | 简单部署 |
| Llama.cpp | 30-80 | 中 | CPU/GPU 灵活 |
| vLLM | 100-500+ | 低 | 高性能生产 |

## 下一步

- [Llama.cpp 集成](./llamacpp.md) - 轻量级推理
- [Ollama 集成](./ollama.md) - 简单易用
- [HuggingFace 集成](./huggingface.md) - 云端推理
- [API 使用](./api.md) - 完整 API 文档
