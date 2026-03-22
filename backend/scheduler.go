package main

import (
	"container/heap"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const (
	defaultTaskTimeout   = 30 * time.Minute
	defaultResultChanCap = 100
)

// Scheduler manages task scheduling and execution with multiple backends
type Scheduler struct {
	tasks        map[string]*ScheduledTask
	taskQueue    *TaskHeap
	workers      map[string]*WorkerInfo
	gpuManager   *GPUManager
	modelManager *ModelManager

	mu             sync.RWMutex
	cond           *sync.Cond
	ctx            context.Context
	cancel         context.CancelFunc
	resultChan     chan *Result
	pendingCount   int
	runningCount   int
	completedCount int
	failedCount    int
}

// TaskState represents the state of a task
type TaskState int

const (
	TaskPending TaskState = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

// ScheduledTask wraps a task with scheduling metadata
type ScheduledTask struct {
	Task      *Task     `json:"task"`
	State     TaskState `json:"state"`
	WorkerID  string    `json:"worker_id,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Result    *Result   `json:"result,omitempty"`
}

// TaskHeapItem implements heap.Interface and holds a Task with its index
type TaskHeapItem struct {
	Task  *Task
	Index int
}

// TaskHeap implements heap.Interface for priority queue
type TaskHeap []*TaskHeapItem

func (h TaskHeap) Len() int { return len(h) }

func (h TaskHeap) Less(i, j int) bool {
	return h[i].Task.Priority > h[j].Task.Priority // Higher priority first
}

func (h TaskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *TaskHeap) Push(x any) {
	item := x.(*TaskHeapItem)
	item.Index = len(*h)
	*h = append(*h, item)
}

func (h *TaskHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[:n-1]
	return item
}

// NewScheduler creates a new scheduler
func NewScheduler(gpu *GPUManager, mm *ModelManager) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Scheduler{
		tasks:        make(map[string]*ScheduledTask),
		taskQueue:    &TaskHeap{},
		workers:      make(map[string]*WorkerInfo),
		gpuManager:   gpu,
		modelManager: mm,
		ctx:          ctx,
		cancel:       cancel,
		resultChan:   make(chan *Result, defaultResultChanCap),
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Submit adds a new task to the queue
func (s *Scheduler) Submit(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d-%d", time.Now().UnixNano(), len(s.tasks))
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	st := &ScheduledTask{
		Task:  task,
		State: TaskPending,
	}
	s.tasks[task.ID] = st
	heap.Push(s.taskQueue, &TaskHeapItem{Task: task})
	s.pendingCount++
	s.cond.Signal()

	return nil
}

// Start begins the scheduler processing loop
func (s *Scheduler) Start() {
	go s.processLoop()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cancel()
}

// GetStats returns queue statistics
func (s *Scheduler) GetStats() QueueStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return QueueStats{
		Pending:   s.pendingCount,
		Running:   s.runningCount,
		Completed: s.completedCount,
		Failed:    s.failedCount,
	}
}

// GetTask returns task status
func (s *Scheduler) GetTask(taskID string) *ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tasks[taskID]
}

// GetResults returns the result channel
func (s *Scheduler) GetResults() <-chan *Result {
	return s.resultChan
}

func (s *Scheduler) processLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.mu.Lock()
			if s.taskQueue.Len() == 0 {
				s.cond.Wait()
			}
			s.mu.Unlock()
			s.processNext()
		}
	}
}

func (s *Scheduler) processNext() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.taskQueue.Len() == 0 {
		return
	}

	// Get highest priority task
	item := heap.Pop(s.taskQueue).(*TaskHeapItem)
	task := item.Task
	st, ok := s.tasks[task.ID]
	if !ok {
		return
	}

	// Check if task can be scheduled
	model := s.modelManager.GetModel(task.Model)
	if model == nil {
		result := &Result{
			TaskID: st.Task.ID,
			Error:  fmt.Errorf("model not found: %s", task.Model).Error(),
		}
		s.failTask(st, result, time.Now())
		return
	}

	// Allocate GPU for non-Ollama tasks
	if model.Backend != BackendOllama {
		_, ok := s.gpuManager.Allocate(task.ID, model.VRAMReqMB)
		if !ok {
			// Re-add to queue if no GPU available
			heap.Push(s.taskQueue, &TaskHeapItem{Task: task})
			return
		}
	}

	// Start task
	st.State = TaskRunning
	st.WorkerID = string(model.Backend)
	st.StartTime = time.Now()
	s.runningCount++
	s.pendingCount--

	// Execute task asynchronously based on backend
	go s.executeTask(st)
}

func (s *Scheduler) executeTask(st *ScheduledTask) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTaskTimeout)
	defer cancel()

	result := &Result{
		TaskID:      st.Task.ID,
		CompletedAt: time.Now(),
	}

	model := s.modelManager.GetModel(st.Task.Model)

	// Route to appropriate handler based on backend
	switch model.Backend {
	case BackendOllama:
		s.executeOllamaTask(ctx, st, result)
	case BackendHuggingFace:
		s.executeHuggingFaceTask(ctx, st, result)
	case BackendSDWebUI:
		s.executeSDWebUITask(ctx, st, result)
	case BackendLlamaCpp:
		s.executeLlamaCppTask(ctx, st, result)
	case BackendVLLM:
		s.executeVLLMTask(ctx, st, result)
	case BackendLTXVideo:
		s.executeLTXVideoTask(ctx, st, result)
	case BackendExternal:
		s.executeExternalTask(ctx, st, result)
	default:
		result.Error = fmt.Sprintf("unsupported backend: %s", model.Backend)
		s.failTask(st, result, start)
		return
	}

	if result.Error != "" {
		s.failTask(st, result, start)
	} else {
		s.completeTask(st, result, start)
	}
}

func (s *Scheduler) executeOllamaTask(ctx context.Context, st *ScheduledTask, result *Result) {
	result.ContentType = "text/plain"

	// Check if model is available
	if !s.modelManager.IsModelAvailableInOllama(st.Task.Model) {
		result.Error = fmt.Sprintf("model '%s' not found in Ollama. Please run: ollama pull %s", st.Task.Model, st.Task.Model)
		return
	}

	ollama := s.modelManager.GetOllamaClient()

	// Check if Input is JSON (chat messages) or plain text
	var messages []Message
	if st.Task.Input != "" && string(st.Task.Input[0]) == "[" {
		if err := json.Unmarshal([]byte(st.Task.Input), &messages); err != nil {
			messages = []Message{{Role: "user", Content: st.Task.Input}}
		}
	} else {
		messages = []Message{{Role: "user", Content: st.Task.Input}}
	}

	resp, err := ollama.Chat(ctx, st.Task.Model, messages)
	if err != nil {
		result.Error = fmt.Sprintf("Ollama error: %v", err)
		return
	}
	result.Output = resp.Response
}

func (s *Scheduler) executeHuggingFaceTask(ctx context.Context, st *ScheduledTask, result *Result) {
	hf := s.modelManager.GetHuggingFaceClient()
	model := s.modelManager.GetModel(st.Task.Model)

	switch model.TaskType {
	case "text-to-image":
		result.ContentType = "image/png"
		imageData, err := hf.GenerateImage(ctx, model.HFModelID, st.Task.Input, st.Task.Parameters)
		if err != nil {
			result.Error = fmt.Sprintf("image generation failed: %v", err)
			return
		}
		// Save to file
		outputPath, err := SaveOutput(imageData, "image/png", st.Task.ID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to save image: %v", err)
			return
		}
		result.OutputPath = outputPath
		result.OutputData = base64.StdEncoding.EncodeToString(imageData)
		result.Output = fmt.Sprintf("Image saved to: %s", outputPath)

	case "image-to-image":
		result.ContentType = "image/png"
		if st.Task.InputFile == "" {
			result.Error = "image-to-image requires input_image field"
			return
		}
		imageData, err := ParseImageFromBase64(st.Task.InputFile)
		if err != nil {
			result.Error = fmt.Sprintf("failed to parse input image: %v", err)
			return
		}
		outputData, err := hf.ImageToImage(ctx, model.HFModelID, imageData, st.Task.Input, st.Task.Parameters)
		if err != nil {
			result.Error = fmt.Sprintf("image transformation failed: %v", err)
			return
		}
		outputPath, err := SaveOutput(outputData, "image/png", st.Task.ID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to save image: %v", err)
			return
		}
		result.OutputPath = outputPath
		result.OutputData = base64.StdEncoding.EncodeToString(outputData)
		result.Output = fmt.Sprintf("Transformed image saved to: %s", outputPath)

	case "text-to-speech":
		result.ContentType = "audio/mpeg"
		audioData, err := hf.GenerateSpeech(ctx, model.HFModelID, st.Task.Input)
		if err != nil {
			result.Error = fmt.Sprintf("speech generation failed: %v", err)
			return
		}
		outputPath, err := SaveOutput(audioData, "audio/mpeg", st.Task.ID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to save audio: %v", err)
			return
		}
		result.OutputPath = outputPath
		result.OutputData = base64.StdEncoding.EncodeToString(audioData)
		result.Output = fmt.Sprintf("Speech saved to: %s", outputPath)

	case "automatic-speech-recognition":
		result.ContentType = "text/plain"
		if st.Task.InputFile == "" {
			result.Error = "audio transcription requires audio data"
			return
		}
		audioData, err := ParseImageFromBase64(st.Task.InputFile)
		if err != nil {
			result.Error = fmt.Sprintf("failed to parse audio: %v", err)
			return
		}
		text, err := hf.Transcribe(ctx, model.HFModelID, audioData)
		if err != nil {
			result.Error = fmt.Sprintf("transcription failed: %v", err)
			return
		}
		result.Output = text

	case "text-to-video", "image-to-video", "video-to-video":
		result.ContentType = "video/mp4"
		isImage := model.TaskType == "image-to-video" || model.TaskType == "video-to-video"
		videoData, err := hf.GenerateVideo(ctx, model.HFModelID, st.Task.Input, isImage)
		if err != nil {
			result.Error = fmt.Sprintf("video generation failed: %v", err)
			return
		}
		outputPath, err := SaveOutput(videoData, "video/mp4", st.Task.ID)
		if err != nil {
			result.Error = fmt.Sprintf("failed to save video: %v", err)
			return
		}
		result.OutputPath = outputPath
		result.Output = fmt.Sprintf("Video saved to: %s", outputPath)

	default:
		result.Error = fmt.Sprintf("unsupported task type: %s", model.TaskType)
	}
}

func (s *Scheduler) executeSDWebUITask(ctx context.Context, st *ScheduledTask, result *Result) {
	hf := s.modelManager.GetHuggingFaceClient()

	result.ContentType = "image/png"
	imageData, err := hf.GenerateImage(ctx, st.Task.Model, st.Task.Input, st.Task.Parameters)
	if err != nil {
		result.Error = fmt.Sprintf("SD WebUI image generation failed: %v", err)
		return
	}
	outputPath, err := SaveOutput(imageData, "image/png", st.Task.ID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to save image: %v", err)
		return
	}
	result.OutputPath = outputPath
	result.OutputData = base64.StdEncoding.EncodeToString(imageData)
	result.Output = fmt.Sprintf("Image saved to: %s", outputPath)
}

func (s *Scheduler) executeLlamaCppTask(ctx context.Context, st *ScheduledTask, result *Result) {
	result.ContentType = "text/plain"

	llamaCpp := s.modelManager.GetLlamaCppClient()

	var messages []Message
	if st.Task.Input != "" && string(st.Task.Input[0]) == "[" {
		if err := json.Unmarshal([]byte(st.Task.Input), &messages); err != nil {
			messages = []Message{{Role: "user", Content: st.Task.Input}}
		}
	} else {
		messages = []Message{{Role: "user", Content: st.Task.Input}}
	}

	resp, err := llamaCpp.Chat(ctx, st.Task.Model, messages, st.Task.Parameters)
	if err != nil {
		result.Error = fmt.Sprintf("llama.cpp error: %v", err)
		return
	}
	result.Output = resp.Response
}

func (s *Scheduler) executeVLLMTask(ctx context.Context, st *ScheduledTask, result *Result) {
	result.ContentType = "text/plain"

	vllm := s.modelManager.GetVLLMClient()

	var messages []Message
	if st.Task.Input != "" && string(st.Task.Input[0]) == "[" {
		if err := json.Unmarshal([]byte(st.Task.Input), &messages); err != nil {
			messages = []Message{{Role: "user", Content: st.Task.Input}}
		}
	} else {
		messages = []Message{{Role: "user", Content: st.Task.Input}}
	}

	resp, err := vllm.Chat(ctx, st.Task.Model, messages, st.Task.Parameters)
	if err != nil {
		result.Error = fmt.Sprintf("vLLM error: %v", err)
		return
	}
	result.Output = resp.Response
}

func (s *Scheduler) executeLTXVideoTask(ctx context.Context, st *ScheduledTask, result *Result) {
	result.ContentType = "video/mp4"

	ltxVideo := s.modelManager.GetLTXVideoClient()

	videoData, contentType, err := ltxVideo.GenerateVideo(ctx, st.Task.Input, st.Task.InputFile, st.Task.Parameters)
	if err != nil {
		result.Error = fmt.Sprintf("LTX-Video error: %v", err)
		return
	}

	outputPath, err := SaveOutput(videoData, contentType, st.Task.ID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to save video: %v", err)
		return
	}
	result.OutputPath = outputPath
	result.Output = fmt.Sprintf("Video saved to: %s", outputPath)
}

func (s *Scheduler) executeExternalTask(ctx context.Context, st *ScheduledTask, result *Result) {
	result.Error = "external backend not implemented"
}

func (s *Scheduler) completeTask(st *ScheduledTask, result *Result, start time.Time) {
	result.Duration = time.Since(start).Milliseconds()

	s.mu.Lock()
	st.State = TaskCompleted
	st.EndTime = time.Now()
	st.Result = result
	s.completedCount++
	s.runningCount--
	s.mu.Unlock()

	select {
	case s.resultChan <- result:
	default:
	}
}

func (s *Scheduler) failTask(st *ScheduledTask, result *Result, start time.Time) {
	result.Duration = time.Since(start).Milliseconds()

	s.mu.Lock()
	st.State = TaskFailed
	st.EndTime = time.Now()
	st.Result = result
	s.failedCount++
	s.runningCount--
	s.mu.Unlock()

	select {
	case s.resultChan <- result:
	default:
	}
}

// RegisterWorker adds a worker to the scheduler
func (s *Scheduler) RegisterWorker(worker *WorkerInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	worker.Status = "online"
	worker.LastHeartbeat = time.Now()
	s.workers[worker.ID] = worker
}

// UnregisterWorker removes a worker
func (s *Scheduler) UnregisterWorker(workerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workers, workerID)
}

// GetWorkers returns all registered workers
func (s *Scheduler) GetWorkers() []*WorkerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]*WorkerInfo, 0, len(s.workers))
	for _, w := range s.workers {
		workers = append(workers, w)
	}
	return workers
}
