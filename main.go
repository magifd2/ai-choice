// ai-choice: a CLI tool that classifies free-form text into a predefined tag
// using an OpenAI-compatible LLM.
//
// Usage:
//
//	echo "What is the weather today?" | ai-choice -system system.yaml -choices choices.yaml
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

// maxInputBytes is the hard limit on stdin input size.
// It prevents memory exhaustion (DoS) and avoids exceeding LLM context windows.
const maxInputBytes = 128 * 1024 // 128 KB

func main() {
	os.Exit(run())
}

func run() int {
	systemPath := flag.String("system", "system.yaml", "path to system config file (LLM API settings)")
	choicesPath := flag.String("choices", "choices.yaml", "path to choices config file (classification options)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "ai-choice %s\n", version)
		return 0
	}

	// Read user input from stdin with a size cap.
	input, err := readInput(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Load configuration from separate system and choices files.
	cfg, err := config.Load(*systemPath, *choicesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading config: %v\n", err)
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

// readInput reads from r up to maxInputBytes.
// It returns an error if the input is empty (after trimming) or exceeds the limit.
func readInput(r io.Reader) (string, error) {
	// Read one byte beyond the limit so we can detect overflow.
	lr := io.LimitReader(r, maxInputBytes+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	if len(b) > maxInputBytes {
		return "", fmt.Errorf("input exceeds maximum allowed size of %d KB", maxInputBytes/1024)
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("no input provided on stdin")
	}
	return s, nil
}
