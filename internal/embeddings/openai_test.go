package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAIClientPayloadBatchingAndRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := requests.Add(1)
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		var request embeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Model != "model" || request.Dimensions != 2 {
			t.Errorf("request = %#v", request)
		}
		if call == 1 {
			http.Error(w, "retry", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[`)
		for i := range request.Input {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"index":%d,"embedding":[1,0]}`, i)
		}
		fmt.Fprintf(w, `],"usage":{"total_tokens":%d}}`, len(request.Input)*2)
	}))
	defer server.Close()

	inputs := make([]string, 65)
	for i := range inputs {
		inputs[i] = fmt.Sprintf("input %d", i)
	}
	client := &OpenAIClient{APIKey: "secret", BaseURL: server.URL, HTTP: server.Client(), MaxRetries: 2, Backoff: time.Millisecond}
	vectors, tokens, err := client.Embed(context.Background(), "model", 2, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 65 || tokens != 130 {
		t.Fatalf("got %d vectors and %d tokens", len(vectors), tokens)
	}
	if requests.Load() != 3 { // retry + successful batch of 64 + final batch of 1
		t.Fatalf("requests = %d, want 3", requests.Load())
	}
}

func TestOpenAIClientHonorsContextDuringBackoff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "retry", http.StatusInternalServerError)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	client := &OpenAIClient{APIKey: "secret", BaseURL: server.URL, HTTP: server.Client(), Backoff: time.Second}
	_, _, err := client.Embed(ctx, "model", 2, []string{"input"})
	if err == nil {
		t.Fatal("expected context error")
	}
}
