package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LTXVideoClient struct {
	BaseURL string
	Client  *http.Client
}

type LTXVideoRequest struct {
	Prompt    string  `json:"prompt"`
	Image     string  `json:"image,omitempty"`
	Width     int     `json:"width,omitempty"`
	Height    int     `json:"height,omitempty"`
	NumFrames int     `json:"num_frames,omitempty"`
	Steps     int     `json:"steps,omitempty"`
	CFGScale  float64 `json:"cfg_scale,omitempty"`
	Seed      int64   `json:"seed,omitempty"`
}

type LTXVideoResponse struct {
	Images []string `json:"images,omitempty"`
	Videos []string `json:"videos,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func NewLTXVideoClient(baseURL string) *LTXVideoClient {
	if baseURL == "" {
		baseURL = "http://localhost:8188"
	}
	return &LTXVideoClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

func (c *LTXVideoClient) IsAvailable() bool {
	resp, err := c.Client.Get(c.BaseURL + "/system_stats")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *LTXVideoClient) GenerateVideo(ctx context.Context, prompt string, inputImage string, params map[string]any) ([]byte, string, error) {
	width := 768
	height := 512
	numFrames := 33
	steps := 50
	cfgScale := 7.0

	if w, ok := params["width"].(float64); ok {
		width = int(w)
	}
	if h, ok := params["height"].(float64); ok {
		height = int(h)
	}
	if nf, ok := params["num_frames"].(float64); ok {
		numFrames = int(nf)
	}
	if s, ok := params["steps"].(float64); ok {
		steps = int(s)
	}
	if cgf, ok := params["cfg_scale"].(float64); ok {
		cfgScale = cgf
	}

	req := map[string]any{
		"prompt":     prompt,
		"width":      width,
		"height":     height,
		"num_frames": numFrames,
		"steps":      steps,
		"cfg_scale":  cfgScale,
	}

	if inputImage != "" {
		req["image"] = inputImage
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/video/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("LTX-Video error: status %d, %s", resp.StatusCode, string(body))
	}

	var result LTXVideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	if result.Error != "" {
		return nil, "", fmt.Errorf("LTX-Video error: %s", result.Error)
	}

	var outputData string
	if len(result.Videos) > 0 {
		outputData = result.Videos[0]
	} else if len(result.Images) > 0 {
		outputData = result.Images[0]
	}

	return []byte(outputData), "video/mp4", nil
}
