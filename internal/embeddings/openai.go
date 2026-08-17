package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const MaxBatchInputs = 64

type OpenAIClient struct {
	APIKey     string
	BaseURL    string
	HTTP       *http.Client
	MaxRetries int
	Backoff    time.Duration
}

type embeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func (c *OpenAIClient) Embed(ctx context.Context, model string, dims int, inputs []string) ([][]float32, int, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, 0, fmt.Errorf("empty OpenAI API key")
	}
	if len(inputs) == 0 {
		return nil, 0, nil
	}
	all := make([][]float32, 0, len(inputs))
	totalTokens := 0
	for start := 0; start < len(inputs); start += MaxBatchInputs {
		end := start + MaxBatchInputs
		if end > len(inputs) {
			end = len(inputs)
		}
		vectors, tokens, err := c.embedBatch(ctx, model, dims, inputs[start:end])
		if err != nil {
			return nil, totalTokens, err
		}
		all = append(all, vectors...)
		totalTokens += tokens
	}
	return all, totalTokens, nil
}

func (c *OpenAIClient) embedBatch(ctx context.Context, model string, dims int, inputs []string) ([][]float32, int, error) {
	payload, err := json.Marshal(embeddingRequest{Model: model, Input: inputs, Dimensions: dims})
	if err != nil {
		return nil, 0, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	maxRetries := c.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 4
	}
	backoff := c.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/embeddings", bytes.NewReader(payload))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, 0, readErr
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var decoded embeddingResponse
			if err := json.Unmarshal(body, &decoded); err != nil {
				return nil, 0, fmt.Errorf("decode OpenAI embeddings response: %w", err)
			}
			if len(decoded.Data) != len(inputs) {
				return nil, 0, fmt.Errorf("OpenAI returned %d embeddings for %d inputs", len(decoded.Data), len(inputs))
			}
			vectors := make([][]float32, len(inputs))
			for _, item := range decoded.Data {
				if item.Index < 0 || item.Index >= len(vectors) {
					return nil, 0, fmt.Errorf("OpenAI returned invalid embedding index %d", item.Index)
				}
				if len(item.Embedding) != dims {
					return nil, 0, fmt.Errorf("OpenAI returned %d dimensions, want %d", len(item.Embedding), dims)
				}
				vectors[item.Index] = item.Embedding
			}
			return vectors, decoded.Usage.TotalTokens, nil
		}

		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if !retryable || attempt >= maxRetries {
			return nil, 0, fmt.Errorf("OpenAI embeddings %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		delay := backoff << attempt
		if retryAfter, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && retryAfter > 0 {
			delay = time.Duration(retryAfter) * time.Second
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, 0, ctx.Err()
		case <-timer.C:
		}
	}
}
