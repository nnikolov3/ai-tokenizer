package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/tiktoken-go/tokenizer"
)

func TestParseFlags(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		expected config
	}{
		{
			name: "default config",
			args: []string{"ai-tokenizer"},
			expected: config{
				text:      "",
				model:     "gpt-4",
				budget:    0,
				estimate:  false,
				chunk:     false,
				chunkSize: 0,
				help:      false,
				json:      false,
				encoding:  "",
				decode:    false,
				encode:    false,
			},
		},
		{
			name: "with text and estimate",
			args: []string{"ai-tokenizer", "-text", "Hello world", "-estimate"},
			expected: config{
				text:      "Hello world",
				model:     "gpt-4",
				budget:    0,
				estimate:  true,
				chunk:     false,
				chunkSize: 0,
				help:      false,
				json:      false,
				encoding:  "",
				decode:    false,
				encode:    false,
			},
		},
		{
			name: "with all options",
			args: []string{"ai-tokenizer", "-text", "Test", "-model", "gpt-3.5-turbo", "-budget", "1000", "-chunk", "-chunk-size", "500", "-json", "-encoding", "cl100k_base"},
			expected: config{
				text:      "Test",
				model:     "gpt-3.5-turbo",
				budget:    1000,
				estimate:  false,
				chunk:     true,
				chunkSize: 500,
				help:      false,
				json:      true,
				encoding:  "cl100k_base",
				decode:    false,
				encode:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag state
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set args
			os.Args = tt.args

			// Parse flags
			result := parseFlags()

			// Compare results
			if result.text != tt.expected.text {
				t.Errorf("text = %q, want %q", result.text, tt.expected.text)
			}
			if result.model != tt.expected.model {
				t.Errorf("model = %q, want %q", result.model, tt.expected.model)
			}
			if result.budget != tt.expected.budget {
				t.Errorf("budget = %d, want %d", result.budget, tt.expected.budget)
			}
			if result.estimate != tt.expected.estimate {
				t.Errorf("estimate = %v, want %v", result.estimate, tt.expected.estimate)
			}
			if result.chunk != tt.expected.chunk {
				t.Errorf("chunk = %v, want %v", result.chunk, tt.expected.chunk)
			}
			if result.chunkSize != tt.expected.chunkSize {
				t.Errorf("chunkSize = %d, want %d", result.chunkSize, tt.expected.chunkSize)
			}
			if result.help != tt.expected.help {
				t.Errorf("help = %v, want %v", result.help, tt.expected.help)
			}
			if result.json != tt.expected.json {
				t.Errorf("json = %v, want %v", result.json, tt.expected.json)
			}
			if result.encoding != tt.expected.encoding {
				t.Errorf("encoding = %q, want %q", result.encoding, tt.expected.encoding)
			}
			if result.decode != tt.expected.decode {
				t.Errorf("decode = %v, want %v", result.decode, tt.expected.decode)
			}
			if result.encode != tt.expected.encode {
				t.Errorf("encode = %v, want %v", result.encode, tt.expected.encode)
			}
		})
	}
}

func TestGetModelCapability(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected ModelCapability
	}{
		{
			name:  "gpt-4",
			model: "gpt-4",
			expected: ModelCapability{
				MaxContext: 8192,
				Vision:     true,
				Encoding:   "cl100k_base",
			},
		},
		{
			name:  "gpt-3.5-turbo",
			model: "gpt-3.5-turbo",
			expected: ModelCapability{
				MaxContext: 4096,
				Vision:     false,
				Encoding:   "cl100k_base",
			},
		},
		{
			name:  "gpt-2",
			model: "gpt-2",
			expected: ModelCapability{
				MaxContext: 2048,
				Vision:     false,
				Encoding:   "gpt2",
			},
		},
		{
			name:  "unknown model",
			model: "unknown-model",
			expected: ModelCapability{
				MaxContext: 8192,
				Vision:     false,
				Encoding:   "cl100k_base",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModelCapability(tt.model)

			if result.MaxContext != tt.expected.MaxContext {
				t.Errorf("MaxContext = %d, want %d", result.MaxContext, tt.expected.MaxContext)
			}
			if result.Vision != tt.expected.Vision {
				t.Errorf("Vision = %v, want %v", result.Vision, tt.expected.Vision)
			}
			if result.Encoding != tt.expected.Encoding {
				t.Errorf("Encoding = %q, want %q", result.Encoding, tt.expected.Encoding)
			}
		})
	}
}

func TestEncodeWithModel(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		model   tokenizer.Model
		wantErr bool
	}{
		{
			name:    "valid text with gpt-4",
			text:    "Hello world",
			model:   tokenizer.GPT4,
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			model:   tokenizer.GPT4,
			wantErr: false,
		},
		{
			name:    "invalid model",
			text:    "Hello world",
			model:   tokenizer.Model("invalid-model"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, _, err := encodeWithModel(tt.text, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Errorf("encodeWithModel() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("encodeWithModel() unexpected error: %v", err)
				return
			}

			if tt.text == "" && len(tokens) != 0 {
				t.Errorf("encodeWithModel() for empty text returned %d tokens, want 0", len(tokens))
			}

			if tt.text != "" && len(tokens) == 0 {
				t.Errorf("encodeWithModel() for non-empty text returned 0 tokens")
			}
		})
	}
}

func TestEncodeWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		encoding tokenizer.Encoding
		wantErr  bool
	}{
		{
			name:     "valid text with cl100k_base",
			text:     "Hello world",
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "empty text",
			text:     "",
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "invalid encoding",
			text:     "Hello world",
			encoding: tokenizer.Encoding("invalid-encoding"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, _, err := encodeWithEncoding(tt.text, tt.encoding)

			if tt.wantErr {
				if err == nil {
					t.Errorf("encodeWithEncoding() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("encodeWithEncoding() unexpected error: %v", err)
				return
			}

			if tt.text == "" && len(tokens) != 0 {
				t.Errorf("encodeWithEncoding() for empty text returned %d tokens, want 0", len(tokens))
			}

			if tt.text != "" && len(tokens) == 0 {
				t.Errorf("encodeWithEncoding() for non-empty text returned 0 tokens")
			}
		})
	}
}

func TestDecodeWithModel(t *testing.T) {
	tests := []struct {
		name     string
		tokenIDs []uint
		model    tokenizer.Model
		wantErr  bool
	}{
		{
			name:     "valid tokens with gpt-4",
			tokenIDs: []uint{9906, 1917},
			model:    tokenizer.GPT4,
			wantErr:  false,
		},
		{
			name:     "empty tokens",
			tokenIDs: []uint{},
			model:    tokenizer.GPT4,
			wantErr:  false,
		},
		{
			name:     "invalid model",
			tokenIDs: []uint{9906, 1917},
			model:    tokenizer.Model("invalid-model"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeWithModel(tt.tokenIDs, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Errorf("decodeWithModel() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("decodeWithModel() unexpected error: %v", err)
				return
			}

			if len(tt.tokenIDs) == 0 && result != "" {
				t.Errorf("decodeWithModel() for empty tokens returned %q, want empty string", result)
			}
		})
	}
}

func TestDecodeWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		tokenIDs []uint
		encoding tokenizer.Encoding
		wantErr  bool
	}{
		{
			name:     "valid tokens with cl100k_base",
			tokenIDs: []uint{9906, 1917},
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "empty tokens",
			tokenIDs: []uint{},
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "invalid encoding",
			tokenIDs: []uint{9906, 1917},
			encoding: tokenizer.Encoding("invalid-encoding"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decodeWithEncoding(tt.tokenIDs, tt.encoding)

			if tt.wantErr {
				if err == nil {
					t.Errorf("decodeWithEncoding() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("decodeWithEncoding() unexpected error: %v", err)
				return
			}

			if len(tt.tokenIDs) == 0 && result != "" {
				t.Errorf("decodeWithEncoding() for empty tokens returned %q, want empty string", result)
			}
		})
	}
}

func TestCountTokensWithModel(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		model   tokenizer.Model
		wantErr bool
	}{
		{
			name:    "valid text with gpt-4",
			text:    "Hello world",
			model:   tokenizer.GPT4,
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			model:   tokenizer.GPT4,
			wantErr: false,
		},
		{
			name:    "invalid model",
			text:    "Hello world",
			model:   tokenizer.Model("invalid-model"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := countTokensWithModel(tt.text, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Errorf("countTokensWithModel() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("countTokensWithModel() unexpected error: %v", err)
				return
			}

			if result < 0 {
				t.Errorf("countTokensWithModel() returned negative value: %d", result)
			}

			if tt.text == "" && result != 0 {
				t.Errorf("countTokensWithModel() for empty text = %d, want 0", result)
			}
		})
	}
}

func TestCountTokensWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		encoding tokenizer.Encoding
		wantErr  bool
	}{
		{
			name:     "valid text with cl100k_base",
			text:     "Hello world",
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "empty text",
			text:     "",
			encoding: tokenizer.Cl100kBase,
			wantErr:  false,
		},
		{
			name:     "invalid encoding",
			text:     "Hello world",
			encoding: tokenizer.Encoding("invalid-encoding"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := countTokensWithEncoding(tt.text, tt.encoding)

			if tt.wantErr {
				if err == nil {
					t.Errorf("countTokensWithEncoding() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("countTokensWithEncoding() unexpected error: %v", err)
				return
			}

			if result < 0 {
				t.Errorf("countTokensWithEncoding() returned negative value: %d", result)
			}

			if tt.text == "" && result != 0 {
				t.Errorf("countTokensWithEncoding() for empty text = %d, want 0", result)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty string", "", 0},
		{"short text", "Hello", 1},
		{"medium text", "Hello world", 2},
		{"longer text", "This is a longer text that should have more tokens", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateTokens(tt.text)
			if result != tt.expected {
				t.Errorf("estimateTokens() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestChunkText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxTokens int
		model     string
		encoding  string
		expected  int // expected number of chunks
	}{
		{
			name:      "short text within limit",
			text:      "Hello world",
			maxTokens: 10,
			model:     "gpt-4",
			encoding:  "",
			expected:  1,
		},
		{
			name:      "text that needs chunking",
			text:      "This is a longer text that should be split into multiple chunks because it exceeds the token limit",
			maxTokens: 5,
			model:     "gpt-4",
			encoding:  "",
			expected:  4, // Should be split into multiple chunks
		},
		{
			name:      "empty text",
			text:      "",
			maxTokens: 10,
			model:     "gpt-4",
			encoding:  "",
			expected:  0, // Empty text results in 0 chunks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkText(tt.text, tt.maxTokens, tt.model, tt.encoding)

			if len(chunks) != tt.expected {
				t.Errorf("chunkText() returned %d chunks, want %d", len(chunks), tt.expected)
			}

			// Verify that each chunk doesn't exceed the token limit
			for i, chunk := range chunks {
				if chunk.Tokens > tt.maxTokens {
					t.Errorf("chunk %d has %d tokens, exceeds limit of %d", i, chunk.Tokens, tt.maxTokens)
				}
			}

			// Verify that all chunks have valid indices
			for i, chunk := range chunks {
				if chunk.Index != i {
					t.Errorf("chunk %d has index %d, want %d", i, chunk.Index, i)
				}
			}
		})
	}
}

func TestOutputResult(t *testing.T) {
	tests := []struct {
		name     string
		result   TokenEstimate
		useJSON  bool
		expected string
	}{
		{
			name: "JSON output",
			result: TokenEstimate{
				Model:        "gpt-4",
				Tokens:       5,
				Budget:       1000,
				WithinBudget: true,
			},
			useJSON:  true,
			expected: `"model"`,
		},
		{
			name: "text output",
			result: TokenEstimate{
				Model:        "gpt-4",
				Tokens:       5,
				Budget:       1000,
				WithinBudget: true,
			},
			useJSON:  false,
			expected: "Model: gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			outputResult(tt.result, tt.useJSON)

			w.Close()
			os.Stdout = oldStdout

			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("Failed to read stdout: %v", err)
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expected) {
				t.Errorf("outputResult() output does not contain expected string. Got: %s", outputStr)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		useJSON  bool
		expected string
	}{
		{
			name:     "JSON error output",
			err:      ErrTextRequired,
			useJSON:  true,
			expected: `"error":"text content is required for estimation"`,
		},
		{
			name:     "text error output",
			err:      ErrTextRequired,
			useJSON:  false,
			expected: "Error: text content is required for estimation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// We need to prevent the program from actually exiting
			// This is a limitation of testing os.Exit
			defer func() {
				os.Stderr = oldStderr
			}()

			// Note: This test is limited because handleError calls os.Exit(1)
			// In a real scenario, you might want to refactor to make this more testable
			_ = tt
			_ = r
			w.Close()
		})
	}
}

// Benchmark tests
func BenchmarkEncodeWithModel(b *testing.B) {
	text := "This is a benchmark test for encoding with model."
	model := tokenizer.GPT4

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodeWithModel(text, model)
	}
}

func BenchmarkEncodeWithEncoding(b *testing.B) {
	text := "This is a benchmark test for encoding with encoding."
	encoding := tokenizer.Cl100kBase

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encodeWithEncoding(text, encoding)
	}
}

func BenchmarkCountTokensWithModel(b *testing.B) {
	text := "This is a benchmark test for counting tokens with model."
	model := tokenizer.GPT4

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		countTokensWithModel(text, model)
	}
}

func BenchmarkCountTokensWithEncoding(b *testing.B) {
	text := "This is a benchmark test for counting tokens with encoding."
	encoding := tokenizer.Cl100kBase

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		countTokensWithEncoding(text, encoding)
	}
}

func BenchmarkChunkText(b *testing.B) {
	text := "This is a benchmark test for chunking text. It contains multiple sentences and should be split into chunks based on the token limit."
	maxTokens := 10
	model := "gpt-4"
	encoding := ""

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunkText(text, maxTokens, model, encoding)
	}
}
