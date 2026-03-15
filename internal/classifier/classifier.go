// Package classifier provides the core classification logic that maps free-form
// user text to a predefined tag using an LLM.
package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/magifd2/ai-choice/internal/config"
	"github.com/magifd2/ai-choice/internal/llm"
)

// LLMClient is the interface that the classifier uses to communicate with the LLM.
// Using an interface makes it easy to substitute a mock in tests.
type LLMClient interface {
	Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
}

// Classify sends the user input to the LLM and returns the best-matching tag
// from cfg.Choices.  It uses tool calling as the primary mechanism and falls
// back to JSON / plain-text extraction when the model does not return a tool
// call.
func Classify(ctx context.Context, input string, cfg *config.Config, client LLMClient) (string, error) {
	// Build the nonce-wrapped user message.
	wrapped, _, err := llm.WrapUserInput(input)
	if err != nil {
		return "", fmt.Errorf("wrapping user input: %w", err)
	}

	systemPrompt := llm.BuildSystemPrompt(cfg.Choices)

	tool := buildTool(cfg.Choices)

	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: wrapped},
		},
		Tools: []llm.Tool{tool},
		ToolChoice: map[string]any{
			"type": "function",
			"function": map[string]string{
				"name": "select_choice",
			},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("calling LLM: %w", err)
	}

	return extractTag(resp, cfg.Choices), nil
}

// buildTool constructs the select_choice tool definition with an enum of all tags.
func buildTool(choices []config.Choice) llm.Tool {
	tags := make([]string, len(choices))
	for i, ch := range choices {
		tags[i] = ch.Tag
	}

	// Build the enum as []any for JSON compatibility.
	enum := make([]any, len(tags))
	for i, t := range tags {
		enum[i] = t
	}

	return llm.Tool{
		Type: "function",
		Function: llm.ToolFunction{
			Name:        "select_choice",
			Description: "Select the most appropriate choice for the user's input",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tag": map[string]any{
						"type":        "string",
						"enum":        enum,
						"description": "The selected tag",
					},
				},
				"required": []string{"tag"},
			},
		},
	}
}

// extractTag attempts to find the selected tag in the LLM response using
// several strategies, in priority order:
//  1. Tool call result (most reliable)
//  2. JSON object in content  {"tag": "..."}
//  3. Any known tag string appearing verbatim in the content
//  4. Last choice as default
func extractTag(resp *llm.ChatResponse, choices []config.Choice) string {
	if len(resp.Choices) == 0 {
		return defaultTag(choices)
	}
	msg := resp.Choices[0].Message

	// Strategy 1: tool call
	for _, tc := range msg.ToolCalls {
		if tc.Function.Name == "select_choice" {
			if tag := parseTagFromArgs(tc.Function.Arguments, choices); tag != "" {
				return tag
			}
		}
	}

	// Strategy 2: JSON content
	if tag := parseTagFromJSON(msg.Content, choices); tag != "" {
		return tag
	}

	// Strategy 3: verbatim tag in content
	if tag := findTagInText(msg.Content, choices); tag != "" {
		return tag
	}

	// Strategy 4: default
	return defaultTag(choices)
}

// parseTagFromArgs parses the function call arguments JSON for the "tag" field.
func parseTagFromArgs(arguments string, choices []config.Choice) string {
	var args map[string]string
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return ""
	}
	return validateTag(args["tag"], choices)
}

// parseTagFromJSON tries to decode the content string as JSON with a "tag" field.
func parseTagFromJSON(content string, choices []config.Choice) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	// Try the full content first.
	if tag := tryJSONTag(content, choices); tag != "" {
		return tag
	}

	// Try to extract the first JSON object from potentially noisy text.
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		if tag := tryJSONTag(content[start:end+1], choices); tag != "" {
			return tag
		}
	}

	return ""
}

func tryJSONTag(s string, choices []config.Choice) string {
	var obj map[string]string
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return ""
	}
	return validateTag(obj["tag"], choices)
}

// findTagInText searches the response text for any of the known tags.
func findTagInText(content string, choices []config.Choice) string {
	lower := strings.ToLower(content)
	for _, ch := range choices {
		if strings.Contains(lower, strings.ToLower(ch.Tag)) {
			return ch.Tag
		}
	}
	return ""
}

// validateTag returns tag if it matches one of the known choices, else "".
func validateTag(tag string, choices []config.Choice) string {
	for _, ch := range choices {
		if ch.Tag == tag {
			return tag
		}
	}
	return ""
}

// defaultTag returns the last choice's tag, acting as a catch-all default.
func defaultTag(choices []config.Choice) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[len(choices)-1].Tag
}
