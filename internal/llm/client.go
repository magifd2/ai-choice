package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Request / Response types
// ---------------------------------------------------------------------------

// Message represents a single turn in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ToolFunction describes the schema of a callable function.
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Tool wraps a function definition for the tools array in a chat request.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ChatRequest is the JSON body sent to the /chat/completions endpoint.
type ChatRequest struct {
	Model      string    `json:"model"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice any       `json:"tool_choice,omitempty"`
}

// FunctionCall holds the name and JSON-encoded arguments of a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall represents a single function invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// ResponseMessage is the assistant's reply, which may include tool calls.
type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Choice is one candidate completion returned by the API.
type Choice struct {
	Index        int             `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// ChatResponse is the top-level object returned by the API.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Choices []Choice `json:"choices"`
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client sends requests to an OpenAI-compatible chat completions endpoint.
type Client struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
	maxRetries int
}

// NewClient creates a Client.
// endpoint should be the base URL (e.g. "https://api.openai.com/v1").
func NewClient(endpoint, apiKey, model string, timeout time.Duration, maxRetries int) *Client {
	return &Client{
		endpoint:   strings.TrimRight(endpoint, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
}

// retryableError is returned when the caller should back-off and retry.
type retryableError struct {
	statusCode int
	err        error
}

func (e *retryableError) Error() string {
	if e.statusCode != 0 {
		return fmt.Sprintf("HTTP %d: %v", e.statusCode, e.err)
	}
	return e.err.Error()
}

func (e *retryableError) Unwrap() error { return e.err }

// Chat sends a ChatRequest and returns the parsed ChatResponse.
// It retries on network errors, HTTP 429, and HTTP 5xx using exponential backoff.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Model = c.model

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, body)
		if err != nil {
			var re *retryableError
			if errors.As(err, &re) {
				lastErr = err
				continue
			}
			return nil, err // non-retryable
		}
		return resp, nil
	}
	return nil, fmt.Errorf("all %d attempts failed, last error: %w", c.maxRetries+1, lastErr)
}

// doRequest performs a single HTTP round-trip and returns a parsed response or error.
func (c *Client) doRequest(ctx context.Context, body []byte) (*ChatResponse, error) {
	url := c.endpoint + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Network-level errors are always retryable.
		return nil, &retryableError{err: fmt.Errorf("sending request: %w", err)}
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("reading response body: %w", err)}
	}

	switch {
	case httpResp.StatusCode == http.StatusOK:
		// continue to parse

	case httpResp.StatusCode == http.StatusTooManyRequests:
		return nil, &retryableError{
			statusCode: httpResp.StatusCode,
			err:        fmt.Errorf("rate limited: %s", truncate(string(respBody), 200)),
		}

	case httpResp.StatusCode >= 500:
		return nil, &retryableError{
			statusCode: httpResp.StatusCode,
			err:        fmt.Errorf("server error: %s", truncate(string(respBody), 200)),
		}

	default:
		// 4xx (other than 429) — not retryable
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, truncate(string(respBody), 200))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing API response: %w (body: %s)", err, truncate(string(respBody), 200))
	}
	return &chatResp, nil
}

// truncate shortens s to at most n runes.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
