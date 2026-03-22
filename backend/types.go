package main

import "time"

// TaskType defines the type of AI task
type TaskType string

const (
	TaskTypeTextGeneration   TaskType = "text_generation"
	TaskTypeImageGeneration  TaskType = "image_generation"
	TaskTypeImageEdit        TaskType = "image_edit"
	TaskTypeAudioGeneration  TaskType = "audio_generation"
	TaskTypeSpeechGeneration TaskType = "speech_generation"
	TaskTypeAudioTranscribe  TaskType = "audio_transcribe"
	TaskTypeVideoGeneration  TaskType = "video_generation"
	TaskTypeImageToVideo     TaskType = "image_to_video"
	TaskTypeVideoToVideo     TaskType = "video_to_video"
)

// ModelType defines the type of model
type ModelType string

const (
	ModelTypeLLM    ModelType = "llm"
	ModelTypeVision ModelType = "vision"
	ModelTypeImage  ModelType = "image"
	ModelTypeAudio  ModelType = "audio"
	ModelTypeVideo  ModelType = "video"
	ModelTypeSpeech ModelType = "speech"
)

// ModelBackend defines where the model runs
type ModelBackend string

const (
	BackendOllama      ModelBackend = "ollama"
	BackendHuggingFace ModelBackend = "huggingface"
	BackendSDWebUI     ModelBackend = "sd_webui"
	BackendLlamaCpp    ModelBackend = "llamacpp"
	BackendVLLM        ModelBackend = "vllm"
	BackendLTXVideo    ModelBackend = "ltx_video"
	BackendExternal    ModelBackend = "external"
)

// Model represents an AI model configuration
type Model struct {
	Name        string       `json:"name"`
	Type        ModelType    `json:"type"`
	Backend     ModelBackend `json:"backend"`
	Path        string       `json:"path"`
	VRAMReqMB   int64        `json:"vram_required_mb"`
	Loaded      bool         `json:"loaded"`
	LoadTime    time.Time    `json:"load_time,omitempty"`
	Description string       `json:"description"`
	// HuggingFace specific
	HFModelID string `json:"hf_model_id,omitempty"`
	TaskType  string `json:"task_type,omitempty"` // "text-to-image", "text-generation", etc.
}

// Task represents an AI task request
type Task struct {
	ID         string         `json:"id"`
	Type       TaskType       `json:"type"`
	Model      string         `json:"model"`
	Input      string         `json:"input"`
	InputFile  string         `json:"input_file,omitempty"` // Base64 or path for image input
	Parameters map[string]any `json:"parameters,omitempty"`
	WorkerID   string         `json:"worker_id,omitempty"`
	Priority   int            `json:"priority"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Result represents an AI task result
type Result struct {
	TaskID      string    `json:"task_id"`
	Output      string    `json:"output,omitempty"`
	OutputPath  string    `json:"output_path,omitempty"`
	OutputData  string    `json:"output_data,omitempty"` // Base64 encoded output
	ContentType string    `json:"content_type,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    int64     `json:"duration_ms"`
	CompletedAt time.Time `json:"completed_at"`
}

// GPUInfo represents GPU information
type GPUInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	TotalMB  int64  `json:"total_mb"`
	UsedMB   int64  `json:"used_mb"`
	FreeMB   int64  `json:"free_mb"`
	UtilRate int    `json:"utilization_percent"`
	MemRate  int    `json:"memory_percent"`
}

// WorkerInfo represents a worker node
type WorkerInfo struct {
	ID            string    `json:"id"`
	Address       string    `json:"address"`
	Port          int       `json:"port"`
	GPUs          []GPUInfo `json:"gpus"`
	Models        []string  `json:"models"`
	Load          float64   `json:"load"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// QueueStats represents task queue statistics
type QueueStats struct {
	Pending   int `json:"pending"`
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// HealthResponse represents system health status
type HealthResponse struct {
	Status       string        `json:"status"`
	GPUs         []GPUInfo     `json:"gpus"`
	Workers      []*WorkerInfo `json:"workers"`
	QueueStats   QueueStats    `json:"queue_stats"`
	Uptime       int64         `json:"uptime_seconds"`
	Backends     BackendStatus `json:"backends"`
	OllamaModels []string      `json:"ollama_models,omitempty"` // Models actually installed in Ollama
}

// BackendStatus represents which backends are available
type BackendStatus struct {
	Ollama      bool `json:"ollama"`
	LlamaCpp    bool `json:"llamacpp"`
	VLLM        bool `json:"vllm"`
	HuggingFace bool `json:"huggingface"`
	SDWebUI     bool `json:"sdwebui"`
	LTXVideo    bool `json:"ltxvideo"`
}

// GenerateRequest for text/image/audio generation
type GenerateRequest struct {
	Model      string         `json:"model"`
	Prompt     string         `json:"prompt"`
	InputImage string         `json:"input_image,omitempty"` // Base64 encoded
	Parameters map[string]any `json:"parameters,omitempty"`
}

// GenerateResponse for generation tasks
type GenerateResponse struct {
	TaskID     string `json:"task_id"`
	Status     string `json:"status"`
	OutputPath string `json:"output_path,omitempty"`
}

// ModelListResponse for listing models
type ModelListResponse struct {
	Models []Model `json:"models"`
}

// ImageGenerateRequest for image generation
type ImageGenerateRequest struct {
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	Width          int     `json:"width,omitempty"`
	Height         int     `json:"height,omitempty"`
	Steps          int     `json:"steps,omitempty"`
	CFGScale       float64 `json:"cfg_scale,omitempty"`
}

// SpeechGenerateRequest for text-to-speech
type SpeechGenerateRequest struct {
	Model string `json:"model"`
	Text  string `json:"text"`
}

// VideoGenerateRequest for video generation
type VideoGenerateRequest struct {
	Model      string `json:"model"`
	Prompt     string `json:"prompt"`
	InputImage string `json:"input_image,omitempty"` // For image-to-video
	NumFrames  int    `json:"num_frames,omitempty"`
}
