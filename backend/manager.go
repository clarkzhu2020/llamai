package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ModelManager handles model loading and management with multiple backends
type ModelManager struct {
	models       map[string]*Model
	ollama       *OllamaClient
	huggingface  *HuggingFaceClient
	llamaCpp     *LlamaCppClient
	vllm         *VLLMClient
	ltxVideo     *LTXVideoClient
	ollamaModels map[string]bool
	mu           sync.RWMutex
}

// NewModelManager creates a new model manager with Ollama and HuggingFace integration
func NewModelManager(ollamaURL, llamaCppURL, vllmURL, ltxVideoURL string) *ModelManager {
	mm := &ModelManager{
		models:       make(map[string]*Model),
		ollama:       NewOllamaClient(ollamaURL),
		huggingface:  NewHuggingFaceClient(""),
		llamaCpp:     NewLlamaCppClient(llamaCppURL),
		vllm:         NewVLLMClient(vllmURL),
		ltxVideo:     NewLTXVideoClient(ltxVideoURL),
		ollamaModels: make(map[string]bool),
	}
	mm.registerAllModels()
	mm.syncWithOllama()
	mm.syncWithVLLM()
	mm.syncWithSDWebUI()
	return mm
}

// syncWithOllama syncs available models with Ollama server
func (mm *ModelManager) syncWithOllama() {
	if !mm.ollama.IsAvailable() {
		log.Print("Warning: Ollama server not available at localhost:11434")
		log.Print("Please start Ollama or install models")
		return
	}

	models, err := mm.ollama.ListModels()
	if err != nil {
		log.Printf("Warning: Failed to list Ollama models: %v", err)
		return
	}

	log.Printf("Connected to Ollama - %d models available", len(models))
	for _, m := range models {
		mm.ollamaModels[m.Name] = true
		log.Printf("  - %s", m.Name)

		// Register actual installed model to mm.models
		modelType := ModelTypeLLM
		if hasVisionFamily(m) {
			modelType = ModelTypeVision
		}

		mm.mu.Lock()
		mm.models[m.Name] = &Model{
			Name:        m.Name,
			Type:        modelType,
			Backend:     BackendOllama,
			Path:        "",
			VRAMReqMB:   estimateVRAM(m),
			Description: formatDescription(m),
		}
		mm.mu.Unlock()
	}
}

func (mm *ModelManager) syncWithVLLM() {
	if !mm.vllm.IsAvailable() {
		return
	}

	models := mm.vllm.ListModels()
	if len(models) == 0 {
		return
	}

	log.Printf("Connected to vLLM - %d models available", len(models))
	for _, name := range models {
		mm.mu.Lock()
		if _, exists := mm.models[name]; !exists {
			mm.models[name] = &Model{
				Name:        name,
				Type:        ModelTypeLLM,
				Backend:     BackendVLLM,
				VRAMReqMB:   estimateVLLMVRAM(name),
				Description: name + " (vLLM)",
			}
			log.Printf("  - %s", name)
		}
		mm.mu.Unlock()
	}
}

func estimateVLLMVRAM(name string) int64 {
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "70b") || strings.Contains(nameLower, "72b") {
		return 16000
	}
	if strings.Contains(nameLower, "30b") || strings.Contains(nameLower, "34b") {
		return 8000
	}
	if strings.Contains(nameLower, "13b") {
		return 8000
	}
	return 4000
}

func (mm *ModelManager) syncWithSDWebUI() {
	sdURL := os.Getenv("SD_WEBUI_URL")
	if sdURL == "" {
		sdURL = "http://localhost:7860"
	}
	if !mm.IsSDWebUIAvailable(sdURL) {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(sdURL + "/sdapi/v1/sd-models")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var sdModels []struct {
		Title string `json:"title"`
		Name  string `json:"model_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sdModels); err != nil {
		return
	}

	if len(sdModels) == 0 {
		return
	}

	log.Printf("Connected to SD WebUI - %d models available", len(sdModels))
	for _, m := range sdModels {
		mm.mu.Lock()
		modelName := m.Title
		if modelName == "" {
			modelName = m.Name
		}
		if _, exists := mm.models[modelName]; !exists {
			mm.models[modelName] = &Model{
				Name:        modelName,
				Type:        ModelTypeImage,
				Backend:     BackendSDWebUI,
				Path:        m.Name,
				VRAMReqMB:   8000,
				Description: modelName + " (SD WebUI)",
			}
			log.Printf("  - %s", modelName)
		}
		mm.mu.Unlock()
	}
}

func hasVisionFamily(m OllamaModel) bool {
	if m.Details.Families != nil {
		visionFamilies := []string{"llava", "clip", "qwen2vl", "qwen2-vl", "llama3.2-vision"}
		for _, f := range m.Details.Families {
			for _, vf := range visionFamilies {
				if strings.Contains(strings.ToLower(f), vf) {
					return true
				}
			}
		}
	}
	return false
}

func estimateVRAM(m OllamaModel) int64 {
	if m.Details.ParameterSize != "" {
		if size, err := parseSizeGB(m.Details.ParameterSize); err == nil {
			return int64(size * 2) // Rough estimate: 2GB per parameter for FP16
		}
	}
	if m.Size > 0 {
		return m.Size / (1024 * 1024 * 1024) // Convert bytes to GB, rough estimate
	}
	return 4000 // Default 4GB
}

func parseSizeGB(sizeStr string) (float64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.ReplaceAll(sizeStr, "B", "")
	sizeStr = strings.ReplaceAll(sizeStr, "b", "")

	multiplier := 1.0
	if strings.HasSuffix(sizeStr, "K") || strings.HasSuffix(sizeStr, "k") {
		multiplier = 1024.0
		sizeStr = strings.TrimSuffix(sizeStr, "K")
		sizeStr = strings.TrimSuffix(sizeStr, "k")
	} else if strings.HasSuffix(sizeStr, "M") || strings.HasSuffix(sizeStr, "m") {
		multiplier = 1024.0 * 1024.0
		sizeStr = strings.TrimSuffix(sizeStr, "M")
		sizeStr = strings.TrimSuffix(sizeStr, "m")
	} else if strings.HasSuffix(sizeStr, "G") || strings.HasSuffix(sizeStr, "g") {
		multiplier = 1024.0 * 1024.0 * 1024.0
		sizeStr = strings.TrimSuffix(sizeStr, "G")
		sizeStr = strings.TrimSuffix(sizeStr, "g")
	}

	sizeStr = strings.TrimSpace(sizeStr)
	if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		return size * multiplier / (1024 * 1024 * 1024), nil
	}
	return 4.0, fmt.Errorf("failed to parse size")
}

func formatDescription(m OllamaModel) string {
	desc := m.Name
	if m.Details.ParameterSize != "" {
		desc = m.Details.ParameterSize + " " + desc
	}
	if m.Details.QuantizationLevel != "" {
		desc += " (" + m.Details.QuantizationLevel + ")"
	}
	return desc
}

// registerAllModels registers all available models from all backends
func (mm *ModelManager) registerAllModels() {
	// ========== OLLAMA MODELS (LLM & Vision) ==========
	ollamaLLM := []Model{
		{Name: "llama3", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Meta Llama 3 8B (via Ollama)"},
		{Name: "llama3.2", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Meta Llama 3.2 (via Ollama)"},
		{Name: "llama3.1", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Meta Llama 3.1 8B (via Ollama)"},
		{Name: "mistral", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Mistral 7B (via Ollama)"},
		{Name: "mistral-nemo", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Mistral Nemo 12B (via Ollama)"},
		{Name: "qwen2.5", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Qwen 2.5 7B (via Ollama)"},
		{Name: "qwen2.5-coder", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Qwen 2.5 Coder 7B (via Ollama)"},
		{Name: "phi3", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 2000, Description: "Microsoft Phi-3 Mini (via Ollama)"},
		{Name: "phi3.5", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 2000, Description: "Microsoft Phi-3.5 Mini (via Ollama)"},
		{Name: "codellama", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Code Llama (via Ollama)"},
		{Name: "deepseek-coder", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "DeepSeek Coder (via Ollama)"},
		{Name: "deepseek-v2", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 8000, Description: "DeepSeek V2 (via Ollama)"},
		{Name: "wizardlm2", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "WizardLM 2 (via Ollama)"},
		{Name: "wizardlm2-7b", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "WizardLM 2 7B (via Ollama)"},
		{Name: "mixtral", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Mixtral 8x7B (via Ollama)"},
		{Name: "gemma2", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Google Gemma 2 (via Ollama)"},
		{Name: "gemma2-9b", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 5000, Description: "Google Gemma 2 9B (via Ollama)"},
		{Name: "nomic-embed-text", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 1000, Description: "Nomic Embed Text (via Ollama)"},
		{Name: "command-r", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Command R (via Ollama)"},
		{Name: "command-r-plus", Type: ModelTypeLLM, Backend: BackendOllama, Path: "", VRAMReqMB: 8000, Description: "Command R+ (via Ollama)"},
		{Name: "llava", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 6000, Description: "LLaVA Vision Model (via Ollama)"},
		{Name: "llava-llama3", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 8000, Description: "LLaVA with Llama 3 (via Ollama)"},
		{Name: "llava-llama3.2", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 8000, Description: "LLaVA with Llama 3.2 (via Ollama)"},
		{Name: "qwen2-vl", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 6000, Description: "Qwen2 VL Vision (via Ollama)"},
		{Name: "llama3.2-vision", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 8000, Description: "Llama 3.2 Vision (via Ollama)"},
		{Name: "moondream", Type: ModelTypeVision, Backend: BackendOllama, Path: "", VRAMReqMB: 4000, Description: "Moondream Vision (via Ollama)"},
	}

	// ========== HUGGINGFACE IMAGE MODELS ==========
	// Text-to-Image models
	hfTextToImage := []Model{
		// Stable Diffusion variants
		{Name: "stabilityai/stable-diffusion-xl-base-1.0", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "stabilityai/stable-diffusion-xl-base-1.0", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "Stable Diffusion XL 1.0 (HF)"},
		{Name: "stabilityai/stable-diffusion-2-1", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "stabilityai/stable-diffusion-2-1", TaskType: "text-to-image", VRAMReqMB: 5000, Description: "Stable Diffusion 2.1 (HF)"},
		{Name: "stabilityai/stable-diffusion-3-medium", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "stabilityai/stable-diffusion-3-medium", TaskType: "text-to-image", VRAMReqMB: 12000, Description: "Stable Diffusion 3 Medium (HF)"},

		// SDXL-Lightning (Ultra-fast, 1-8 steps)
		{Name: "ByteDance/SDXL-Lightning-4step", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "ByteDance/SDXL-Lightning", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "SDXL-Lightning 4-step (Ultra fast!)"},
		{Name: "ByteDance/SDXL-Lightning-2step", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "ByteDance/SDXL-Lightning", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "SDXL-Lightning 2-step (Ultra fast)"},

		// Flux models
		{Name: "black-forest-labs/FLUX.1-dev", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "black-forest-labs/FLUX.1-dev", TaskType: "text-to-image", VRAMReqMB: 16000, Description: "FLUX.1 Dev (HF)"},
		{Name: "black-forest-labs/FLUX.1-schnell", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "black-forest-labs/FLUX.1-schnell", TaskType: "text-to-image", VRAMReqMB: 16000, Description: "FLUX.1 Schnell (HF)"},

		// Playground v2.5
		{Name: "playgroundai/playground-v2.5-1024px-aesthetic", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "playgroundai/playground-v2.5-1024px-aesthetic", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "Playground v2.5 1024px (HF)"},

		// PixArt Sigma
		{Name: "PixArt-X/PixArt-Sigma", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "PixArt-X/PixArt-Sigma", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "PixArt Sigma (HF)"},

		// Kandinsky
		{Name: "kandinsky-community/kandinsky-3", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "kandinsky-community/kandinsky-3", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "Kandinsky 3 (HF)"},
		{Name: "kandinsky-community/kandinsky-2-2", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "kandinsky-community/kandinsky-2-2", TaskType: "text-to-image", VRAMReqMB: 6000, Description: "Kandinsky 2.2 (HF)"},

		// DALL-E alternatives
		{Name: "prompthero/openjourney", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "prompthero/openjourney", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "OpenJourney (HF)"},
		{Name: "nitrosocke/Ghibli-Diffusion", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "nitrosocke/Ghibli-Diffusion", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "Ghibli Diffusion (HF)"},
	}

	// Image-to-Image models
	hfImageToImage := []Model{
		{Name: "stabilityai/stable-diffusion-img2img", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "stabilityai/stable-diffusion-img2img", TaskType: "image-to-image", VRAMReqMB: 5000, Description: "SD Image-to-Image (HF)"},
		{Name: "timbrooks/instructpix2pix", Type: ModelTypeImage, Backend: BackendHuggingFace, HFModelID: "timbrooks/instructpix2pix", TaskType: "image-to-image", VRAMReqMB: 8000, Description: "InstructPix2Pix (HF)"},
	}

	// ========== HUGGINGFACE SPEECH/TTS MODELS ==========
	hfSpeechModels := []Model{
		{Name: "suno/bark", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "suno/bark", TaskType: "text-to-speech", VRAMReqMB: 4000, Description: "Bark TTS (HF)"},
		{Name: "suno/bark-small", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "suno/bark-small", TaskType: "text-to-speech", VRAMReqMB: 2000, Description: "Bark Small TTS (HF)"},
		{Name: "facebook/fastspeech2-en-ljspeech", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "facebook/fastspeech2-en-ljspeech", TaskType: "text-to-speech", VRAMReqMB: 2000, Description: "FastSpeech 2 (HF)"},
		{Name: "microsoft/speecht5_tts", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "microsoft/speecht5_tts", TaskType: "text-to-speech", VRAMReqMB: 2000, Description: "SpeechT5 TTS (HF)"},
		{Name: "espnet/espnet_model", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "espnet/espnet_model", TaskType: "text-to-speech", VRAMReqMB: 3000, Description: "ESPNet TTS (HF)"},
		{Name: "coqui/xtts", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "coqui/xtts", TaskType: "text-to-speech", VRAMReqMB: 4000, Description: "XTTS v2 TTS (HF)"},
		{Name: "plamacharen/Audiobox", Type: ModelTypeSpeech, Backend: BackendHuggingFace, HFModelID: "plamacharen/Audiobox", TaskType: "text-to-speech", VRAMReqMB: 6000, Description: "Audiobox TTS (HF)"},
	}

	// ========== HUGGINGFACE SPEECH RECOGNITION (Whisper) ==========
	hfSpeechRecognition := []Model{
		{Name: "openai/whisper-large-v3", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-large-v3", TaskType: "automatic-speech-recognition", VRAMReqMB: 6000, Description: "Whisper Large V3 (HF)"},
		{Name: "openai/whisper-large-v2", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-large-v2", TaskType: "automatic-speech-recognition", VRAMReqMB: 5000, Description: "Whisper Large V2 (HF)"},
		{Name: "openai/whisper-medium", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-medium", TaskType: "automatic-speech-recognition", VRAMReqMB: 3000, Description: "Whisper Medium (HF)"},
		{Name: "openai/whisper-small", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-small", TaskType: "automatic-speech-recognition", VRAMReqMB: 2000, Description: "Whisper Small (HF)"},
		{Name: "openai/whisper-base", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-base", TaskType: "automatic-speech-recognition", VRAMReqMB: 1000, Description: "Whisper Base (HF)"},
		{Name: "openai/whisper-tiny", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "openai/whisper-tiny", TaskType: "automatic-speech-recognition", VRAMReqMB: 500, Description: "Whisper Tiny (HF)"},
		{Name: "distil-whisper/distil-large-v2", Type: ModelTypeAudio, Backend: BackendHuggingFace, HFModelID: "distil-whisper/distil-large-v2", TaskType: "automatic-speech-recognition", VRAMReqMB: 3000, Description: "Distil-Whisper Large V2 (HF)"},
	}

	// ========== HUGGINGFACE VIDEO MODELS ==========
	hfVideoModels := []Model{
		// Text-to-Video
		{Name: "damo-vilab/text-to-video-ms-1.7b", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "damo-vilab/text-to-video-ms-1.7b", TaskType: "text-to-video", VRAMReqMB: 8000, Description: "Text-to-Video 1.7B (HF)"},
		{Name: "damo-vilab/text-to-video-ms-1.7b-zero", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "damo-vilab/text-to-video-ms-1.7b-zero", TaskType: "text-to-video", VRAMReqMB: 8000, Description: "Text-to-Video Zero (HF)"},
		{Name: "Tencent/HunyuanVideo", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "Tencent/HunyuanVideo", TaskType: "text-to-video", VRAMReqMB: 16000, Description: "HunyuanVideo (HF)"},
		{Name: "LGPL-3.0/OpenMusediffusion", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "LGPL-3.0/OpenMusediffusion", TaskType: "text-to-video", VRAMReqMB: 8000, Description: "OpenMusE Diffusion (HF)"},
		{Name: "ModelScope/Text-to-Video", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "ModelScope/Text-to-Video", TaskType: "text-to-video", VRAMReqMB: 8000, Description: "ModelScope Text-to-Video (HF)"},

		// Image-to-Video
		{Name: "stabilityai/stable-video-diffusion", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "stabilityai/stable-video-diffusion", TaskType: "image-to-video", VRAMReqMB: 12000, Description: "SVD Image-to-Video (HF)"},
		{Name: "zeroscope_v2_576w", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "zeroscope_v2_576w", TaskType: "image-to-video", VRAMReqMB: 8000, Description: "ZeroScope 576w (HF)"},
		{Name: "damo-vilab/image-to-video", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "damo-vilab/image-to-video", TaskType: "image-to-video", VRAMReqMB: 8000, Description: "Image-to-Video (HF)"},

		// Video-to-Video
		{Name: "Camenduru/stable-diffusion-video2video", Type: ModelTypeVideo, Backend: BackendHuggingFace, HFModelID: "Camenduru/stable-diffusion-video2video", TaskType: "video-to-video", VRAMReqMB: 12000, Description: "Video-to-Video (HF)"},
	}

	// ========== LTX-VIDEO MODELS ==========
	// LTX-Video via ComfyUI or direct PyTorch
	ltxVideoModels := []Model{
		{Name: "ltx-video-2.3-dev", Type: ModelTypeVideo, Backend: BackendLTXVideo, HFModelID: "Lightricks/LTX-2.3", TaskType: "text-to-video", VRAMReqMB: 24000, Description: "LTX-Video 2.3 Dev (DiT-based video+audio)"},
		{Name: "ltx-video-2.3-distilled", Type: ModelTypeVideo, Backend: BackendLTXVideo, HFModelID: "Lightricks/LTX-2.3", TaskType: "text-to-video", VRAMReqMB: 24000, Description: "LTX-Video 2.3 Distilled (8 steps)"},

		// Spatial/Temporal Upscalers
		{Name: "ltx-upscaler-x2", Type: ModelTypeVideo, Backend: BackendLTXVideo, HFModelID: "Lightricks/LTX-2.3", TaskType: "video-upscaler-x2", VRAMReqMB: 16000, Description: "LTX Spatial Upscaler x2"},
		{Name: "ltx-upscaler-x1.5", Type: ModelTypeVideo, Backend: BackendLTXVideo, HFModelID: "Lightricks/LTX-2.3", TaskType: "video-upscaler-x1.5", VRAMReqMB: 16000, Description: "LTX Spatial Upscaler x1.5"},
	}

	// ========== SD WebUI LOCAL MODELS ==========
	// These work with Stable Diffusion WebUI API
	sdWebUIModels := []Model{
		{Name: "sd-xl", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "SDXL", TaskType: "text-to-image", VRAMReqMB: 8000, Description: "Stable Diffusion XL (SD WebUI)"},
		{Name: "sd-2.1", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "v2-1_768-ema-pruned", TaskType: "text-to-image", VRAMReqMB: 5000, Description: "Stable Diffusion 2.1 (SD WebUI)"},
		{Name: "sdxl-turbo", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "sdxl-turbo", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "SDXL Turbo (SD WebUI)"},
		{Name: "sd-turbo", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "sd-turbo", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "SD Turbo (SD WebUI)"},
		{Name: "majicmix", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "majicmix", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "MajicMix (SD WebUI)"},
		{Name: "anything-v5", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "anything-v5", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "Anything V5 (SD WebUI)"},
		{Name: "counterfeit", Type: ModelTypeImage, Backend: BackendSDWebUI, Path: "counterfeit", TaskType: "text-to-image", VRAMReqMB: 4000, Description: "Counterfeit (SD WebUI)"},
	}

	// ========== LLAMA.CPP MODELS ==========
	// These work with llama.cpp server (llama-server)
	llamaCppModels := []Model{
		{Name: "llama-7b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/llama-7b.gguf", VRAMReqMB: 4000, Description: "LLaMA 7B (llama.cpp)"},
		{Name: "llama-13b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/llama-13b.gguf", VRAMReqMB: 8000, Description: "LLaMA 13B (llama.cpp)"},
		{Name: "llama-70b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/llama-70b.gguf", VRAMReqMB: 16000, Description: "LLaMA 70B (llama.cpp)"},
		{Name: "mistral-7b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/mistral-7b.gguf", VRAMReqMB: 4000, Description: "Mistral 7B (llama.cpp)"},
		{Name: "mixtral-8x7b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/mixtral-8x7b.gguf", VRAMReqMB: 8000, Description: "Mixtral 8x7B (llama.cpp)"},
		{Name: "qwen2-7b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/qwen2-7b.gguf", VRAMReqMB: 4000, Description: "Qwen2 7B (llama.cpp)"},
		{Name: "qwen2-72b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/qwen2-72b.gguf", VRAMReqMB: 16000, Description: "Qwen2 72B (llama.cpp)"},
		{Name: "yi-6b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/yi-6b.gguf", VRAMReqMB: 4000, Description: "Yi 6B (llama.cpp)"},
		{Name: "yi-34b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/yi-34b.gguf", VRAMReqMB: 8000, Description: "Yi 34B (llama.cpp)"},
		{Name: "deepseek-7b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/deepseek-7b.gguf", VRAMReqMB: 4000, Description: "DeepSeek 7B (llama.cpp)"},
		{Name: "phi-3-mini", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/phi-3-mini.gguf", VRAMReqMB: 2000, Description: "Phi-3 Mini (llama.cpp)"},
		{Name: "gemma-2b", Type: ModelTypeLLM, Backend: BackendLlamaCpp, Path: "models/gemma-2b.gguf", VRAMReqMB: 2000, Description: "Gemma 2B (llama.cpp)"},
	}

	// ========== VLLM MODELS ==========
	// These work with vLLM server
	vllmModels := []Model{
		{Name: "meta-llama/Llama-2-7b-hf", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 4000, Description: "LLaMA-2 7B (vLLM)"},
		{Name: "meta-llama/Llama-2-13b-hf", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 8000, Description: "LLaMA-2 13B (vLLM)"},
		{Name: "meta-llama/Llama-2-70b-hf", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 16000, Description: "LLaMA-2 70B (vLLM)"},
		{Name: "meta-llama/Meta-Llama-3-8B", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 4000, Description: "LLaMA-3 8B (vLLM)"},
		{Name: "meta-llama/Meta-Llama-3-70B", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 16000, Description: "LLaMA-3 70B (vLLM)"},
		{Name: "mistralai/Mistral-7B-v0.1", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 4000, Description: "Mistral 7B (vLLM)"},
		{Name: "mistralai/Mixtral-8x7B-Instruct-v0.1", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 8000, Description: "Mixtral 8x7B (vLLM)"},
		{Name: "Qwen/Qwen2-7B", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 4000, Description: "Qwen2 7B (vLLM)"},
		{Name: "Qwen/Qwen2-72B", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 16000, Description: "Qwen2 72B (vLLM)"},
		{Name: "deepseek-ai/DeepSeek-V2", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 8000, Description: "DeepSeek V2 (vLLM)"},
		{Name: "01-ai/Yi-1.5-34B", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 8000, Description: "Yi-1.5 34B (vLLM)"},
		{Name: "microsoft/phi-3-medium", Type: ModelTypeLLM, Backend: BackendVLLM, Path: "", VRAMReqMB: 8000, Description: "Phi-3 Medium (vLLM)"},
	}

	// Register all models
	allModels := make([]Model, 0)
	allModels = append(allModels, ollamaLLM...)
	allModels = append(allModels, hfTextToImage...)
	allModels = append(allModels, hfImageToImage...)
	allModels = append(allModels, hfSpeechModels...)
	allModels = append(allModels, hfSpeechRecognition...)
	allModels = append(allModels, hfVideoModels...)
	allModels = append(allModels, sdWebUIModels...)
	allModels = append(allModels, llamaCppModels...)
	allModels = append(allModels, vllmModels...)
	allModels = append(allModels, ltxVideoModels...)

	for i := range allModels {
		mm.models[allModels[i].Name] = &allModels[i]
	}
}

// GetModel returns a model by name
func (mm *ModelManager) GetModel(name string) *Model {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.models[name]
}

// GetModelByBackend returns all models for a specific backend
func (mm *ModelManager) GetModelByBackend(backend ModelBackend) []Model {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]Model, 0)
	for _, m := range mm.models {
		if m.Backend == backend {
			models = append(models, *m)
		}
	}
	return models
}

// IsOllamaAvailable returns whether Ollama server is connected
func (mm *ModelManager) IsOllamaAvailable() bool {
	return mm.ollama.IsAvailable()
}

// IsModelAvailableInOllama checks if model is available
func (mm *ModelManager) IsModelAvailableInOllama(name string) bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.ollamaModels[name]
}

// GetOllamaModelNames returns list of all available Ollama model names
func (mm *ModelManager) GetOllamaModelNames() []string {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	names := make([]string, 0, len(mm.ollamaModels))
	for name := range mm.ollamaModels {
		names = append(names, name)
	}
	return names
}

// IsHuggingFaceAvailable checks if HuggingFace API is available
func (mm *ModelManager) IsHuggingFaceAvailable() bool {
	return mm.huggingface.IsAvailable()
}

// IsSDWebUIAvailable checks if Stable Diffusion WebUI is available
func (mm *ModelManager) IsSDWebUIAvailable(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url + "/sdapi/v1/options")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ListModels returns all registered models
func (mm *ModelManager) ListModels() []Model {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]Model, 0, len(mm.models))
	for _, m := range mm.models {
		models = append(models, *m)
	}
	return models
}

// ListModelsByType returns models filtered by type
func (mm *ModelManager) ListModelsByType(modelType ModelType) []Model {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]Model, 0)
	for _, m := range mm.models {
		if m.Type == modelType {
			models = append(models, *m)
		}
	}
	return models
}

// ListModelsByBackend returns models filtered by backend
func (mm *ModelManager) ListModelsByBackend(backend ModelBackend) []Model {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	models := make([]Model, 0)
	for _, m := range mm.models {
		if m.Backend == backend {
			models = append(models, *m)
		}
	}
	return models
}

// RegisterModel adds a new model
func (mm *ModelManager) RegisterModel(model *Model) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.models[model.Name]; exists {
		return fmt.Errorf("model already exists: %s", model.Name)
	}
	mm.models[model.Name] = model
	return nil
}

// UnregisterModel removes a model
func (mm *ModelManager) UnregisterModel(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if _, exists := mm.models[name]; !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	delete(mm.models, name)
	return nil
}

// GetOllamaClient returns the Ollama client
func (mm *ModelManager) GetOllamaClient() *OllamaClient {
	return mm.ollama
}

// GetHuggingFaceClient returns the HuggingFace client
func (mm *ModelManager) GetHuggingFaceClient() *HuggingFaceClient {
	return mm.huggingface
}

// GetLlamaCppClient returns the Llama.cpp client
func (mm *ModelManager) GetLlamaCppClient() *LlamaCppClient {
	return mm.llamaCpp
}

// GetVLLMClient returns the vLLM client
func (mm *ModelManager) GetVLLMClient() *VLLMClient {
	return mm.vllm
}

// GetLTXVideoClient returns the LTX-Video client
func (mm *ModelManager) GetLTXVideoClient() *LTXVideoClient {
	return mm.ltxVideo
}

// IsLlamaCppAvailable returns whether Llama.cpp server is available
func (mm *ModelManager) IsLlamaCppAvailable() bool {
	return mm.llamaCpp.IsAvailable()
}

// IsVLLMAvailable returns whether vLLM server is available
func (mm *ModelManager) IsVLLMAvailable() bool {
	return mm.vllm.IsAvailable()
}

// IsLTXVideoAvailable returns whether LTX-Video server is available
func (mm *ModelManager) IsLTXVideoAvailable() bool {
	return mm.ltxVideo.IsAvailable()
}

// RefreshOllamaModels refreshes the list of models from Ollama
func (mm *ModelManager) RefreshOllamaModels() {
	mm.syncWithOllama()
}

// LoadModel marks a model as loaded
func (mm *ModelManager) LoadModel(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[name]
	if !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	model.Loaded = true
	return nil
}

// UnloadModel marks a model as unloaded
func (mm *ModelManager) UnloadModel(name string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	model, exists := mm.models[name]
	if !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	model.Loaded = false
	return nil
}
