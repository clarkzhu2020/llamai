package main

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GPUManager struct {
	gpus map[int]*GPU
	mu   sync.RWMutex
}

type GPU struct {
	ID              int
	Name            string
	TotalMB         int64
	UsedMB          int64
	TaskAllocations map[string]int64
}

func NewGPUManager() *GPUManager {
	return &GPUManager{
		gpus: make(map[int]*GPU),
	}
}

func (m *GPUManager) DetectGPUs() int {
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.used", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(output), "\n")
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		totalMB, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)

		m.AddGPU(index, name, totalMB)
		count++
	}

	return count
}

func (m *GPUManager) AddGPU(id int, name string, totalMB int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gpus[id] = &GPU{
		ID:              id,
		Name:            name,
		TotalMB:         totalMB,
		TaskAllocations: make(map[string]int64),
	}
}

func (m *GPUManager) AddSimulatedGPU(id int, name string, totalMB int64) {
	m.AddGPU(id, name, totalMB)
}

func (m *GPUManager) Allocate(taskID string, requiredMB int64) (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, gpu := range m.gpus {
		free := gpu.TotalMB - gpu.UsedMB
		if free >= requiredMB {
			gpu.UsedMB += requiredMB
			gpu.TaskAllocations[taskID] = requiredMB
			return id, true
		}
	}
	return -1, false
}

func (m *GPUManager) Release(taskID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, gpu := range m.gpus {
		if allocated, ok := gpu.TaskAllocations[taskID]; ok {
			gpu.UsedMB -= allocated
			delete(gpu.TaskAllocations, taskID)
		}
	}
}

func (m *GPUManager) GetInfo() []GPUInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd := exec.Command("nvidia-smi", "--query-gpu=index,memory.used,utilization.gpu", "--format=csv,noheader,nounits")
	output, _ := cmd.Output()

	gpuUsed := make(map[int]int64)
	gpuUtil := make(map[int]int)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 3 {
			if idx, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
				if used, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
					gpuUsed[idx] = used
				}
				if util, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
					gpuUtil[idx] = util
				}
			}
		}
	}

	infos := make([]GPUInfo, 0, len(m.gpus))
	for id, gpu := range m.gpus {
		usedMB := gpu.UsedMB
		if realUsed, ok := gpuUsed[id]; ok {
			usedMB = realUsed
		}
		utilRate := 0
		if realUtil, ok := gpuUtil[id]; ok {
			utilRate = realUtil
		}

		infos = append(infos, GPUInfo{
			ID:       id,
			Name:     gpu.Name,
			TotalMB:  gpu.TotalMB,
			UsedMB:   usedMB,
			FreeMB:   gpu.TotalMB - usedMB,
			UtilRate: utilRate,
			MemRate:  int(float64(usedMB) / float64(gpu.TotalMB) * 100),
		})
	}
	return infos
}

func (m *GPUManager) GetBestGPU(requiredMB int64) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bestID := -1
	bestFree := int64(0)

	for id, gpu := range m.gpus {
		free := gpu.TotalMB - gpu.UsedMB
		if free >= requiredMB && free > bestFree {
			bestFree = free
			bestID = id
		}
	}

	return bestID, bestID >= 0
}

func (m *GPUManager) GetTotalFree() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, gpu := range m.gpus {
		total += gpu.TotalMB - gpu.UsedMB
	}
	return total
}

func (m *GPUManager) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cmd := exec.Command("nvidia-smi", "--query-gpu=index", "--format=csv,noheader")
			cmd.Output()
		}
	}()
}
