# LocalMAI - 多模态 AI 模型服务平台

一个类似 Ollama 的多模态 AI 模型管理平台,支持分布式 Worker、GPU 调度和多种模型后端。

## 功能特性

- **多模态支持**: 文本生成、图像生成、语音合成、视频生成
- **多种后端支持**: Ollama、HuggingFace、Stable Diffusion WebUI
- **分布式架构**: Worker 节点支持分布式任务执行
- **GPU 调度**: 智能 GPU 内存分配和负载均衡
- **RESTful API**: 易于与应用集成
- **Web UI**: 内置可视化界面

## 快速开始

```bash
# 1. 配置环境变量
export HF_TOKEN=your_hf_token_here

# 2. 进入后端目录
cd backend

# 3. 下载依赖
go mod tidy

# 4. 启动服务
go run .

# 5. 打开浏览器访问
# http://localhost:8080
```

## 详细文档

请参阅 [backend/README.md](backend/README.md) 获取完整的:
- 环境配置指南
- API 使用文档
- 模型列表
- 各类型模型详细用法
- 故障排除

## 项目结构

```
localMAI/
├── backend/
│   ├── main.go           # 主入口
│   ├── api.go            # API 处理器
│   ├── manager.go        # 模型管理器
│   ├── scheduler.go      # 任务调度器
│   ├── gpu.go           # GPU 管理器
│   ├── ollama.go         # Ollama 客户端
│   ├── huggingface.go   # HuggingFace 客户端
│   ├── types.go         # 类型定义
│   ├── worker/
│   │   └── worker.go    # Worker 节点
│   ├── frontend/
│   │   └── index.html   # Web UI
│   └── README.md         # 详细文档
└── README.md             # 本文件
```

## 支持的模型类型

| 类型 | 后端 | 示例 |
|------|------|------|
| 文本生成 (LLM) | Ollama | llama3, mistral, qwen2.5 |
| 视觉模型 | Ollama | llava, qwen2-vl |
| 图像生成 | HF/SD WebUI | SDXL, SDXL-Lightning, FLUX |
| 语音合成 | HuggingFace | Bark, FastSpeech2, XTTS |
| 语音识别 | HuggingFace | Whisper Large/Base |
| 视频生成 | HuggingFace | HunyuanVideo, ZeroScope |

## 许可证

MIT License
