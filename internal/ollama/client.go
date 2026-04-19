package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Host       string
	Model      string
	Key        string
	HTTPClient *http.Client
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

type StreamToken struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

func NewClient(host, model, key string) *Client {
	return &Client{
		Host:       host,
		Model:      model,
		Key:        key,
		HTTPClient: &http.Client{},
	}
}

// Generate sends a prompt and returns the complete model response.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.Host + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Response, nil
}

// GenerateStream sends a prompt and calls onToken for each streamed token.
func (c *Client) GenerateStream(ctx context.Context, prompt string, onToken func(string)) error {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.Host + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var token StreamToken
		if err := json.Unmarshal(scanner.Bytes(), &token); err != nil {
			continue
		}
		if token.Response != "" {
			onToken(token.Response)
		}
		if token.Done {
			break
		}
	}

	return scanner.Err()
}