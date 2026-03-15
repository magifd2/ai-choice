// Package config handles loading and validation of the ai-choice configuration file.
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

// Config holds the full application configuration loaded from a YAML file.
type Config struct {
	// Endpoint is the base URL of the OpenAI-compatible API (e.g. "https://api.openai.com/v1").
	Endpoint string `yaml:"endpoint"`

	// APIKey is the API key used for authentication. If the value starts with "$",
	// it is treated as an environment variable name and resolved at load time.
	APIKey string `yaml:"api_key"`

	// Model is the model identifier to use for inference.
	Model string `yaml:"model"`

	// TimeoutSeconds is the per-request HTTP timeout in seconds.
	TimeoutSeconds int `yaml:"timeout_seconds"`

	// MaxRetries is the maximum number of retry attempts on transient errors.
	MaxRetries int `yaml:"max_retries"`

	// Choices is the ordered list of classification options.
	Choices []Choice `yaml:"choices"`
}

// Timeout returns the configured HTTP timeout as a time.Duration.
func (c *Config) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Load reads a YAML config file from path, expands environment variables,
// and validates the resulting Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Expand environment variable references in APIKey.
	if strings.HasPrefix(cfg.APIKey, "$") {
		envName := strings.TrimPrefix(cfg.APIKey, "$")
		val := os.Getenv(envName)
		if val == "" {
			return nil, fmt.Errorf("environment variable %q referenced by api_key is not set or empty", envName)
		}
		cfg.APIKey = val
	}

	// Apply sensible defaults.
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
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
