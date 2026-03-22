# LTX-Video 集成指南

LTX-Video 是 Lightricks 开发的 DiT-based 图文音视频生成模型，支持文生视频、图生视频、视频编辑和音频生成。

## 模型信息

| 模型 | 类型 | VRAM | 描述 |
|------|------|------|------|
| ltx-video-2.3-dev | text-to-video | 24GB | LTX-Video 2.3 开发版 |
| ltx-video-2.3-distilled | text-to-video | 24GB | LTX-Video 2.3 蒸馏版 (8步) |
| ltx-upscaler-x2 | video-upscaler | 16GB | 空间放大 x2 |
| ltx-upscaler-x1.5 | video-upscaler | 16GB | 空间放大 x1.5 |

**官方主页**: https://huggingface.co/Lightricks/LTX-2.3

## 系统要求

- **GPU**: NVIDIA GPU (推荐 24GB+ VRAM)
- **CUDA**: > 12.7
- **Python**: >= 3.12
- **PyTorch**: ~= 2.7

## 使用方式

LTX-Video **不支持** HuggingFace Inference API，需要通过以下方式使用：

### 方式一：ComfyUI (推荐)

1. 安装 ComfyUI：
```bash
git clone https://github.com/comfyanonymous/ComfyUI.git
cd ComfyUI
pip install -r requirements.txt
```

2. 在 ComfyUI Manager 中搜索并安装 **LTXVideo** 节点

3. 启动 ComfyUI：
```bash
python main.py --force-fp16 --listen 0.0.0.0 --port 8188
```

4. 在 LocalMAI 中配置：
```env
LTX_VIDEO_URL=http://localhost:8188
```

### 方式二：PyTorch 独立运行

```bash
# 克隆官方仓库
git clone https://github.com/Lightricks/LTX-2.git
cd LTX-2

# 安装依赖
uv sync
source .venv/bin/activate

# 查看 ltx-pipelines 包说明
cat packages/ltx-pipelines/README.md
```

## ComfyUI 工作流

### Text-to-Video

```
Text Prompt → LTXVideo Node → Video Output
```

### Image-to-Video

```
Input Image + Text → LTXVideo Node → Video Output
```

### 提示词技巧

1. 视频长度必须是 8 的倍数 + 1 帧 (如: 9, 17, 25, 33...)
2. 分辨率必须是 32 的倍数 (如: 768x512, 1024x768...)
3. 使用描述性强的提示词
4. 参考: https://ltx.video/blog/how-to-prompt-for-ltx-2

## LocalMAI API 使用

### 生成视频

```bash
curl -X POST http://localhost:8080/api/video/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ltx-video-2.3-dev",
    "prompt": "A serene lake at sunset with mountains in the background, cinematic drone shot",
    "width": 768,
    "height": 512,
    "num_frames": 33,
    "steps": 50,
    "cfg_scale": 7.0
  }'
```

### 图片转视频

```bash
curl -X POST http://localhost:8080/api/video/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ltx-video-2.3-dev",
    "prompt": "The scene continues with gentle camera movement",
    "input_image": "data:image/png;base64,...",
    "width": 768,
    "height": 512,
    "num_frames": 33
  }'
```

### Python 示例

```python
import requests
import base64

response = requests.post("http://localhost:8080/api/video/generate", json={
    "model": "ltx-video-2.3-dev",
    "prompt": "A beautiful sunset over the ocean, realistic, 8k",
    "width": 768,
    "height": 512,
    "num_frames": 33,
    "steps": 50
})

task_id = response.json()["task_id"]

# 轮询结果
import time
while True:
    result = requests.get(f"http://localhost:8080/api/tasks/{task_id}").json()
    if result["state"] == "completed":
        print("Video saved to:", result["result"]["output_path"])
        break
    elif result["state"] == "failed":
        print("Error:", result["result"]["error"])
        break
    time.sleep(2)
```

## 参数说明

| 参数 | 类型 | 默认值 | 描述 |
|------|------|--------|------|
| prompt | string | 必需 | 视频描述提示词 |
| input_image | string | 可选 | Base64 编码的输入图片 |
| width | int | 768 | 视频宽度 (需是 32 的倍数) |
| height | int | 512 | 视频高度 (需是 32 的倍数) |
| num_frames | int | 33 | 帧数 (需是 8 的倍数 +1) |
| steps | int | 50 | 推理步数 |
| cfg_scale | float | 7.0 | CFG 引导强度 |

## 常见问题

### Q: LTX-Video API 不可用

确保 ComfyUI 已启动并安装了 LTXVideo 节点：
```bash
curl http://localhost:8188/system_stats
```

### Q: 显存不足 (CUDA out of memory)

- 降低分辨率
- 减少帧数
- 使用 distilled 版本 (8步)
- 需要约 24GB VRAM

### Q: 生成速度慢

- 开发版需要 50+ 步
- 蒸馏版只需 8 步
- 使用 GPU 加速

### Q: 提示词不生效

- LTX-Video 对提示词风格敏感
- 使用描述性语言
- 参考官方提示词指南: https://ltx.video/blog/how-to-prompt-for-ltx-2

## 模型下载

如果需要本地下载模型：

```bash
# 安装 git-lfs
git lfs install

# 克隆模型
git clone https://huggingface.co/Lightricks/LTX-2.3 models/LTX-2.3

# 或使用 huggingface-cli
huggingface-cli download Lightricks/LTX-2.3 --local-dir ./models/LTX-2.3
```

## 与其他方案对比

| 方案 | 视频质量 | 生成速度 | VRAM 需求 | 音频支持 |
|------|----------|----------|-----------|----------|
| LTX-Video 2.3 | ⭐⭐⭐⭐⭐ | 中 | 24GB | ✅ |
| Stable Video Diffusion | ⭐⭐⭐⭐ | 中 | 12GB | ❌ |
| zeroscope | ⭐⭐⭐ | 快 | 8GB | ❌ |
| ModelScope | ⭐⭐⭐ | 慢 | 8GB | ❌ |

## 下一步

- [ComfyUI 集成](./comfyui.md) - ComfyUI 安装配置
- [视频生成](./video-generation.md) - 其他视频模型
- [API 使用](./api.md) - 完整 API 文档
