package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// APIHandler handles HTTP API requests
type APIHandler struct {
	manager   *ModelManager
	scheduler *Scheduler
	gpuMgr    *GPUManager
	startTime time.Time
	apiKey    string
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(mgr *ModelManager, sched *Scheduler, gpu *GPUManager) *APIHandler {
	apiKey := os.Getenv("LOCALMAI_API_KEY")
	return &APIHandler{
		manager:   mgr,
		scheduler: sched,
		gpuMgr:    gpu,
		startTime: time.Now(),
		apiKey:    apiKey,
	}
}

func (h *APIHandler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			providedKey = r.URL.Query().Get("api_key")
		}
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(h.apiKey)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RegisterRoutes registers all API routes
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	// Health check - no auth required
	mux.HandleFunc("/api/health", h.handleHealth)

	// Protected API endpoints
	authHandler := h.requireAuth

	// Model endpoints
	mux.HandleFunc("/api/models", authHandler(h.handleModels))
	mux.HandleFunc("/api/models/", authHandler(h.handleModelByName))

	// Task endpoints
	mux.HandleFunc("/api/tasks", authHandler(h.handleTasks))
	mux.HandleFunc("/api/tasks/", authHandler(h.handleTaskByID))

	// Generation endpoints
	mux.HandleFunc("/api/generate", authHandler(h.handleGenerate))
	mux.HandleFunc("/api/chat", authHandler(h.handleChat))

	// Media generation endpoints
	mux.HandleFunc("/api/image/generate", authHandler(h.handleImageGenerate))
	mux.HandleFunc("/api/image/edit", authHandler(h.handleImageEdit))
	mux.HandleFunc("/api/speech/generate", authHandler(h.handleSpeechGenerate))
	mux.HandleFunc("/api/speech/transcribe", authHandler(h.handleSpeechTranscribe))
	mux.HandleFunc("/api/video/generate", authHandler(h.handleVideoGenerate))

	// GPU endpoints
	mux.HandleFunc("/api/gpus", authHandler(h.handleGPUs))

	// Worker endpoints
	mux.HandleFunc("/api/workers", authHandler(h.handleWorkers))
	mux.HandleFunc("/api/workers/register", authHandler(h.handleWorkerRegister))

	// Queue stats
	mux.HandleFunc("/api/queue/stats", authHandler(h.handleQueueStats))

	// Web UI static files - no auth required
	mux.HandleFunc("/", h.handleStatic)
}

func (h *APIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:     "ok",
		GPUs:       h.gpuMgr.GetInfo(),
		Workers:    h.scheduler.GetWorkers(),
		QueueStats: h.scheduler.GetStats(),
		Uptime:     int64(time.Since(h.startTime).Seconds()),
	}
	jsonResponse(w, http.StatusOK, resp)
}

func (h *APIHandler) handleModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		modelType := r.URL.Query().Get("type")
		if modelType != "" {
			models := h.manager.ListModelsByType(ModelType(modelType))
			jsonResponse(w, http.StatusOK, ModelListResponse{Models: models})
		} else {
			models := h.manager.ListModels()
			jsonResponse(w, http.StatusOK, ModelListResponse{Models: models})
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) handleModelByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/models/")
	model := h.manager.GetModel(name)

	if model == nil {
		http.Error(w, "Model not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, model)
	case http.MethodPost:
		// Load model
		if err := h.manager.LoadModel(name); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, "Failed to load model", err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "loaded"})
	case http.MethodDelete:
		// Unload model
		if err := h.manager.UnloadModel(name); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, "Failed to unload model", err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "unloaded"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	taskID := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	task := h.scheduler.GetTask(taskID)

	if task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, task)
}

func (h *APIHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	model := h.manager.GetModel(req.Model)
	if model == nil {
		http.Error(w, "Model not found", http.StatusNotFound)
		return
	}

	taskType := TaskTypeTextGeneration
	switch model.Type {
	case ModelTypeImage:
		taskType = TaskTypeImageGeneration
	case ModelTypeAudio:
		taskType = TaskTypeAudioGeneration
	case ModelTypeVideo:
		taskType = TaskTypeVideoGeneration
	case ModelTypeVision:
		taskType = TaskTypeImageEdit
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:         taskID,
		Type:       taskType,
		Model:      req.Model,
		Input:      req.Prompt,
		Parameters: req.Parameters,
		Priority:   1,
		CreatedAt:  time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// List tasks - for now just return queue stats
		stats := h.scheduler.GetStats()
		jsonResponse(w, http.StatusOK, stats)
	case http.MethodPost:
		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		taskID := uuid.New().String()
		task := &Task{
			ID:         taskID,
			Type:       TaskTypeTextGeneration,
			Model:      req.Model,
			Input:      req.Prompt,
			Parameters: req.Parameters,
			Priority:   1,
			CreatedAt:  time.Now(),
		}

		if err := h.scheduler.Submit(task); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
			return
		}

		jsonResponse(w, http.StatusAccepted, GenerateResponse{
			TaskID: taskID,
			Status: "queued",
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream  bool           `json:"stream"`
		Options map[string]any `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert messages to JSON string for the task
	messagesJSON, _ := json.Marshal(req.Messages)

	taskID := uuid.New().String()
	task := &Task{
		ID:         taskID,
		Type:       TaskTypeTextGeneration,
		Model:      req.Model,
		Input:      string(messagesJSON), // Store messages as JSON
		Parameters: req.Options,
		Priority:   1,
		CreatedAt:  time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleImageGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model          string  `json:"model"`
		Prompt         string  `json:"prompt"`
		NegativePrompt string  `json:"negative_prompt,omitempty"`
		Width          int     `json:"width,omitempty"`
		Height         int     `json:"height,omitempty"`
		Steps          int     `json:"steps,omitempty"`
		CFGScale       float64 `json:"cfg_scale,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Set defaults
	if req.Width == 0 {
		req.Width = 512
	}
	if req.Height == 0 {
		req.Height = 512
	}
	if req.Steps == 0 {
		req.Steps = 30
	}
	if req.CFGScale == 0 {
		req.CFGScale = 7.0
	}

	params := map[string]any{
		"negative_prompt": req.NegativePrompt,
		"width":           req.Width,
		"height":          req.Height,
		"steps":           req.Steps,
		"cfg_scale":       req.CFGScale,
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:         taskID,
		Type:       TaskTypeImageGeneration,
		Model:      req.Model,
		Input:      req.Prompt,
		Parameters: params,
		Priority:   1,
		CreatedAt:  time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleSpeechGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SpeechGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:        taskID,
		Type:      TaskTypeSpeechGeneration,
		Model:     req.Model,
		Input:     req.Text,
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleVideoGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VideoGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	taskType := TaskTypeVideoGeneration
	if req.InputImage != "" {
		taskType = TaskTypeImageToVideo
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:        taskID,
		Type:      taskType,
		Model:     req.Model,
		Input:     req.Prompt,
		InputFile: req.InputImage,
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleImageEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model      string  `json:"model"`
		Prompt     string  `json:"prompt"`
		InputImage string  `json:"input_image"`
		Strength   float64 `json:"strength,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.InputImage == "" {
		http.Error(w, "input_image is required for image editing", http.StatusBadRequest)
		return
	}

	if req.Strength == 0 {
		req.Strength = 0.75
	}

	params := map[string]any{
		"strength": req.Strength,
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:         taskID,
		Type:       TaskTypeImageEdit,
		Model:      req.Model,
		Input:      req.Prompt,
		InputFile:  req.InputImage,
		Parameters: params,
		Priority:   1,
		CreatedAt:  time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleSpeechTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
		Audio string `json:"audio"` // Base64 encoded
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Audio == "" {
		http.Error(w, "audio field is required", http.StatusBadRequest)
		return
	}

	taskID := uuid.New().String()
	task := &Task{
		ID:        taskID,
		Type:      TaskTypeAudioTranscribe,
		Model:     req.Model,
		Input:     "", // Will be set by scheduler
		InputFile: req.Audio,
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := h.scheduler.Submit(task); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to submit task", err)
		return
	}

	jsonResponse(w, http.StatusAccepted, GenerateResponse{
		TaskID: taskID,
		Status: "queued",
	})
}

func (h *APIHandler) handleGPUs(w http.ResponseWriter, r *http.Request) {
	gpus := h.gpuMgr.GetInfo()
	jsonResponse(w, http.StatusOK, gpus)
}

func (h *APIHandler) handleWorkers(w http.ResponseWriter, r *http.Request) {
	workers := h.scheduler.GetWorkers()
	jsonResponse(w, http.StatusOK, workers)
}

func (h *APIHandler) handleWorkerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var worker WorkerInfo
	if err := json.NewDecoder(r.Body).Decode(&worker); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	h.scheduler.RegisterWorker(&worker)
	jsonResponse(w, http.StatusCreated, map[string]string{"status": "registered"})
}

func (h *APIHandler) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	stats := h.scheduler.GetStats()
	jsonResponse(w, http.StatusOK, stats)
}

func (h *APIHandler) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Try to read from actual frontend directory first
	if data, err := readFrontendFile(path); err == nil {
		ext := filepath.Ext(path)
		contentType := getContentType(ext)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// Fallback to embedded filesystem
	content, err := fs.ReadFile(path)
	if err != nil {
		// Fallback to embedded index.html
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fallbackContent, _ := fs.ReadFile("/index.html")
		io.WriteString(w, fallbackContent)
		return
	}

	ext := filepath.Ext(path)
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, content)
}

func readFrontendFile(path string) ([]byte, error) {
	// Try actual filesystem first
	filePath := filepath.Join("frontend", path)
	return os.ReadFile(filePath)
}

func getContentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".woff", ".woff2":
		return "font/woff2"
	default:
		return "text/plain"
	}
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func logError(r *http.Request, err error) {
	log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err)
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	if err != nil {
		logError(r, err)
	}
	http.Error(w, message, status)
}

// fs is a simple embedded file system
type embeddedFS map[string]string

func (fs embeddedFS) ReadFile(name string) (string, error) {
	if content, ok := fs[name]; ok {
		return content, nil
	}
	return "", fmt.Errorf("file not found")
}

var fs = embeddedFS{
	"/index.html": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LocalMAI - Multimodal AI Platform</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f0f23; color: #fff; min-height: 100vh; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        header { text-align: center; padding: 40px 0; border-bottom: 1px solid #333; margin-bottom: 30px; }
        h1 { font-size: 2.5em; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
        .subtitle { color: #888; margin-top: 10px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: #1a1a2e; border-radius: 12px; padding: 20px; border: 1px solid #333; }
        .card h3 { color: #667eea; margin-bottom: 15px; font-size: 1.2em; }
        .card p, .card li { color: #ccc; line-height: 1.6; }
        .card ul { list-style: none; padding-left: 0; }
        .card li { padding: 5px 0; border-bottom: 1px solid #333; }
        .card li:last-child { border-bottom: none; }
        .badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 0.8em; margin-left: 8px; }
        .badge-llm { background: #667eea; }
        .badge-vision { background: #764ba2; }
        .badge-image { background: #f093fb; color: #000; }
        .badge-audio { background: #4facfe; color: #000; }
        .badge-video { background: #43e97b; color: #000; }
        .api-section { margin-top: 30px; background: #1a1a2e; border-radius: 12px; padding: 20px; }
        .api-section h2 { color: #667eea; margin-bottom: 20px; }
        .endpoint { background: #0f0f23; padding: 15px; border-radius: 8px; margin: 10px 0; font-family: monospace; }
        .method { color: #4facfe; font-weight: bold; }
        .path { color: #ccc; }
        button { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; margin: 5px; }
        button:hover { opacity: 0.9; }
        input, select, textarea { width: 100%; padding: 10px; border-radius: 6px; border: 1px solid #333; background: #0f0f23; color: #fff; margin: 5px 0; }
        textarea { min-height: 100px; resize: vertical; }
        #output { background: #0f0f23; padding: 20px; border-radius: 8px; min-height: 100px; white-space: pre-wrap; margin-top: 20px; }
        .status { display: flex; gap: 20px; margin: 20px 0; }
        .status-item { background: #1a1a2e; padding: 15px 25px; border-radius: 8px; text-align: center; }
        .status-value { font-size: 2em; font-weight: bold; color: #667eea; }
        .status-label { color: #888; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>LocalMAI</h1>
            <p class="subtitle">Multimodal AI Model Management Platform</p>
        </header>

        <div class="status">
            <div class="status-item">
                <div class="status-value" id="gpu-count">0</div>
                <div class="status-label">GPUs</div>
            </div>
            <div class="status-item">
                <div class="status-value" id="worker-count">0</div>
                <div class="status-label">Workers</div>
            </div>
            <div class="status-item">
                <div class="status-value" id="model-count">0</div>
                <div class="status-label">Models</div>
            </div>
        </div>

        <div class="grid">
            <div class="card">
                <h3>Text Generation</h3>
                <ul>
                    <li>llama3 <span class="badge badge-llm">LLM</span></li>
                    <li>mistral <span class="badge badge-llm">LLM</span></li>
                    <li>qwen2.5 <span class="badge badge-llm">LLM</span></li>
                    <li>codellama <span class="badge badge-llm">LLM</span></li>
                </ul>
            </div>
            <div class="card">
                <h3>Vision Models</h3>
                <ul>
                    <li>llava <span class="badge badge-vision">Vision</span></li>
                    <li>qwen-vl <span class="badge badge-vision">Vision</span></li>
                    <li>llama3-vision <span class="badge badge-vision">Vision</span></li>
                </ul>
            </div>
            <div class="card">
                <h3>Image Generation</h3>
                <ul>
                    <li>stable-diffusion-xl <span class="badge badge-image">Image</span></li>
                    <li>sd-turbo <span class="badge badge-image">Image</span></li>
                    <li>sdxl-lightning <span class="badge badge-image">Image</span></li>
                </ul>
            </div>
            <div class="card">
                <h3>Audio Models</h3>
                <ul>
                    <li>whisper-large <span class="badge badge-audio">Audio</span></li>
                    <li>bark <span class="badge badge-audio">Audio</span></li>
                    <li>vall-e <span class="badge badge-audio">Audio</span></li>
                </ul>
            </div>
            <div class="card">
                <h3>Video Models</h3>
                <ul>
                    <li>zeroscope <span class="badge badge-video">Video</span></li>
                    <li>modelscope-t2v <span class="badge badge-video">Video</span></li>
                </ul>
            </div>
            <div class="card">
                <h3>GPU Resources</h3>
                <ul id="gpu-list">
                    <li>No GPUs detected</li>
                </ul>
            </div>
        </div>

        <div class="api-section">
            <h2>Quick Generate</h2>
            <select id="model-select">
                <option value="llama3">LLM - llama3</option>
                <option value="mistral">LLM - mistral</option>
                <option value="qwen2.5">LLM - qwen2.5</option>
                <option value="stable-diffusion-xl">Image - stable-diffusion-xl</option>
                <option value="whisper-large">Audio - whisper-large</option>
            </select>
            <textarea id="prompt" placeholder="Enter your prompt..."></textarea>
            <div>
                <button onclick="generate()">Generate</button>
                <button onclick="clearOutput()">Clear</button>
            </div>
            <div id="output">Output will appear here...</div>
        </div>

        <div class="api-section">
            <h2>API Endpoints</h2>
            <div class="endpoint"><span class="method">GET</span> <span class="path">/api/health</span> - Health check</div>
            <div class="endpoint"><span class="method">GET</span> <span class="path">/api/models</span> - List all models</div>
            <div class="endpoint"><span class="method">POST</span> <span class="path">/api/generate</span> - Generate content</div>
            <div class="endpoint"><span class="method">POST</span> <span class="path">/api/chat</span> - Chat completion</div>
            <div class="endpoint"><span class="method">GET</span> <span class="path">/api/gpus</span> - GPU info</div>
            <div class="endpoint"><span class="method">GET</span> <span class="path">/api/queue/stats</span> - Queue statistics</div>
        </div>
    </div>

    <script>
        const API_BASE = window.location.origin;

        async function loadStats() {
            try {
                const res = await fetch(API_BASE + '/api/health');
                const data = await res.json();
                document.getElementById('gpu-count').textContent = data.gpus?.length || 0;
                document.getElementById('worker-count').textContent = data.workers?.length || 0;
                
                // Update GPU list
                const gpuList = document.getElementById('gpu-list');
                if (data.gpus && data.gpus.length > 0) {
                    gpuList.innerHTML = data.gpus.map(g => 
                        '<li>' + g.name + ': ' + g.free_mb + '/' + g.total_mb + ' MB free</li>'
                    ).join('');
                }
            } catch (e) {
                console.log('API not ready');
            }
        }

        async function loadModels() {
            try {
                const res = await fetch(API_BASE + '/api/models');
                const data = await res.json();
                document.getElementById('model-count').textContent = data.models?.length || 0;
            } catch (e) {
                console.log('API not ready');
            }
        }

        async function generate() {
            const model = document.getElementById('model-select').value;
            const prompt = document.getElementById('prompt').value;
            const output = document.getElementById('output');

            output.textContent = 'Generating...';

            try {
                const res = await fetch(API_BASE + '/api/generate', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ model, prompt })
                });
                const data = await res.json();
                output.textContent = 'Task ID: ' + data.task_id + '\nStatus: ' + data.status;
                
                // Poll for result
                pollResult(data.task_id);
            } catch (e) {
                output.textContent = 'Error: ' + e.message;
            }
        }

        async function pollResult(taskId) {
            const output = document.getElementById('output');
            for (let i = 0; i < 30; i++) {
                await new Promise(r => setTimeout(r, 500));
                try {
                    const res = await fetch(API_BASE + '/api/tasks/' + taskId);
                    const task = await res.json();
                    if (task.state === 'completed') {
                        output.textContent = task.Task.Input + '\n\n---\n\n' + (task.Result?.Output || 'No output');
                        return;
                    }
                } catch (e) {}
            }
            output.textContent += '\n\n(Timed out waiting for result)';
        }

        function clearOutput() {
            document.getElementById('output').textContent = 'Output will appear here...';
        }

        loadStats();
        loadModels();
        setInterval(loadStats, 5000);
    </script>
</body>
</html>`,
}
