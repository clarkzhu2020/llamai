# 故障排除指南

常见问题及解决方案。

## 启动问题

### 服务无法启动

```
Error: port already in use
```

解决方案：
```bash
# 查找占用端口的进程
lsof -i :8080

# 或使用
netstat -ano | findstr :8080

# 终止进程或修改 .env 中的 PORT
```

### 编译错误

```bash
# 清理并重新下载依赖
go clean -mod
go mod tidy
go build
```

## Ollama 相关

### Ollama 服务不可用

启动时显示：
```
⚠️  Ollama server not detected
```

解决方案：
```bash
# 1. 检查 Ollama 是否运行
curl http://localhost:11434

# 2. 启动 Ollama 服务
ollama serve

# 3. 验证模型已安装
ollama list

# 4. 如未安装，拉取模型
ollama pull llama3
```

### 模型未找到

```
Error: model 'xxx' not found in Ollama
```

解决方案：
```bash
# 列出可用模型
ollama list

# 拉取缺失的模型
ollama pull llama3
ollama pull mistral
```

## Llama.cpp 相关

### Llama.cpp 服务不可用

```
⚠️  Llama.cpp not available at http://localhost:8080
```

解决方案：
```bash
# 1. 编译 llama.cpp
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
mkdir build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release

# 2. 启动 server
./bin/llama-server -m models/llama-7b.gguf -c 2048 --host 0.0.0.0 --port 8080
```

### CUDA 不可用

```
CUDA error: no CUDA capable device found
```

解决方案：
```bash
# 使用 CPU 模式
./llama-server -m models/llama-7b.gguf -c 2048 --host 0.0.0.0 --port 8080

# 或安装 CUDA 驱动
nvidia-smi  # 检查驱动
```

### 内存不足

```
error: failed to allocate CUDA memory
```

解决方案：
```bash
# 使用更小的量化模型
./llama-server -m models/llama-2-7b.Q2_K.gguf -c 2048

# 减小上下文长度
./llama-server -m models/llama-7b.gguf -c 1024
```

## vLLM 相关

### vLLM 服务不可用

```
⚠️  vLLM not available at http://localhost:8000
```

解决方案：
```bash
# 1. 安装 vLLM
pip install vllm

# 2. 启动 server
vllm serve meta-llama/Llama-2-7b-hf --host 0.0.0.0 --port 8000
```

### CUDA out of memory

```
OutOfMemoryError: CUDA out of memory
```

解决方案：
```bash
# 减小 GPU 内存使用
vllm serve model --gpu-memory-utilization 0.8

# 使用更小的模型
vllm serve meta-llama/Llama-2-7b-hf

# 启用量化
vllm serve model --quantization awq
```

### 模型下载失败

```bash
# 设置 HuggingFace 镜像
export HF_ENDPOINT=https://hf-mirror.com

# 或使用代理
export HTTPS_PROXY=http://proxy:port
```

## HuggingFace 相关

### HF_TOKEN 未设置

```
⚠️  HF_TOKEN not set
```

解决方案：
```bash
# 1. 获取 Token: https://huggingface.co/settings/tokens
# 2. 编辑 .env 文件
nano .env
# 添加: HF_TOKEN=hf_your_token

# 3. 重启服务
```

### API 请求失败

```
Error: HuggingFace API error: Access token missing
```

解决方案：
```bash
# 确认 Token 正确
echo $HF_TOKEN

# 检查 Token 权限（需要 Read 权限）
# 访问 https://huggingface.co/settings/tokens
```

### 请求超时

HuggingFace API 有 10 分钟超时限制。

解决方案：
```bash
# 使用更小/更快的模型
stabilityai/stable-diffusion-2-1  # 比 SDXL 快
ByteDance/SDXL-Lightning-4step    # 4 步完成
```

## Stable Diffusion WebUI 相关

### SD WebUI 不可用

```
⚠️  SD WebUI not available at http://localhost:7860
```

解决方案：
```bash
# 1. 克隆仓库
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui

# 2. 启动 API 模式
./webui.sh --api --listen 0.0.0.0 --port 7860

# Windows: 编辑 webui-user.bat
# set COMMANDLINE_ARGS=--api --listen
```

### 显存不足

```
RuntimeError: CUDA out of memory
```

解决方案：
```bash
# 启动时添加参数
./webui.sh --medvram --lowvram

# 或减小图片尺寸
# 1024x1024 -> 512x512
```

### 模型加载失败

```bash
# 检查模型文件
ls -la models/Stable-diffusion/

# 确认文件完整 (应该有 .safetensors 或 .ckpt 扩展名)
# 重新下载模型
```

## 图像生成问题

### 生成结果为空白

可能原因：
1. 提示词不够具体
2. 步数太少
3. 模型问题

解决方案：
```bash
# 增加步数
steps: 50

# 使用更具体的提示词
"a beautiful red sunset over the ocean, photorealistic, 8k"

# 添加 negative_prompt
negative_prompt: "blurry, low quality, watermark"
```

### 图片扭曲变形

解决方案：
```bash
# 使用 Euler a 采样器
sampler: "Euler a"

# 降低 CFG
cfg_scale: 5.0

# 增加步数
steps: 40
```

## API 问题

### 401 Unauthorized

```
Error: Unauthorized
```

解决方案：
```bash
# 如果设置了 API Key，确保请求时包含
curl -H "X-API-Key: your-key" http://localhost:8080/api/health

# 或禁用认证 (不推荐用于生产环境)
# 编辑 .env: LOCALMAI_API_KEY=
```

### 任务一直 pending

可能原因：
1. 队列阻塞
2. 后端服务未连接

解决方案：
```bash
# 检查队列状态
curl http://localhost:8080/api/queue/stats

# 检查后端连接
curl http://localhost:8080/api/health
```

### 返回结果格式错误

确保请求头正确：
```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{"model": "llama3", "prompt": "hello"}'
```

## 性能问题

### 推理速度慢

通用建议：
1. 使用 GPU 加速
2. 使用更小的模型
3. 减少上下文长度
4. 启用量化

### Web UI 响应慢

```bash
# 减少轮询频率
# 前端默认每 500ms 轮询一次

# 增加服务器资源
```

## 日志查看

```bash
# 实时查看日志
go run . 2>&1 | tail -f

# 或重定向到文件
go run . > app.log 2>&1

# 查看错误
grep -i error app.log
```

## 获取帮助

1. 检查服务状态：`curl http://localhost:8080/api/health`
2. 查看后端日志
3. 查阅文档：
   - [API 文档](./api.md)
   - [Ollama 文档](./ollama.md)
   - [Llama.cpp 文档](./llamacpp.md)
   - [vLLM 文档](./vllm.md)
   - [HuggingFace 文档](./huggingface.md)
   - [SD WebUI 文档](./sd-webui.md)

## 报告问题

报告问题时请包含：
1. LocalMAI 版本和日志
2. 后端服务版本
3. 操作系统和环境
4. 复现步骤
5. 错误信息
