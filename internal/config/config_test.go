package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/magifd2/ai-choice/internal/config"
)

// writeTemp writes content to a temporary YAML file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

const validYAML = `
endpoint: "https://api.openai.com/v1"
api_key: "sk-test"
model: "gpt-4o-mini"
timeout_seconds: 10
max_retries: 2
choices:
  - tag: "weather"
    description: "Weather forecast questions"
  - tag: "default"
    description: "Anything else"
`

func TestLoad_Valid(t *testing.T) {
	path := writeTemp(t, validYAML)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Endpoint != "https://api.openai.com/v1" {
		t.Errorf("Endpoint = %q, want %q", cfg.Endpoint, "https://api.openai.com/v1")
	}
	if cfg.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test")
	}
	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4o-mini")
	}
	if cfg.TimeoutSeconds != 10 {
		t.Errorf("TimeoutSeconds = %d, want 10", cfg.TimeoutSeconds)
	}
	if cfg.MaxRetries != 2 {
		t.Errorf("MaxRetries = %d, want 2", cfg.MaxRetries)
	}
	if len(cfg.Choices) != 2 {
		t.Fatalf("len(Choices) = %d, want 2", len(cfg.Choices))
	}
	if cfg.Choices[0].Tag != "weather" {
		t.Errorf("Choices[0].Tag = %q, want %q", cfg.Choices[0].Tag, "weather")
	}
}

func TestLoad_EnvVarAPIKey(t *testing.T) {
	const envName = "TEST_AI_CHOICE_APIKEY"
	t.Setenv(envName, "sk-from-env")

	yaml := strings.ReplaceAll(validYAML, `"sk-test"`, `"$`+envName+`"`)
	path := writeTemp(t, yaml)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.APIKey != "sk-from-env" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-from-env")
	}
}

func TestLoad_EnvVarAPIKey_Missing(t *testing.T) {
	const envName = "TEST_AI_CHOICE_MISSING_KEY_XYZ"
	os.Unsetenv(envName)

	yaml := strings.ReplaceAll(validYAML, `"sk-test"`, `"$`+envName+`"`)
	path := writeTemp(t, yaml)

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load() expected error for missing env var, got nil")
	}
	if !strings.Contains(err.Error(), envName) {
		t.Errorf("error message should mention %q, got: %v", envName, err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("Load() expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTemp(t, "this: is: not: valid: yaml: :")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}

func TestLoad_MissingEndpoint(t *testing.T) {
	yaml := `
api_key: "sk-test"
model: "gpt-4o-mini"
choices:
  - tag: "a"
    description: "desc"
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load() expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "endpoint") {
		t.Errorf("error should mention endpoint, got: %v", err)
	}
}

func TestLoad_NoChoices(t *testing.T) {
	yaml := `
endpoint: "https://api.openai.com/v1"
api_key: "sk-test"
model: "gpt-4o-mini"
choices: []
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load() expected validation error for empty choices, got nil")
	}
}

func TestLoad_DuplicateTag(t *testing.T) {
	yaml := `
endpoint: "https://api.openai.com/v1"
api_key: "sk-test"
model: "gpt-4o-mini"
choices:
  - tag: "dup"
    description: "first"
  - tag: "dup"
    description: "second"
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load() expected validation error for duplicate tag, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error should mention 'duplicate', got: %v", err)
	}
}

func TestLoad_DefaultTimeout(t *testing.T) {
	yaml := `
endpoint: "https://api.openai.com/v1"
api_key: "sk-test"
model: "gpt-4o-mini"
choices:
  - tag: "a"
    description: "desc"
`
	path := writeTemp(t, yaml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.Timeout() != 30*time.Second {
		t.Errorf("Timeout() = %v, want 30s", cfg.Timeout())
	}
}
