package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HuggingFaceClient for inference API
type HuggingFaceClient struct {
	APIKey   string
	BaseURL  string
	Client   *http.Client
	LocalMode bool
}

// HuggingFaceResponse for API responses
type HuggingFaceResponse struct {
	GeneratedText string `json:"generated_text,omitempty"`
	Audio         string `json:"audio,omitempty"`
	Image         string `json:"image,omitempty"`
	Video         string `json:"video,omitempty"`
	Error         string `json:"error,omitempty"`
}

// InferenceRequest for various generation tasks
type InferenceRequest struct {
	Model       string            `json:"model,omitempty"`
	Inputs      string            `json:"inputs,omitempty"`
	Prompt      string            `json:"prompt,omitempty"`
	Parameters  map[string]any    `json:"parameters,omitempty"`
}

// NewHuggingFaceClient creates a new HuggingFace client
func NewHuggingFaceClient(apiKey string) *HuggingFaceClient {
	if apiKey == "" {
		apiKey = os.Getenv("HF_TOKEN")
	}
	return &HuggingFaceClient{
		APIKey:   apiKey,
		BaseURL:  "https://api-inference.huggingface.co/models",
		Client: &http.Client{
			Timeout: 10 * time.Minute,
		},
		LocalMode: apiKey == "",
	}
}

// IsAvailable checks if HuggingFace API is accessible
func (c *HuggingFaceClient) IsAvailable() bool {
	if c.LocalMode {
		// Check if we're running in local mode with transformers
		return false
	}
	resp, err := c.Client.Get("https://huggingface.co")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

// GenerateImage creates an image from text prompt
func (c *HuggingFaceClient) GenerateImage(ctx context.Context, model, prompt string, params map[string]any) ([]byte, error) {
	if c.LocalMode {
		return c.generateImageLocal(ctx, model, prompt, params)
	}
	return c.generateImageAPI(ctx, model, prompt, params)
}

func (c *HuggingFaceClient) generateImageAPI(ctx context.Context, model, prompt string, params map[string]any) ([]byte, error) {
	// SDXL-Lightning requires specific handling
	modelID, steps := c.getSDXLLightningConfig(model)

	reqBody := map[string]any{
		"inputs": prompt,
	}

	// Build parameters for this specific model
	genParams := c.buildImageParams(modelID, params, steps)
	if genParams != nil {
		reqBody["parameters"] = genParams
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/"+modelID, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HF API error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HF returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// getSDXLLightningConfig returns the actual model ID and recommended steps for SDXL-Lightning
func (c *HuggingFaceClient) getSDXLLightningConfig(model string) (string, int) {
	// SDXL-Lightning variants - use the 4-step version as default
	switch model {
	case "ByteDance/SDXL-Lightning-4step":
		return "ByteDance/SDXL-Lightning", 4
	case "ByteDance/SDXL-Lightning-2step":
		return "ByteDance/SDXL-Lightning", 2
	case "ByteDance/SDXL-Lightning-8step":
		return "ByteDance/SDXL-Lightning", 8
	case "ByteDance/SDXL-Lightning-1step":
		return "ByteDance/SDXL-Lightning", 1
	default:
		return model, 30 // Default steps for other models
	}
}

// buildImageParams builds parameters for image generation based on model
func (c *HuggingFaceClient) buildImageParams(modelID string, params map[string]any, steps int) map[string]any {
	// SDXL-Lightning specific parameters
	if strings.HasPrefix(modelID, "ByteDance/SDXL-Lightning") {
		genParams := map[string]any{
			"num_inference_steps": steps,
			"guidance_scale":      0, // Must be 0 for SDXL-Lightning
		}

		// Add optional parameters from user
		if params != nil {
			if neg, ok := params["negative_prompt"]; ok {
				genParams["negative_prompt"] = neg
			}
			if width, ok := params["width"]; ok {
				genParams["width"] = width
			}
			if height, ok := params["height"]; ok {
				genParams["height"] = height
			}
			if guidance, ok := params["guidance_scale"]; ok {
				// Allow override but warn if not 0
				genParams["guidance_scale"] = guidance
			}
		}

		return genParams
	}

	// Default parameters for other models
	if params == nil {
		return nil
	}

	genParams := map[string]any{}
	if v, ok := params["num_inference_steps"]; ok {
		genParams["num_inference_steps"] = v
	}
	if v, ok := params["guidance_scale"]; ok {
		genParams["guidance_scale"] = v
	}
	if v, ok := params["negative_prompt"]; ok {
		genParams["negative_prompt"] = v
	}

	if len(genParams) == 0 {
		return nil
	}
	return genParams
}

// generateImageLocal uses local inference (placeholder - requires transformers setup)
func (c *HuggingFaceClient) generateImageLocal(ctx context.Context, model, prompt string, params map[string]any) ([]byte, error) {
	// In a full implementation, this would use Go bindings for Python transformers/diffusers
	// or spawn a Python process for inference
	// For now, return an error suggesting cloud mode
	
	// Check if we can use a local SD WebUI
	sdURL := os.Getenv("SD_WEBUI_URL")
	if sdURL != "" {
		return c.generateImageWebUI(ctx, sdURL, model, prompt, params)
	}

	return nil, fmt.Errorf("image generation requires either HF_API_TOKEN or SD_WEBUI_URL environment")
}

// generateImageWebUI uses Stable Diffusion WebUI API
func (c *HuggingFaceClient) generateImageWebUI(ctx context.Context, webUIBase, model, prompt string, params map[string]any) ([]byte, error) {
	// Stable Diffusion WebUI API endpoint
	url := webUIBase + "/sdapi/v1/txt2img"
	
	reqBody := map[string]any{
		"prompt":          prompt,
		"negative_prompt": params["negative_prompt"],
		"steps":           params["steps"],
		"width":           params["width"],
		"height":          params["height"],
		"cfg_scale":       params["cfg_scale"],
	}
	if reqBody["steps"] == nil {
		reqBody["steps"] = 30
	}
	if reqBody["width"] == nil {
		reqBody["width"] = 512
	}
	if reqBody["height"] == nil {
		reqBody["height"] = 512
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SD WebUI error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SD WebUI returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Images []string `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	return base64.StdEncoding.DecodeString(result.Images[0])
}

// ImageToImage transforms an image based on prompt
func (c *HuggingFaceClient) ImageToImage(ctx context.Context, model string, imageData []byte, prompt string, params map[string]any) ([]byte, error) {
	webUIBase := os.Getenv("SD_WEBUI_URL")
	if webUIBase == "" {
		return nil, fmt.Errorf("image-to-image requires SD_WEBUI_URL environment variable")
	}

	url := webUIBase + "/sdapi/v1/img2img"
	
	reqBody := map[string]any{
		"init_images":     []string{base64.StdEncoding.EncodeToString(imageData)},
		"prompt":          prompt,
		"negative_prompt": params["negative_prompt"],
		"steps":           params["steps"],
		"denoising_strength": params["denoising_strength"],
	}
	if reqBody["steps"] == nil {
		reqBody["steps"] = 30
	}
	if reqBody["denoising_strength"] == nil {
		reqBody["denoising_strength"] = 0.75
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Images []string `json:"images"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	return base64.StdEncoding.DecodeString(result.Images[0])
}

// GenerateSpeech converts text to speech
func (c *HuggingFaceClient) GenerateSpeech(ctx context.Context, model, text string) ([]byte, error) {
	if c.LocalMode {
		return c.generateSpeechLocal(ctx, model, text)
	}
	return c.generateSpeechAPI(ctx, model, text)
}

func (c *HuggingFaceClient) generateSpeechAPI(ctx context.Context, model, text string) ([]byte, error) {
	reqBody := map[string]any{
		"inputs": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/"+model, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HF returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// Transcribe converts speech to text
func (c *HuggingFaceClient) Transcribe(ctx context.Context, model string, audioData []byte) (string, error) {
	if c.LocalMode {
		return c.transcribeLocal(ctx, model, audioData)
	}
	return c.transcribeAPI(ctx, model, audioData)
}

func (c *HuggingFaceClient) transcribeAPI(ctx context.Context, model string, audioData []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/"+model, bytes.NewReader(audioData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HF API error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HF returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Whisper returns JSON with "text" field
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	return result.Text, nil
}

func (c *HuggingFaceClient) transcribeLocal(ctx context.Context, model string, audioData []byte) (string, error) {
	// Check for Whisper API server
	whisperURL := os.Getenv("WHISPER_URL")
	if whisperURL != "" {
		return c.transcribeExternal(ctx, whisperURL, audioData)
	}

	return "", fmt.Errorf("speech transcription requires HF_TOKEN or WHISPER_URL environment")
}

func (c *HuggingFaceClient) transcribeExternal(ctx context.Context, whisperURL string, audioData []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", whisperURL+"/transcribe", bytes.NewReader(audioData))
	if err != nil {
		return "", err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Text, nil
}

func (c *HuggingFaceClient) generateSpeechLocal(ctx context.Context, model, text string) ([]byte, error) {
	// Check for Coqui TTS or other local TTS
	ttsURL := os.Getenv("TTS_URL")
	if ttsURL != "" {
		return c.generateSpeechExternal(ctx, ttsURL, text)
	}

	return nil, fmt.Errorf("speech generation requires HF_TOKEN or TTS_URL environment")
}

func (c *HuggingFaceClient) generateSpeechExternal(ctx context.Context, ttsURL, text string) ([]byte, error) {
	reqBody := map[string]any{
		"text": text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", ttsURL+"/tts", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// GenerateVideo creates video from text or image
func (c *HuggingFaceClient) GenerateVideo(ctx context.Context, model, input string, isImage bool) ([]byte, error) {
	// Video generation typically requires cloud API or heavy local setup
	// Check for external video generation service first
	
	videoURL := os.Getenv("VIDEO_GEN_URL")
	if videoURL != "" {
		return c.generateVideoExternal(ctx, videoURL, model, input, isImage)
	}

	// Use HuggingFace Inference API if token available
	if !c.LocalMode {
		return c.generateVideoAPI(ctx, model, input, isImage)
	}

	return nil, fmt.Errorf("video generation requires VIDEO_GEN_URL or HF_TOKEN environment")
}

func (c *HuggingFaceClient) generateVideoAPI(ctx context.Context, model, input string, isImage bool) ([]byte, error) {
	reqBody := map[string]any{
		"inputs": input,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/"+model, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HF returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *HuggingFaceClient) generateVideoExternal(ctx context.Context, videoURL, model, input string, isImage bool) ([]byte, error) {
	reqBody := map[string]any{
		"model":  model,
		"input":  input,
		"is_image_input": isImage,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", videoURL+"/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// SaveOutput saves generated content to file
func SaveOutput(data []byte, outputType, taskID string) (string, error) {
	dir := "./outputs"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	var ext string
	switch outputType {
	case "image/png":
		ext = ".png"
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "audio/mpeg", "audio/mp3":
		ext = ".mp3"
	case "audio/wav":
		ext = ".wav"
	case "video/mp4":
		ext = ".mp4"
	default:
		ext = ".bin"
	}

	filename := filepath.Join(dir, taskID+ext)
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", err
	}

	return filename, nil
}

// ParseImageFromBase64 parses base64 encoded image
func ParseImageFromBase64(data string) ([]byte, error) {
	// Handle data URL format
	if idx := strings.Index(data, ","); idx != -1 {
		data = data[idx+1:]
	}
	return base64.StdEncoding.DecodeString(data)
}
