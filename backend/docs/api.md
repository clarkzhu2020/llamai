# API 使用文档

LocalMAI 提供统一的 REST API，支持多种后端模型的推理服务。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **认证**: 通过 `X-API-Key` header 或 `api_key` query 参数

## 认证

如果配置了 `LOCALMAI_API_KEY`，需要在请求头中添加：

```bash
curl -X GET http://localhost:8080/api/health \
  -H "X-API-Key: your-api-key"
```

或使用 query 参数：

```bash
curl -X GET "http://localhost:8080/api/health?api_key=your-api-key"
```

## 健康检查

### GET /api/health

检查系统健康状态和可用资源。

```bash
curl http://localhost:8080/api/health
```

响应：
```json
{
  "status": "ok",
  "gpus": [
    {
      "id": 0,
      "name": "NVIDIA GPU 0",
      "total_mb": 16000,
      "used_mb": 0,
      "free_mb": 16000,
      "utilization_percent": 0,
      "memory_percent": 0
    }
  ],
  "workers": [],
  "queue_stats": {
    "pending": 0,
    "running": 0,
    "completed": 0,
    "failed": 0
  },
  "uptime_seconds": 3600
}
```

## 模型管理

### GET /api/models

列出所有可用模型。

```bash
curl http://localhost:8080/api/models
```

按类型筛选：
```bash
curl "http://localhost:8080/api/models?type=llm"
```

### GET /api/models/{name}

获取特定模型详情。

```bash
curl http://localhost:8080/api/models/llama3
```

### POST /api/models/{name}

加载模型。

```bash
curl -X POST http://localhost:8080/api/models/llama3
```

### DELETE /api/models/{name}

卸载模型。

```bash
curl -X DELETE http://localhost:8080/api/models/llama3
```

## 文本生成

### POST /api/generate

文本生成接口。

```bash
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "prompt": "Write a short story about a robot",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 500
    }
  }'
```

响应：
```json
{
  "task_id": "abc123",
  "status": "queued"
}
```

### POST /api/chat

对话生成接口。

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "What is the capital of France?"}
    ],
    "options": {
      "temperature": 0.7
    }
  }'
```

## 任务管理

### GET /api/tasks

获取队列统计。

```bash
curl http://localhost:8080/api/queue/stats
```

### GET /api/tasks/{task_id}

获取任务状态和结果。

```bash
curl http://localhost:8080/api/tasks/abc123
```

响应：
```json
{
  "task": {
    "id": "abc123",
    "type": "text_generation",
    "model": "llama3",
    "input": "Write a short story",
    "priority": 1,
    "created_at": "2024-01-01T00:00:00Z"
  },
  "state": "completed",
  "result": {
    "task_id": "abc123",
    "output": "Once upon a time...",
    "duration_ms": 5000,
    "completed_at": "2024-01-01T00:00:05Z"
  }
}
```

任务状态：
- `pending` - 等待中
- `running` - 执行中
- `completed` - 已完成
- `failed` - 失败

## 图片生成

### POST /api/image/generate

生成图片。

```bash
curl -X POST http://localhost:8080/api/image/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stabilityai/stable-diffusion-xl-base-1.0",
    "prompt": "a beautiful sunset over the ocean",
    "negative_prompt": "blurry, low quality",
    "width": 1024,
    "height": 1024,
    "steps": 30,
    "cfg_scale": 7.5
  }'
```

参数说明：

| 参数 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| model | string | 是 | - | 模型名称 |
| prompt | string | 是 | - | 正向提示词 |
| negative_prompt | string | 否 | "" | 负向提示词 |
| width | int | 否 | 512 | 图片宽度 |
| height | int | 否 | 512 | 图片高度 |
| steps | int | 否 | 30 | 推理步数 |
| cfg_scale | float | 否 | 7.0 | CFG 引导强度 |

### POST /api/image/edit

图片编辑 (img2img)。

```bash
curl -X POST http://localhost:8080/api/image/edit \
  -H "Content-Type: application/json" \
  -d '{
    "model": "stabilityai/stable-diffusion-img2img",
    "prompt": "transform to anime style",
    "input_image": "data:image/png;base64,...",
    "strength": 0.75
  }'
```

参数说明：

| 参数 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| model | string | 是 | - | 模型名称 |
| prompt | string | 是 | - | 提示词 |
| input_image | string | 是 | - | Base64 编码的图片 |
| strength | float | 否 | 0.75 | 变换强度 (0-1) |

## 语音合成

### POST /api/speech/generate

文本转语音。

```bash
curl -X POST http://localhost:8080/api/speech/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "suno/bark",
    "text": "Hello, this is a test of the text to speech system."
  }'
```

## 语音识别

### POST /api/speech/transcribe

语音转文本。

```bash
curl -X POST http://localhost:8080/api/speech/transcribe \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openai/whisper-large",
    "audio": "data:audio/wav;base64,..."
  }'
```

## 视频生成

### POST /api/video/generate

生成视频。

```bash
curl -X POST http://localhost:8080/api/video/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "zeroscope_v2_576w",
    "prompt": "a drone flying over mountains"
  }'
```

图片转视频：
```bash
curl -X POST http://localhost:8080/api/video/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "damo-vilab/image-to-video",
    "prompt": "the scene continues",
    "input_image": "data:image/png;base64,..."
  }'
```

## GPU 监控

### GET /api/gpus

获取 GPU 信息。

```bash
curl http://localhost:8080/api/gpus
```

响应：
```json
[
  {
    "id": 0,
    "name": "NVIDIA GPU 0",
    "total_mb": 16000,
    "used_mb": 0,
    "free_mb": 16000,
    "utilization_percent": 0,
    "memory_percent": 0
  }
]
```

## Worker 管理

### GET /api/workers

列出所有注册的 Workers。

```bash
curl http://localhost:8080/api/workers
```

### POST /api/workers/register

注册新 Worker。

```bash
curl -X POST http://localhost:8080/api/workers/register \
  -H "Content-Type: application/json" \
  -d '{
    "id": "worker-1",
    "address": "192.168.1.100",
    "port": 8081,
    "gpus": [],
    "models": ["llama3", "mistral"]
  }'
```

## 队列统计

### GET /api/queue/stats

获取队列统计信息。

```bash
curl http://localhost:8080/api/queue/stats
```

响应：
```json
{
  "pending": 5,
  "running": 2,
  "completed": 100,
  "failed": 3
}
```

## Python SDK 示例

### 安装

```bash
pip install requests
```

### 文本生成

```python
import requests

# Chat 对话
response = requests.post("http://localhost:8080/api/chat", json={
    "model": "llama3",
    "messages": [
        {"role": "user", "content": "Hello!"}
    ]
})
task_id = response.json()["task_id"]

# 轮询结果
import time
while True:
    result = requests.get(f"http://localhost:8080/api/tasks/{task_id}").json()
    if result["state"] == "completed":
        print(result["result"]["output"])
        break
    elif result["state"] == "failed":
        print("Error:", result["result"]["error"])
        break
    time.sleep(1)
```

### 图片生成

```python
import requests
import base64

# 生成图片
response = requests.post("http://localhost:8080/api/image/generate", json={
    "model": "stabilityai/stable-diffusion-xl-base-1.0",
    "prompt": "a beautiful landscape",
    "width": 1024,
    "height": 1024
})
task_id = response.json()["task_id"]

# 获取结果
result = requests.get(f"http://localhost:8080/api/tasks/{task_id}").json()
if result["state"] == "completed":
    # Base64 图片数据
    image_data = result["result"]["output_data"]
    with open("output.png", "wb") as f:
        f.write(base64.b64decode(image_data))
```

## JavaScript/TypeScript 示例

### 使用 Fetch API

```javascript
// Chat 对话
const response = await fetch("http://localhost:8080/api/chat", {
  method: "POST",
  headers: {"Content-Type": "application/json"},
  body: JSON.stringify({
    model: "llama3",
    messages: [{"role": "user", "content": "Hello!"}]
  })
});
const {task_id} = await response.json();

// 轮询结果
let result;
while (true) {
  const res = await fetch(`http://localhost:8080/api/tasks/${task_id}`);
  result = await res.json();
  if (result.state === "completed") {
    console.log(result.result.output);
    break;
  }
  await new Promise(r => setTimeout(r, 1000));
}
```

### 使用 Node.js

```javascript
const axios = require('axios');

async function generate(prompt) {
  const {data: {task_id}} = await axios.post("http://localhost:8080/api/generate", {
    model: "llama3",
    prompt
  });
  
  while (true) {
    const {data: result} = await axios.get(`http://localhost:8080/api/tasks/${task_id}`);
    if (result.state === "completed") {
      return result.result.output;
    }
    await new Promise(r => setTimeout(r, 1000));
  }
}

generate("Hello world").then(console.log);
```

## 错误响应

错误响应格式：

```json
{
  "error": "error message here"
}
```

常见 HTTP 状态码：

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 (API Key 无效) |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用 (后端未连接) |

## 速率限制

暂无内置速率限制。建议在生产环境中通过 Nginx/API Gateway 配置。

## 下一步

- [Ollama 集成](./ollama.md) - 本地 LLM
- [Llama.cpp 集成](./llamacpp.md) - GGUF 模型
- [vLLM 集成](./vllm.md) - 高性能推理
- [HuggingFace 集成](./huggingface.md) - 云端多模态
- [SD WebUI 集成](./sd-webui.md) - 本地图片生成
