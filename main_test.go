package main

import (
	"strings"
	"testing"
)

func TestReadInput_Normal(t *testing.T) {
	got, err := readInput(strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestReadInput_TrimsWhitespace(t *testing.T) {
	got, err := readInput(strings.NewReader("  hello\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestReadInput_Empty(t *testing.T) {
	_, err := readInput(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestReadInput_WhitespaceOnly(t *testing.T) {
	_, err := readInput(strings.NewReader("   \n\t  "))
	if err == nil {
		t.Fatal("expected error for whitespace-only input, got nil")
	}
}

func TestReadInput_ExactlyAtLimit(t *testing.T) {
	// maxInputBytes of non-whitespace should succeed.
	_, err := readInput(strings.NewReader(strings.Repeat("x", maxInputBytes)))
	if err != nil {
		t.Errorf("unexpected error at exact limit: %v", err)
	}
}

func TestReadInput_OneByteOverLimit(t *testing.T) {
	_, err := readInput(strings.NewReader(strings.Repeat("x", maxInputBytes+1)))
	if err == nil {
		t.Fatal("expected error for oversized input, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error message should mention 'exceeds', got: %v", err)
	}
}

func TestReadInput_WellOverLimit(t *testing.T) {
	_, err := readInput(strings.NewReader(strings.Repeat("a", maxInputBytes*2)))
	if err == nil {
		t.Fatal("expected error for input well over limit, got nil")
	}
}
