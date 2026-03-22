package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatServer struct {
	modelManager *ModelManager
	scheduler    *Scheduler
	port         string
}

type OpenAIModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int    `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type ChatCompletionRequest struct {
	Model            string        `json:"model"`
	Messages         []ChatMessage `json:"messages"`
	Temperature      float64       `json:"temperature,omitempty"`
	MaxTokens        int           `json:"max_tokens,omitempty"`
	TopP             float64       `json:"top_p,omitempty"`
	FrequencyPenalty float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64       `json:"presence_penalty,omitempty"`
	Stop             []string      `json:"stop,omitempty"`
	Stream           bool          `json:"stream,omitempty"`
	User             string        `json:"user,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type CompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Suffix      string   `json:"suffix,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	TopP        float64  `json:"top_p,omitempty"`
	N           int      `json:"n,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	LogProbs    int      `json:"logprobs,omitempty"`
	Echo        bool     `json:"echo,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type CompletionChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
}

type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type EmbeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbeddingsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewOpenAICompatServer(mgr *ModelManager, sched *Scheduler, port string) *OpenAICompatServer {
	return &OpenAICompatServer{
		modelManager: mgr,
		scheduler:    sched,
		port:         port,
	}
}

func (s *OpenAICompatServer) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)
	mux.HandleFunc("/v1/completions", s.handleCompletions)
	mux.HandleFunc("/v1/embeddings", s.handleEmbeddings)
	mux.HandleFunc("/v1/images/generations", s.handleImageGenerations)

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	addr := ":" + s.port
	log.Printf("🌐 OpenAI Compatible API listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *OpenAICompatServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"object":  "service_info",
			"name":    "LocalMAI OpenAI Compatible API",
			"version": "1.0.0",
		})
		return
	}
	http.NotFound(w, r)
}

func (s *OpenAICompatServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (s *OpenAICompatServer) handleModels(w http.ResponseWriter, r *http.Request) {
	models := s.modelManager.ListModels()

	response := OpenAIModelsResponse{
		Object: "list",
		Data: make([]struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		}, 0, len(models)),
	}

	for _, m := range models {
		owner := "localmai"
		switch m.Backend {
		case BackendOllama:
			owner = "ollama"
		case BackendHuggingFace:
			owner = "huggingface"
		case BackendLlamaCpp:
			owner = "llamacpp"
		case BackendVLLM:
			owner = "vllm"
		case BackendLTXVideo:
			owner = "ltx-video"
		case BackendSDWebUI:
			owner = "sd-webui"
		}

		response.Data = append(response.Data, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int    `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      m.Name,
			Object:  "model",
			Created: int(time.Now().Unix()),
			OwnedBy: owner,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *OpenAICompatServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	if len(req.Messages) == 0 {
		http.Error(w, "messages is required", http.StatusBadRequest)
		return
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	messagesJSON, _ := json.Marshal(req.Messages)
	task := &Task{
		ID:    fmt.Sprintf("chat-%d", time.Now().UnixNano()),
		Type:  TaskTypeTextGeneration,
		Model: req.Model,
		Input: string(messagesJSON),
		Parameters: map[string]any{
			"temperature": temperature,
			"max_tokens":  maxTokens,
			"top_p":       req.TopP,
		},
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := s.scheduler.Submit(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit task: %v", err), http.StatusInternalServerError)
		return
	}

	response := <-s.waitForResult(task.ID)

	assistantMessage := ""
	if response.Error != "" {
		assistantMessage = fmt.Sprintf("Error: %s", response.Error)
	} else {
		assistantMessage = response.Output
	}

	completionID := fmt.Sprintf("chatcmpl-%s", task.ID[:8])
	chatResp := ChatCompletionResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: assistantMessage,
				},
				FinishReason: "stop",
			},
		},
	}
	chatResp.Usage.TotalTokens = len(req.Messages)*10 + len(assistantMessage)/4

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

func (s *OpenAICompatServer) handleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 256
	}

	task := &Task{
		ID:    fmt.Sprintf("comp-%d", time.Now().UnixNano()),
		Type:  TaskTypeTextGeneration,
		Model: req.Model,
		Input: req.Prompt,
		Parameters: map[string]any{
			"temperature": temperature,
			"max_tokens":  maxTokens,
			"top_p":       req.TopP,
		},
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := s.scheduler.Submit(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit task: %v", err), http.StatusInternalServerError)
		return
	}

	response := <-s.waitForResult(task.ID)

	text := ""
	if response.Error != "" {
		text = fmt.Sprintf("Error: %s", response.Error)
	} else {
		text = response.Output
	}

	completionID := fmt.Sprintf("completion-%s", task.ID[:8])
	compResp := CompletionResponse{
		ID:      completionID,
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []CompletionChoice{
			{
				Text:         text,
				Index:        0,
				FinishReason: "stop",
			},
		},
	}
	compResp.Usage.TotalTokens = len(req.Prompt)/4 + len(text)/4

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(compResp)
}

func (s *OpenAICompatServer) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EmbeddingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "nomic-embed-text"
	}

	response := EmbeddingsResponse{
		Object: "list",
		Data: make([]struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}, len(req.Input)),
		Model: model,
	}

	for i, text := range req.Input {
		embedding := s.generateSimpleEmbedding(text)
		response.Data[i] = struct {
			Object    string    `json:"object"`
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}{
			Object:    "embedding",
			Embedding: embedding,
			Index:     i,
		}
	}
	response.Usage.TotalTokens = len(req.Input) * 10

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *OpenAICompatServer) handleImageGenerations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		N              int    `json:"n,omitempty"`
		Size           string `json:"size,omitempty"`
		ResponseFormat string `json:"response_format,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = "stabilityai/stable-diffusion-xl-base-1.0"
	}
	if req.N == 0 {
		req.N = 1
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}

	width := 1024
	height := 1024
	if strings.Contains(req.Size, "512") {
		width = 512
		height = 512
	} else if strings.Contains(req.Size, "768") {
		width = 768
		height = 768
	}

	task := &Task{
		ID:    fmt.Sprintf("img-%d", time.Now().UnixNano()),
		Type:  TaskTypeImageGeneration,
		Model: req.Model,
		Input: req.Prompt,
		Parameters: map[string]any{
			"width":  width,
			"height": height,
		},
		Priority:  1,
		CreatedAt: time.Now(),
	}

	if err := s.scheduler.Submit(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to submit task: %v", err), http.StatusInternalServerError)
		return
	}

	response := <-s.waitForResult(task.ID)

	imageResp := map[string]any{
		"created": time.Now().Unix(),
		"model":   req.Model,
		"data": []map[string]any{
			{
				"url":      response.OutputPath,
				"b64_json": response.OutputData,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(imageResp)
}

func (s *OpenAICompatServer) waitForResult(taskID string) <-chan *Result {
	resultChan := make(chan *Result, 1)

	go func() {
		for i := 0; i < 300; i++ {
			task := s.scheduler.GetTask(taskID)
			if task != nil && task.Result != nil {
				resultChan <- task.Result
				return
			}
			time.Sleep(1 * time.Second)
		}
		resultChan <- &Result{
			TaskID: taskID,
			Error:  "timeout waiting for result",
		}
	}()

	return resultChan
}

func (s *OpenAICompatServer) generateSimpleEmbedding(text string) []float64 {
	embedding := make([]float64, 384)
	for i, c := range text {
		if i >= 384 {
			break
		}
		embedding[i] = float64(c)/255.0*2 - 1
	}
	return embedding
}

func (s *OpenAICompatServer) proxyRequest(ctx context.Context, backendURL string, method string, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, backendURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Minute}
	return client.Do(req)
}
