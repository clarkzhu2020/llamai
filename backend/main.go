package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("📁 No .env file found, using environment variables")
	} else {
		log.Println("✅ Loaded configuration from .env")
	}

	// Get configuration
	ollamaURL := getEnv("OLLAMA_BASE_URL", "http://localhost:11434")
	llamaCppURL := getEnv("LLAMA_CPP_URL", "http://localhost:8080")
	vllmURL := getEnv("VLLM_URL", "http://localhost:8000")
	ltxVideoURL := getEnv("LTX_VIDEO_URL", "http://localhost:8188")
	port := getEnv("PORT", "8080")

	log.Println("╔════════════════════════════════════════════════════════════════╗")
	log.Println("║                  LocalMAI - Multimodal AI Platform             ║")
	log.Println("║                                                                ║")
	log.Println("║  Supported Backends:                                          ║")
	log.Println("║  • Ollama - LLM & Vision models                               ║")
	log.Println("║  • Llama.cpp - GGUF format LLM models                       ║")
	log.Println("║  • vLLM - High-throughput LLM inference                       ║")
	log.Println("║  • HuggingFace - Image, Speech, Video models                  ║")
	log.Println("║  • Stable Diffusion WebUI - Image generation                   ║")
	log.Println("║  • LTX-Video - Video + Audio generation (DiT-based)           ║")
	log.Println("╚════════════════════════════════════════════════════════════════╝")

	// Initialize components with Ollama connection
	modelManager := NewModelManager(ollamaURL, llamaCppURL, vllmURL, ltxVideoURL)
	gpuManager := NewGPUManager()
	scheduler := NewScheduler(gpuManager, modelManager)

	// Detect real GPUs or use simulated
	gpuCount := gpuManager.DetectGPUs()
	if gpuCount > 0 {
		log.Printf("✅ Detected %d NVIDIA GPU(s)", gpuCount)
	} else {
		simulatedMemory := getEnvInt("GPU_MEMORY_MB", 16000)
		gpuManager.AddSimulatedGPU(0, "Simulated GPU", int64(simulatedMemory))
		log.Printf("📊 No NVIDIA GPU detected, using simulated GPU with %d MB", simulatedMemory)
	}

	// Check Ollama status
	if modelManager.IsOllamaAvailable() {
		log.Println("✅ Connected to Ollama server")
	} else {
		log.Println("⚠️  Ollama server not detected (text/chat will show errors)")
		log.Println("   Install Ollama: https://ollama.com")
	}

	// Check HuggingFace configuration
	hfToken := os.Getenv("HF_TOKEN")
	if hfToken != "" {
		log.Println("✅ HuggingFace API token configured")
		if modelManager.IsHuggingFaceAvailable() {
			log.Println("✅ HuggingFace API is accessible")
		} else {
			log.Println("⚠️  HuggingFace API not accessible, check network")
		}
	} else {
		log.Println("⚠️  HF_TOKEN not set (HuggingFace inference disabled)")
		log.Println("   Set HF_TOKEN in .env for image/speech/video generation")
	}

	// Check Stable Diffusion WebUI configuration
	sdURL := os.Getenv("SD_WEBUI_URL")
	if sdURL == "" {
		sdURL = "http://localhost:7860"
	}
	if modelManager.IsSDWebUIAvailable(sdURL) {
		log.Printf("✅ Stable Diffusion WebUI connected at %s", sdURL)
	} else {
		log.Printf("⚠️  SD WebUI not available at %s", sdURL)
		log.Println("   Install SD WebUI: https://github.com/AUTOMATIC1111/stable-diffusion-webui")
		log.Println("   Start with: ./webui.sh --api --listen 0.0.0.0 --port 7860")
	}

	// Check Llama.cpp configuration
	if modelManager.IsLlamaCppAvailable() {
		log.Printf("✅ Llama.cpp server connected at %s", llamaCppURL)
	} else {
		log.Printf("⚠️  Llama.cpp not available at %s", llamaCppURL)
		log.Println("   Install llama.cpp: https://github.com/ggerganov/llama.cpp")
		log.Println("   Start server: ./llama-server -m models/llama-7b.gguf -c 2048 -fa")
	}

	// Check vLLM configuration
	if modelManager.IsVLLMAvailable() {
		log.Printf("✅ vLLM server connected at %s", vllmURL)
	} else {
		log.Printf("⚠️  vLLM not available at %s", vllmURL)
		log.Println("   Install vLLM: https://docs.vllm.ai/en/latest/getting_started/installation.html")
		log.Println("   Start server: vllm serve meta-llama/Llama-2-7b-hf --gpu-memory-utilization 0.9")
	}

	// Check LTX-Video configuration
	if modelManager.IsLTXVideoAvailable() {
		log.Printf("✅ LTX-Video server connected at %s", ltxVideoURL)
	} else {
		log.Printf("⚠️  LTX-Video not available at %s", ltxVideoURL)
		log.Println("   Install LTX-Video via ComfyUI: https://github.com/comfyanonymous/ComfyUI")
		log.Println("   Or use LTX-Video standalone: https://github.com/Lightricks/LTX-2")
	}

	// Count models by type
	models := modelManager.ListModels()
	llmCount := len(modelManager.ListModelsByType(ModelTypeLLM))
	visionCount := len(modelManager.ListModelsByType(ModelTypeVision))
	imageCount := len(modelManager.ListModelsByType(ModelTypeImage))
	speechCount := len(modelManager.ListModelsByType(ModelTypeSpeech))
	videoCount := len(modelManager.ListModelsByType(ModelTypeVideo))

	log.Printf("📦 Loaded %d models total:", len(models))
	log.Printf("   • %d LLM models (Ollama/llama.cpp/vLLM)", llmCount)
	log.Printf("   • %d Vision models (Ollama)", visionCount)
	log.Printf("   • %d Image models (HF/SD WebUI)", imageCount)
	log.Printf("   • %d Speech/TTS models (HF)", speechCount)
	log.Printf("   • %d Video models (HF)", videoCount)

	// Show available generation backends
	availableBackends := []string{}
	if modelManager.IsOllamaAvailable() {
		availableBackends = append(availableBackends, "Ollama (LLM/Vision)")
	}
	if modelManager.IsLlamaCppAvailable() {
		availableBackends = append(availableBackends, "Llama.cpp (GGUF)")
	}
	if modelManager.IsVLLMAvailable() {
		availableBackends = append(availableBackends, "vLLM (High-throughput)")
	}
	if hfToken != "" && modelManager.IsHuggingFaceAvailable() {
		availableBackends = append(availableBackends, "HuggingFace (Image/Speech/Video)")
	}
	if modelManager.IsSDWebUIAvailable(sdURL) {
		availableBackends = append(availableBackends, "SD WebUI (Image)")
	}

	if len(availableBackends) > 0 {
		log.Printf("🎯 Active backends: %s", joinStrings(availableBackends, ", "))
	} else {
		log.Println("⚠️  No generation backends available!")
	}

	// Start scheduler
	scheduler.Start()
	log.Println("Task scheduler started")

	// Setup API handler
	apiHandler := NewAPIHandler(modelManager, scheduler, gpuManager)
	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	go func() {
		log.Printf("\n🚀 Server listening on http://localhost:%s", port)
		log.Printf("🌐 Web UI: http://localhost:%s", port)
		log.Printf("📖 API Docs: http://localhost:%s/api/health", port)
		log.Println("")
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("\nShutting down...")
	scheduler.Stop()
	log.Println("Goodbye!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
