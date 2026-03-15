// Package config handles loading and validation of the ai-choice configuration files.
//
// Configuration is split into two files:
//   - System config: LLM API settings (endpoint, api_key, model, timeouts)
//   - Choices config: classification choices (tag + description pairs)
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Choice represents a single classification option with a tag and its description.
type Choice struct {
	Tag         string `yaml:"tag"`
	Description string `yaml:"description"`
}

// Config holds the full runtime configuration, merged from the system and choices files.
type Config struct {
	// System settings — loaded from the system config file.

	// Endpoint is the base URL of the OpenAI-compatible API (e.g. "https://api.openai.com/v1").
	Endpoint string

	// APIKey is the resolved API key used for authentication.
	APIKey string

	// Model is the model identifier to use for inference.
	Model string

	// TimeoutSeconds is the per-request HTTP timeout in seconds.
	TimeoutSeconds int

	// MaxRetries is the maximum number of retry attempts on transient errors.
	MaxRetries int

	// Choices — loaded from the choices config file.
	Choices []Choice
}

// Timeout returns the configured HTTP timeout as a time.Duration.
func (c *Config) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// systemFile is the YAML structure of the system config file.
type systemFile struct {
	Endpoint       string `yaml:"endpoint"`
	APIKey         string `yaml:"api_key"`
	Model          string `yaml:"model"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
	MaxRetries     int    `yaml:"max_retries"`
}

// choicesFile is the YAML structure of the choices config file.
type choicesFile struct {
	Choices []Choice `yaml:"choices"`
}

// Load reads the system config from systemPath and the choices config from
// choicesPath, merges them, applies defaults, and validates the result.
func Load(systemPath, choicesPath string) (*Config, error) {
	sys, err := loadSystem(systemPath)
	if err != nil {
		return nil, err
	}

	chs, err := loadChoices(choicesPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Endpoint:       sys.Endpoint,
		APIKey:         sys.APIKey,
		Model:          sys.Model,
		TimeoutSeconds: sys.TimeoutSeconds,
		MaxRetries:     sys.MaxRetries,
		Choices:        chs.Choices,
	}

	applyDefaults(cfg)

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

// loadSystem reads and parses the system config file.
func loadSystem(path string) (systemFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return systemFile{}, fmt.Errorf("reading system config %q: %w", path, err)
	}

	var sys systemFile
	if err := yaml.Unmarshal(data, &sys); err != nil {
		return systemFile{}, fmt.Errorf("parsing system config %q: %w", path, err)
	}

	// Expand environment variable reference in api_key.
	if strings.HasPrefix(sys.APIKey, "$") {
		envName := strings.TrimPrefix(sys.APIKey, "$")
		val := os.Getenv(envName)
		if val == "" {
			return systemFile{}, fmt.Errorf("environment variable %q referenced by api_key is not set or empty", envName)
		}
		sys.APIKey = val
	}

	return sys, nil
}

// loadChoices reads and parses the choices config file.
func loadChoices(path string) (choicesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return choicesFile{}, fmt.Errorf("reading choices config %q: %w", path, err)
	}

	var chs choicesFile
	if err := yaml.Unmarshal(data, &chs); err != nil {
		return choicesFile{}, fmt.Errorf("parsing choices config %q: %w", path, err)
	}

	return chs, nil
}

// applyDefaults fills in zero values with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
}

// validate checks that all required fields are present and consistent.
func (c *Config) validate() error {
	var errs []string

	if strings.TrimSpace(c.Endpoint) == "" {
		errs = append(errs, "endpoint must not be empty")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		errs = append(errs, "api_key must not be empty (or set the referenced environment variable)")
	}
	if strings.TrimSpace(c.Model) == "" {
		errs = append(errs, "model must not be empty")
	}
	if len(c.Choices) < 1 {
		errs = append(errs, "at least one choice must be defined")
	}

	seen := make(map[string]bool)
	for i, ch := range c.Choices {
		if strings.TrimSpace(ch.Tag) == "" {
			errs = append(errs, fmt.Sprintf("choices[%d]: tag must not be empty", i))
		}
		if strings.TrimSpace(ch.Description) == "" {
			errs = append(errs, fmt.Sprintf("choices[%d]: description must not be empty", i))
		}
		if seen[ch.Tag] {
			errs = append(errs, fmt.Sprintf("choices[%d]: duplicate tag %q", i, ch.Tag))
		}
		seen[ch.Tag] = true
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
