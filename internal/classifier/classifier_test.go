package classifier_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/magifd2/ai-choice/internal/classifier"
	"github.com/magifd2/ai-choice/internal/config"
	"github.com/magifd2/ai-choice/internal/llm"
)

// ---------------------------------------------------------------------------
// Mock LLM client
// ---------------------------------------------------------------------------

type mockClient struct {
	resp *llm.ChatResponse
	err  error
}

func (m *mockClient) Chat(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	return m.resp, m.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var testChoices = []config.Choice{
	{Tag: "weather", Description: "Weather forecast"},
	{Tag: "time", Description: "Current time"},
	{Tag: "fortune", Description: "Fortune telling"},
	{Tag: "default", Description: "Everything else"},
}

var testConfig = &config.Config{
	Endpoint:       "https://api.example.com/v1",
	APIKey:         "sk-test",
	Model:          "test-model",
	TimeoutSeconds: 30,
	MaxRetries:     3,
	Choices:        testChoices,
}

func toolCallResponse(tag string) *llm.ChatResponse {
	args, _ := json.Marshal(map[string]string{"tag": tag})
	return &llm.ChatResponse{
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
}

func contentResponse(content string) *llm.ChatResponse {
	return &llm.ChatResponse{
		Choices: []llm.Choice{
			{
				Message: llm.ResponseMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: "stop",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests: primary path (tool call)
// ---------------------------------------------------------------------------

func TestClassify_ToolCall(t *testing.T) {
	for _, tag := range []string{"weather", "time", "fortune", "default"} {
		t.Run(tag, func(t *testing.T) {
			client := &mockClient{resp: toolCallResponse(tag)}
			got, err := classifier.Classify(context.Background(), "some input", testConfig, client)
			if err != nil {
				t.Fatalf("Classify() unexpected error: %v", err)
			}
			if got != tag {
				t.Errorf("Classify() = %q, want %q", got, tag)
			}
		})
	}
}

func TestClassify_ToolCall_InvalidTag_FallsBack(t *testing.T) {
	// Model returns an unknown tag via tool call → should fall back to default.
	args, _ := json.Marshal(map[string]string{"tag": "unknown_tag"})
	resp := &llm.ChatResponse{
		Choices: []llm.Choice{
			{
				Message: llm.ResponseMessage{
					ToolCalls: []llm.ToolCall{
						{Function: llm.FunctionCall{Name: "select_choice", Arguments: string(args)}},
					},
				},
			},
		},
	}
	client := &mockClient{resp: resp}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	// Should fall back to last choice ("default").
	if got != "default" {
		t.Errorf("Classify() = %q, want %q", got, "default")
	}
}

// ---------------------------------------------------------------------------
// Tests: JSON content fallback
// ---------------------------------------------------------------------------

func TestClassify_JSONFallback(t *testing.T) {
	client := &mockClient{resp: contentResponse(`{"tag": "weather"}`)}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	if got != "weather" {
		t.Errorf("Classify() = %q, want %q", got, "weather")
	}
}

func TestClassify_JSONFallback_NoisyText(t *testing.T) {
	// Model returns JSON embedded in prose.
	noisy := `Sure, the answer is {"tag": "time"} as requested.`
	client := &mockClient{resp: contentResponse(noisy)}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	if got != "time" {
		t.Errorf("Classify() = %q, want %q", got, "time")
	}
}

// ---------------------------------------------------------------------------
// Tests: text scan fallback
// ---------------------------------------------------------------------------

func TestClassify_TextFallback(t *testing.T) {
	// No JSON, but the tag appears verbatim in the content.
	client := &mockClient{resp: contentResponse("I would choose fortune for this one.")}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	if got != "fortune" {
		t.Errorf("Classify() = %q, want %q", got, "fortune")
	}
}

// ---------------------------------------------------------------------------
// Tests: default fallback
// ---------------------------------------------------------------------------

func TestClassify_Default_NoChoicesMatched(t *testing.T) {
	client := &mockClient{resp: contentResponse("absolutely no match here xyz")}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	// Last choice is "default".
	if got != "default" {
		t.Errorf("Classify() = %q, want %q", got, "default")
	}
}

func TestClassify_Default_EmptyResponse(t *testing.T) {
	client := &mockClient{resp: &llm.ChatResponse{}}
	got, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err != nil {
		t.Fatalf("Classify() unexpected error: %v", err)
	}
	if got != "default" {
		t.Errorf("Classify() = %q, want %q", got, "default")
	}
}

// ---------------------------------------------------------------------------
// Tests: LLM error
// ---------------------------------------------------------------------------

func TestClassify_LLMError(t *testing.T) {
	client := &mockClient{err: errTest}
	_, err := classifier.Classify(context.Background(), "input", testConfig, client)
	if err == nil {
		t.Fatal("Classify() expected error from LLM, got nil")
	}
}

var errTest = &testError{"simulated LLM failure"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
