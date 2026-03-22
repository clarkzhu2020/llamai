# Llama.cpp 集成指南

Llama.cpp 是一个纯 C/C++ 实现的 LLM 推理框架，支持 GGUF 格式模型，CPU 和 GPU 加速均可。

## 安装 Llama.cpp

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp

# 创建构建目录
mkdir build && cd build

# 配置构建 (Linux/macOS)
cmake .. -DCMAKE_BUILD_TYPE=Release

# Windows (使用 Visual Studio)
# cmake .. -G "Visual Studio 17 2022" -DCMAKE_BUILD_TYPE=Release

# 编译
cmake --build . --config Release

# 编译完成后的可执行文件在 build/bin/ 目录下
```

### 使用预编译二进制

从 [llama.cpp Releases](https://github.com/ggerganov/llama.cpp/releases) 下载对应平台的预编译版本。

### Docker 方式

```bash
# 拉取镜像
docker pull ghcr.io/ggerganov/llama.cpp:server

# 运行
docker run -p 8080:8080 -v /path/to/models:/models \
  ghcr.io/ggerganov/llama.cpp:server \
  -m /models/llama-7b.gguf -c 2048 -fa --host 0.0.0.0 --port 8080
```

## 获取 GGUF 模型

### 从 HuggingFace 下载

HuggingFace 上有大量现成的 GGUF 格式模型：

```bash
# 安装 huggingface-cli
pip install huggingface-hub

# 下载模型
huggingface-cli download TheBloke/Llama-2-7B-GGUF llama-2-7b.Q4_K_M.gguf --local-dir ./models

# 其他常用模型
huggingface-cli download TheBloke/Mistral-7B-Instruct-v0.2-GGUF mistral-7b-instruct-v0.2.Q4_K_M.gguf --local-dir ./models
huggingface-cli download TheBloke/Qwen2-7B-Instruct-GGUF qwen2-7b-instruct.Q4_K_M.gguf --local-dir ./models
```

### 模型推荐

| 模型 | 大小 | 量化 | 推荐场景 |
|------|------|------|----------|
| llama-2-7b.Q4_K_M.gguf | ~4GB | Q4_K_M | 通用对话 |
| mistral-7b-instruct-v0.2.Q4_K_M.gguf | ~4GB | Q4_K_M | 指令跟随 |
| qwen2-7b-instruct.Q4_K_M.gguf | ~4GB | Q4_K_M | 中文对话 |
| phi-3-mini.Q4_K_M.gguf | ~2GB | Q4_K_M | 低资源 |

## 启动 Llama.cpp Server

### 基本命令

```bash
# 进入编译后的目录
cd llama.cpp/build/bin

# 启动 server (CPU)
./llama-server -m models/llama-7b.gguf -c 2048 --host 0.0.0.0 --port 8080

# 启动 server (GPU 加速，CUDA)
./llama-server -m models/llama-7b.gguf -c 2048 -fa -ngl 99 --host 0.0.0.0 --port 8080

# 参数说明
# -m: 模型文件路径
# -c: 上下文长度 (默认 2048)
# -fa: 加速模式 (flash attention)
# -ngl: GPU layers (99 表示全部使用 GPU)
# --host: 监听地址
# --port: 监听端口
```

### 高性能配置

```bash
./llama-server \
  -m models/llama-7b.gguf \
  -c 4096 \           # 更大上下文
  -fa \               # Flash Attention
  -ngl 99 \           # GPU 加速
  -b 512 \            # batch size
  -t 8 \              # 线程数
  --host 0.0.0.0 \
  --port 8080
```

### Metal 加速 (macOS)

```bash
./llama-server -m models/llama-7b.gguf -c 2048 -fa -ngl 99 --host 0.0.0.0 --port 8080
```

## LocalMAI 配置

在 `.env` 文件中配置：

```env
LLAMA_CPP_URL=http://localhost:8080
```

## API 使用

### Chat 对话

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-7b",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "temperature": 0.7,
    "max_tokens": 2048
  }'
```

### 文本补全

```bash
curl -X POST http://localhost:8080/completion \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Once upon a time",
    "temperature": 0.7,
    "max_tokens": 100
  }'
```

### Python 示例

```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed"
)

response = client.chat.completions.create(
    model="llama-7b",
    messages=[{"role": "user", "content": "Write a Python function"}]
)
print(response.choices[0].message.content)
```

## 支持的模型列表

LocalMAI 预注册了以下 Llama.cpp 模型：

| 模型名称 | VRAM 需求 | 描述 |
|---------|----------|------|
| llama-7b | 4GB | LLaMA 7B |
| llama-13b | 8GB | LLaMA 13B |
| llama-70b | 16GB | LLaMA 70B |
| mistral-7b | 4GB | Mistral 7B |
| mixtral-8x7b | 8GB | Mixtral 8x7B |
| qwen2-7b | 4GB | Qwen2 7B |
| qwen2-72b | 16GB | Qwen2 72B |
| yi-6b | 4GB | Yi 6B |
| yi-34b | 8GB | Yi 34B |
| deepseek-7b | 4GB | DeepSeek 7B |
| phi-3-mini | 2GB | Phi-3 Mini |
| gemma-2b | 2GB | Gemma 2B |

## 量化格式说明

GGUF 模型有多种量化格式：

| 格式 | 压缩率 | 质量 | 推荐 |
|------|--------|------|------|
| Q2_K | 87% | 最高 | 最低资源 |
| Q3_K_M | 78% | 高 | 低资源 |
| Q4_0 | 73% | 中 | 平衡 |
| Q4_K_M | 71% | 高 | **推荐** |
| Q5_0 | 65% | 高 | 高质量 |
| Q5_K_M | 58% | 很高 | 高质量需求 |
| Q6_K | 53% | 最高 | 最高质量 |
| Q8_0 | 47% | 几乎无损 | 无资源限制 |

## 常见问题

### Q: CUDA 不可用
```bash
# 确保已安装 CUDA Toolkit
nvcc --version

# 或者使用 CPU 模式
./llama-server -m models/llama-7b.gguf -c 2048 --host 0.0.0.0 --port 8080
```

### Q: 内存不足
```bash
# 使用更小的量化模型
./llama-server -m models/llama-2-7b.Q2_K.gguf -c 2048

# 减小上下文长度
./llama-server -m models/llama-7b.gguf -c 1024
```

### Q: 速度慢
```bash
# 增加线程数
./llama-server -m models/llama-7b.gguf -c 2048 -t 16

# 启用 GPU 加速
./llama-server -m models/llama-7b.gguf -c 2048 -fa -ngl 99
```

### Q: 如何转换模型为 GGUF

使用 llama.cpp 提供的转换脚本：

```bash
# 克隆并准备 transformers 模型
git clone https://github.com/huggingface/transformers.git

# 使用 convert.py 转换
python llama.cpp/convert.py transformers/llama-2-7b/ \
  --outfile models/llama-2-7b.gguf \
  --outtype q4_k_m
```

## 下一步

- [vLLM 集成](./vllm.md) - NVIDIA GPU 高性能推理
- [Ollama 集成](./ollama.md) - 更简单的本地部署
- [HuggingFace 集成](./huggingface.md) - 云端推理
- [API 使用](./api.md) - 完整 API 文档
