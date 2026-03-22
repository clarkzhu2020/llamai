package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type LlamaCppClient struct {
	BaseURL string
	Client  *http.Client
}

type LlamaCppResponse struct {
	Content    string `json:"content,omitempty"`
	Stop       bool   `json:"stop,omitempty"`
	Model      string `json:"model,omitempty"`
	TokensCur  int    `json:"tokens_cur,omitempty"`
	TokensPred int    `json:"tokens_pred,omitempty"`
	Error      string `json:"error,omitempty"`
}

type LlamaCppChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LlamaCppCompletionRequest struct {
	Prompt      string   `json:"prompt"`
	Model       string   `json:"model,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	N           Predict  `json:"n,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	TopP        float64  `json:"top_p,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type Predict struct {
	Tokens int `json:"tokens,omitempty"`
}

type LlamaCppChatRequest struct {
	Messages    []LlamaCppChatMessage `json:"messages"`
	Model       string                `json:"model,omitempty"`
	Stream      bool                  `json:"stream,omitempty"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	TopP        float64               `json:"top_p,omitempty"`
	Stop        []string              `json:"stop,omitempty"`
}

func NewLlamaCppClient(baseURL string) *LlamaCppClient {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &LlamaCppClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *LlamaCppClient) IsAvailable() bool {
	resp, err := c.Client.Get(c.BaseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (c *LlamaCppClient) Generate(ctx context.Context, prompt string, model string, params map[string]any) (string, error) {
	req := LlamaCppCompletionRequest{
		Prompt: prompt,
		Model:  model,
	}

	if temp, ok := params["temperature"].(float64); ok {
		req.Temperature = temp
	} else {
		req.Temperature = 0.7
	}

	if maxTokens, ok := params["max_tokens"].(float64); ok {
		req.MaxTokens = int(maxTokens)
	} else {
		req.MaxTokens = 2048
	}

	if topP, ok := params["top_p"].(float64); ok {
		req.TopP = topP
	} else {
		req.TopP = 0.9
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/completion", bytes.NewBuffer(jsonData))
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
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errMsg, ok := errResp["error"]; ok {
			return "", fmt.Errorf("llama.cpp error: %s", errMsg)
		}
		return "", fmt.Errorf("llama.cpp error: status %d", resp.StatusCode)
	}

	var result LlamaCppResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("llama.cpp error: %s", result.Error)
	}

	return result.Content, nil
}

func (c *LlamaCppClient) Chat(ctx context.Context, model string, messages []Message, params map[string]any) (*OllamaResponse, error) {
	llamaMsgs := make([]LlamaCppChatMessage, len(messages))
	for i, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "assistant"
		} else if role == "user" {
			role = "user"
		} else if role == "system" {
			role = "system"
		}
		llamaMsgs[i] = LlamaCppChatMessage{
			Role:    role,
			Content: m.Content,
		}
	}

	req := LlamaCppChatRequest{
		Messages: llamaMsgs,
		Model:    model,
	}

	if temp, ok := params["temperature"].(float64); ok {
		req.Temperature = temp
	} else {
		req.Temperature = 0.7
	}

	if maxTokens, ok := params["max_tokens"].(float64); ok {
		req.MaxTokens = int(maxTokens)
	} else {
		req.MaxTokens = 2048
	}

	if topP, ok := params["top_p"].(float64); ok {
		req.TopP = topP
	} else {
		req.TopP = 0.9
	}

	jsonData, err := json.Marshal(req)
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
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errMsg, ok := errResp["error"]; ok {
			return nil, fmt.Errorf("llama.cpp chat error: %s", errMsg)
		}
		return nil, fmt.Errorf("llama.cpp chat error: status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from llama.cpp")
	}

	return &OllamaResponse{
		Model:    model,
		Response: result.Choices[0].Message.Content,
		Done:     true,
	}, nil
}
