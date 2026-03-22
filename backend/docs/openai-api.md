# OpenAI 兼容 API

LocalMAI 提供与 OpenAI API 完全兼容的接口，支持直接使用 OpenAI SDK 或任何兼容应用。

## 启动

默认端口 `8081`，可通过 `OPENAI_API_PORT` 配置：

```env
OPENAI_API_PORT=8081
```

## 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/v1/models` | GET | 列出所有可用模型 |
| `/v1/chat/completions` | POST | 聊天补全 |
| `/v1/completions` | POST | 文本补全 |
| `/v1/embeddings` | POST | 向量嵌入 |
| `/v1/images/generations` | POST | 图片生成 |

## 使用 OpenAI SDK

### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8081/v1",
    api_key="not-needed"
)

# Chat
response = client.chat.completions.create(
    model="llama3",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)

# Completion
response = client.completions.create(
    model="llama3",
    prompt="The capital of France is"
)
print(response.choices[0].text)

# Image
response = client.images.generate(
    model="stabilityai/stable-diffusion-xl-base-1.0",
    prompt="A beautiful sunset"
)
print(response.data[0].url)
```

### JavaScript/TypeScript

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8081/v1',
  apiKey: 'not-needed'
});

// Chat
const chat = await client.chat.completions.create({
  model: 'llama3',
  messages: [{role: 'user', content: 'Hello!'}]
});

// Completion
const completion = await client.completions.create({
  model: 'llama3',
  prompt: 'The capital of France is'
});
```

### LangChain

```python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    base_url="http://localhost:8081/v1",
    api_key="not-needed",
    model="llama3"
)
response = llm.invoke("Hello!")
```

### LlamaIndex

```python
from llama_index.core import VectorStoreIndex, SimpleDirectoryReader
from llama_index.llms.openai_like import OpenAILike

llm = OpenAILike(
    model="llama3",
    api_base="http://localhost:8081/v1",
    api_key="not-needed"
)
```

## curl 示例

### Chat

```bash
curl http://localhost:8081/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Completion

```bash
curl http://localhost:8081/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3",
    "prompt": "The capital of France is"
  }'
```

### List Models

```bash
curl http://localhost:8081/v1/models
```

## 支持的模型

所有 LocalMAI 注册的模型都可通过 OpenAI API 访问：

### LLM 模型

```python
# Ollama
client = OpenAI(base_url="http://localhost:8081/v1", api_key="not-needed")
client.chat.completions.create(model="llama3", ...)
client.chat.completions.create(model="mistral", ...)

# Llama.cpp
client.chat.completions.create(model="llama-7b", ...)

# vLLM
client.chat.completions.create(model="meta-llama/Llama-2-7b-hf", ...)
```

### Image 模型

```python
client.images.generate(
    model="stabilityai/stable-diffusion-xl-base-1.0",
    prompt="A beautiful landscape"
)
```

## 应用集成

### SillyTavern

1. 设置 API 类型为 `OpenAI`
2. API URL: `http://localhost:8081/v1`
3. API Key: `not-needed`

### LibreChat

1. 选择 OpenAI Compatible API
2. API Endpoint: `http://localhost:8081/v1`

### OpenWebUI

1. 设置 Admin Panel → Settings → OpenAI
2. API Base: `http://localhost:8081/v1`
3. API Key: `not-needed`

### AnythingLLM

1. 选择 OpenAI Compatible
2. Base URL: `http://localhost:8081/v1`
3. API Key: `not-needed`

## 限制

- Embeddings 目前返回简单模拟向量（建议使用专业 Embedding 服务）
- Streaming 暂不支持
- 部分 OpenAI 参数可能被忽略

## 下一步

- [API 文档](./api.md) - 完整 REST API
- [模型列表](./README.md#模型列表) - 支持的模型
