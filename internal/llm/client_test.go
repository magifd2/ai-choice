package llm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/magifd2/ai-choice/internal/llm"
)

// buildClient creates a Client pointed at the given test-server URL.
func buildClient(t *testing.T, serverURL string, maxRetries int) *llm.Client {
	t.Helper()
	return llm.NewClient(serverURL, "sk-test", "test-model", 5*time.Second, maxRetries)
}

// okResponse writes a valid chat completions JSON body with the given tool-call tag.
func okToolResponse(w http.ResponseWriter, tag string) {
	args, _ := json.Marshal(map[string]string{"tag": tag})
	resp := llm.ChatResponse{
		ID:     "chatcmpl-test",
		Object: "chat.completion",
		Choices: []llm.Choice{
			{
				Message: llm.ResponseMessage{
					Role: "assistant",
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "select_choice",
								Arguments: string(args),
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func TestClient_Chat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %q", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk-test" {
			t.Errorf("unexpected Authorization header %q", auth)
		}
		okToolResponse(w, "weather")
	}))
	defer srv.Close()

	client := buildClient(t, srv.URL, 0)
	resp, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Chat() unexpected error: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("Chat() returned no choices")
	}
	calls := resp.Choices[0].Message.ToolCalls
	if len(calls) == 0 {
		t.Fatal("no tool calls in response")
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
		t.Fatalf("parsing tool call arguments: %v", err)
	}
	if args["tag"] != "weather" {
		t.Errorf("tag = %q, want %q", args["tag"], "weather")
	}
}

func TestClient_Chat_Retry_429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintln(w, `{"error":"rate limited"}`)
			return
		}
		okToolResponse(w, "ok")
	}))
	defer srv.Close()

	// Set maxRetries=3 so we survive 2 failures.
	client := buildClient(t, srv.URL, 3)
	resp, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() unexpected error after retries: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("no choices returned")
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 HTTP calls, got %d", calls.Load())
	}
}

func TestClient_Chat_Retry_500(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, `{"error":"server error"}`)
			return
		}
		okToolResponse(w, "done")
	}))
	defer srv.Close()

	client := buildClient(t, srv.URL, 2)
	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() unexpected error: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", calls.Load())
	}
}

func TestClient_Chat_NoRetry_4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"error":"invalid api key"}`)
	}))
	defer srv.Close()

	client := buildClient(t, srv.URL, 3)
	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Chat() expected error for 401, got nil")
	}
	if calls.Load() != 1 {
		t.Errorf("expected exactly 1 HTTP call (no retry on 401), got %d", calls.Load())
	}
}

func TestClient_Chat_ExhaustedRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, `{"error":"down"}`)
	}))
	defer srv.Close()

	client := buildClient(t, srv.URL, 2)
	_, err := client.Chat(context.Background(), llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Chat() expected error after exhausted retries, got nil")
	}
}

func TestClient_Chat_ContextCancelled(t *testing.T) {
	// Server that is slow enough to allow cancellation.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		okToolResponse(w, "nope")
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	client := buildClient(t, srv.URL, 0)
	_, err := client.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Chat() expected error from cancelled context, got nil")
	}
}
