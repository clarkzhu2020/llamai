package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Worker represents a distributed worker node
type Worker struct {
	ID         string
	Address    string
	Port       int
	ManagerURL string
	models     []string
	gpus       []GPUInfo
	load       float64
	status     string
}

// GPUInfo from main package
type GPUInfo struct {
	ID      int   `json:"id"`
	TotalMB int64 `json:"total_mb"`
	UsedMB  int64 `json:"used_mb"`
	FreeMB  int64 `json:"free_mb"`
}

// WorkerInfo for registration
type WorkerInfo struct {
	ID      string    `json:"id"`
	Address string    `json:"address"`
	Port    int       `json:"port"`
	GPUs    []GPUInfo `json:"gpus"`
	Models  []string  `json:"models"`
	Load    float64   `json:"load"`
}

// Task from main package
type Task struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Model string `json:"model"`
	Input string `json:"input"`
}

// Result from main package
type Result struct {
	TaskID string `json:"task_id"`
	Output string `json:"output"`
}

func main() {
	log.Println("Starting LocalMAI Worker Node")
	log.Println("==============================")

	// Worker configuration
	worker := &Worker{
		ID:         uuid.New().String(),
		Address:    getOutboundIP(),
		Port:       9001,
		ManagerURL: "http://localhost:8080",
		models: []string{
			"llama3", "mistral", "qwen2.5",
			"stable-diffusion-xl", "whisper-large",
		},
		status: "online",
	}

	// Detect GPUs (simulated)
	worker.gpus = []GPUInfo{
		{ID: 0, TotalMB: 16000, UsedMB: 0, FreeMB: 16000},
	}
	log.Printf("Worker ID: %s", worker.ID)
	log.Printf("Address: %s:%d", worker.Address, worker.Port)
	log.Printf("Models: %v", worker.models)

	// Register with manager
	if err := worker.register(); err != nil {
		log.Printf("Warning: Could not register with manager: %v", err)
		log.Println("Worker will run in standalone mode")
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/health", worker.handleHealth)
	mux.HandleFunc("/execute", worker.handleExecute)
	mux.HandleFunc("/models", worker.handleModels)
	mux.HandleFunc("/gpu", worker.handleGPU)

	// Start heartbeat
	go worker.heartbeat()

	// Start server
	addr := fmt.Sprintf(":%d", worker.Port)
	log.Printf("Worker listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func (w *Worker) register() error {
	workerInfo := WorkerInfo{
		ID:      w.ID,
		Address: w.Address,
		Port:    w.Port,
		GPUs:    w.gpus,
		Models:  w.models,
		Load:    w.load,
	}

	body, err := json.Marshal(workerInfo)
	if err != nil {
		return err
	}

	resp, err := http.Post(w.ManagerURL+"/api/workers/register", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed: %d", resp.StatusCode)
	}

	log.Println("Successfully registered with manager")
	return nil
}

func (w *Worker) heartbeat() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		if w.status != "online" {
			continue
		}
		// Update load
		w.load = 0.5 // Simulated load
	}
}

func (w *Worker) handleHealth(wr http.ResponseWriter, r *http.Request) {
	json.NewEncoder(wr).Encode(map[string]any{
		"status":  w.status,
		"id":      w.ID,
		"load":    w.load,
		"models":  w.models,
	})
}

func (w *Worker) handleExecute(wr http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(wr, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Executing task %s: %s (%s)", task.ID, task.Type, task.Model)

	// Simulate task execution
	time.Sleep(time.Duration(100+time.Now().UnixNano()%900) * time.Millisecond)

	result := Result{
		TaskID: task.ID,
		Output: fmt.Sprintf("Processed by worker %s: %s", w.ID, task.Input),
	}

	json.NewEncoder(wr).Encode(result)
}

func (w *Worker) handleModels(wr http.ResponseWriter, r *http.Request) {
	json.NewEncoder(wr).Encode(map[string][]string{
		"models": w.models,
	})
}

func (w *Worker) handleGPU(wr http.ResponseWriter, r *http.Request) {
	json.NewEncoder(wr).Encode(w.gpus)
}

func getOutboundIP() string {
	// Connect to a remote server to determine local IP
	conn, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Body.Close()
	ip, _ := io.ReadAll(conn.Body)
	return string(ip)
}
