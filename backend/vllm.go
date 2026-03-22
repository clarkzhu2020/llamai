package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type VLLMClient struct {
	BaseURL string
	Client  *http.Client
}

type VLLMResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int     `json:"index"`
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func NewVLLMClient(baseURL string) *VLLMClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &VLLMClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *VLLMClient) IsAvailable() bool {
	resp, err := c.Client.Get(c.BaseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *VLLMClient) ListModels() []string {
	resp, err := c.Client.Get(c.BaseURL + "/v1/models")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models
}

func (c *VLLMClient) Generate(ctx context.Context, prompt string, model string, params map[string]any) (string, error) {
	reqBody := map[string]any{
		"prompt": prompt,
		"model":  model,
	}

	if temp, ok := params["temperature"].(float64); ok {
		reqBody["temperature"] = temp
	} else {
		reqBody["temperature"] = 0.7
	}

	if maxTokens, ok := params["max_tokens"].(float64); ok {
		reqBody["max_tokens"] = int(maxTokens)
	} else {
		reqBody["max_tokens"] = 2048
	}

	if topP, ok := params["top_p"].(float64); ok {
		reqBody["top_p"] = topP
	} else {
		reqBody["top_p"] = 0.9
	}

	if stop, ok := params["stop"].([]string); ok {
		reqBody["stop"] = stop
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+("/v1/completions"), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errMsg, ok := errResp["error"].(string); ok {
			return "", fmt.Errorf("vLLM error: %s", errMsg)
		}
		return "", fmt.Errorf("vLLM error: status %d", resp.StatusCode)
	}

	var result VLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("vLLM error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from vLLM")
	}

	return result.Choices[0].Message.Content, nil
}

func (c *VLLMClient) Chat(ctx context.Context, model string, messages []Message, params map[string]any) (*OllamaResponse, error) {
	chatMessages := make([]Message, len(messages))
	for i, m := range messages {
		chatMessages[i] = Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	reqBody := map[string]any{
		"model":    model,
		"messages": chatMessages,
	}

	if temp, ok := params["temperature"].(float64); ok {
		reqBody["temperature"] = temp
	} else {
		reqBody["temperature"] = 0.7
	}

	if maxTokens, ok := params["max_tokens"].(float64); ok {
		reqBody["max_tokens"] = int(maxTokens)
	} else {
		reqBody["max_tokens"] = 2048
	}

	if topP, ok := params["top_p"].(float64); ok {
		reqBody["top_p"] = topP
	} else {
		reqBody["top_p"] = 0.9
	}

	if stop, ok := params["stop"].([]string); ok {
		reqBody["stop"] = stop
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		var errResp map[string]any
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errMsg, ok := errResp["error"].(string); ok {
			return nil, fmt.Errorf("vLLM chat error: %s", errMsg)
		}
		return nil, fmt.Errorf("vLLM chat error: status %d", resp.StatusCode)
	}

	var result VLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("vLLM error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from vLLM")
	}

	return &OllamaResponse{
		Model:    model,
		Response: result.Choices[0].Message.Content,
		Done:     true,
	}, nil
}
