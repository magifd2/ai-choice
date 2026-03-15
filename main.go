// ai-choice: a CLI tool that classifies free-form text into a predefined tag
// using an OpenAI-compatible LLM.
//
// Usage:
//
//	echo "What is the weather today?" | ai-choice -config config.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/magifd2/ai-choice/internal/classifier"
	"github.com/magifd2/ai-choice/internal/config"
	"github.com/magifd2/ai-choice/internal/llm"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "ai-choice %s\n", version)
		return 0
	}

	// Read user input from stdin.
	inputBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading stdin: %v\n", err)
		return 1
	}
	input := strings.TrimSpace(string(inputBytes))
	if input == "" {
		fmt.Fprintln(os.Stderr, "error: no input provided on stdin")
		return 1
	}

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config %q: %v\n", *configPath, err)
		return 1
	}

	// Build the LLM client.
	client := llm.NewClient(
		cfg.Endpoint,
		cfg.APIKey,
		cfg.Model,
		cfg.Timeout(),
		cfg.MaxRetries,
	)

	// Run classification.
	tag, err := classifier.Classify(context.Background(), input, cfg, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: classification failed: %v\n", err)
		return 1
	}

	// Output only the tag.
	fmt.Println(tag)
	return 0
}
