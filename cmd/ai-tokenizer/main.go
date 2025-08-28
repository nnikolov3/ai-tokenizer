// AI Tokenizer - standalone token budgeting and estimation service
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	aitokenizer "github.com/nnikolov3/ai-tokenizer"
	"github.com/tiktoken-go/tokenizer"
)

const (
	defaultModel      = "gpt-4"
	defaultMaxContext = 8192
)

var (
	ErrTextRequired  = errors.New("text content is required for estimation")
	ErrModelRequired = errors.New("model name is required")
	ErrInvalidBudget = errors.New("budget must be a positive integer")
	ErrExceedsBudget = errors.New("text exceeds token budget")
)

type config struct {
	text      string
	model     string
	budget    int
	estimate  bool
	chunk     bool
	chunkSize int
	help      bool
	json      bool
	encoding  string
	decode    bool
	encode    bool
}

type TokenEstimate struct {
	Model        string      `json:"model"`
	Tokens       int         `json:"tokens"`
	Budget       int         `json:"budget,omitempty"`
	WithinBudget bool        `json:"within_budget,omitempty"`
	Chunks       []ChunkInfo `json:"chunks,omitempty"`
	Encoding     string      `json:"encoding,omitempty"`
	TokenIDs     []uint      `json:"token_ids,omitempty"`
	DecodedText  string      `json:"decoded_text,omitempty"`
}

type ChunkInfo struct {
	Index  int    `json:"index"`
	Tokens int    `json:"tokens"`
	Text   string `json:"text,omitempty"`
}

type ModelCapability struct {
	MaxContext int    `json:"max_context"`
	Vision     bool   `json:"vision"`
	Encoding   string `json:"encoding"`
}

func main() {
	config := parseFlags()

	if config.help {
		showHelp()
		return
	}

	if config.encode {
		runEncode(&config)
		return
	}

	if config.decode {
		runDecode(&config)
		return
	}

	if config.estimate {
		runEstimate(&config)
		return
	}

	if config.chunk {
		runChunking(&config)
		return
	}

	showHelp()
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.text, "text", "", "Text content to analyze (required)")
	flag.StringVar(&cfg.model, "model", defaultModel, "Model name for capability lookup")
	flag.IntVar(&cfg.budget, "budget", 0, "Token budget to check against")
	flag.BoolVar(&cfg.estimate, "estimate", false, "Estimate token count for text")
	flag.BoolVar(&cfg.chunk, "chunk", false, "Split text into chunks if over budget")
	flag.IntVar(&cfg.chunkSize, "chunk-size", 0, "Custom chunk size (uses model max if not specified)")
	flag.BoolVar(&cfg.json, "json", false, "Output results in JSON format")
	flag.BoolVar(&cfg.help, "help", false, "Show help")
	flag.BoolVar(&cfg.encode, "encode", false, "Encode text to tokens")
	flag.BoolVar(&cfg.decode, "decode", false, "Decode tokens to text")
	flag.StringVar(&cfg.encoding, "encoding", "", "Specific encoding to use (cl100k_base, gpt2, etc.)")
	flag.Parse()

	return cfg
}

func runEncode(cfg *config) {
	if cfg.text == "" {
		handleError(ErrTextRequired, cfg.json)
		return
	}

	var tokens []uint
	var err error

	if cfg.encoding != "" {
		// Use specific encoding
		encoding := tokenizer.Encoding(cfg.encoding)
		tokens, _, err = encodeWithEncoding(cfg.text, encoding)
	} else {
		// Use model-based encoding
		model := tokenizer.Model(cfg.model)
		tokens, _, err = encodeWithModel(cfg.text, model)
	}

	if err != nil {
		handleError(fmt.Errorf("encoding failed: %w", err), cfg.json)
		return
	}

	result := TokenEstimate{
		Model:    cfg.model,
		Tokens:   len(tokens),
		TokenIDs: tokens,
		Encoding: cfg.encoding,
	}

	outputResult(result, cfg.json)
}

func runDecode(cfg *config) {
	if cfg.text == "" {
		handleError(ErrTextRequired, cfg.json)
		return
	}

	// Parse token IDs from text (assuming comma-separated integers)
	tokenStrs := strings.Split(cfg.text, ",")
	var tokenIDs []uint
	for _, tokenStr := range tokenStrs {
		tokenStr = strings.TrimSpace(tokenStr)
		if tokenStr == "" {
			continue
		}
		var tokenID uint64
		_, err := fmt.Sscanf(tokenStr, "%d", &tokenID)
		if err != nil {
			handleError(fmt.Errorf("invalid token ID: %s", tokenStr), cfg.json)
			return
		}
		tokenIDs = append(tokenIDs, uint(tokenID))
	}

	var decodedText string
	var err error

	if cfg.encoding != "" {
		// Use specific encoding
		encoding := tokenizer.Encoding(cfg.encoding)
		decodedText, err = decodeWithEncoding(tokenIDs, encoding)
	} else {
		// Use model-based encoding
		model := tokenizer.Model(cfg.model)
		decodedText, err = decodeWithModel(tokenIDs, model)
	}

	if err != nil {
		handleError(fmt.Errorf("decoding failed: %w", err), cfg.json)
		return
	}

	result := TokenEstimate{
		Model:       cfg.model,
		Tokens:      len(tokenIDs),
		TokenIDs:    tokenIDs,
		DecodedText: decodedText,
		Encoding:    cfg.encoding,
	}

	outputResult(result, cfg.json)
}

func runEstimate(cfg *config) {
	if cfg.text == "" {
		handleError(ErrTextRequired, cfg.json)
		return
	}

	capability := getModelCapability(cfg.model)
	var tokens int
	var err error

	if cfg.encoding != "" {
		// Use specific encoding
		encoding := tokenizer.Encoding(cfg.encoding)
		tokens, err = countTokensWithEncoding(cfg.text, encoding)
	} else {
		// Use model-based encoding
		model := tokenizer.Model(cfg.model)
		tokens, err = countTokensWithModel(cfg.text, model)
	}

	if err != nil {
		// Fallback to simple estimation
		tokens = estimateTokens(cfg.text)
	}

	result := TokenEstimate{
		Model:    cfg.model,
		Tokens:   tokens,
		Encoding: cfg.encoding,
	}

	if cfg.budget > 0 {
		result.Budget = cfg.budget
		result.WithinBudget = tokens <= cfg.budget
	} else if capability.MaxContext > 0 {
		result.Budget = capability.MaxContext
		result.WithinBudget = tokens <= capability.MaxContext
	}

	outputResult(result, cfg.json)
}

func runChunking(cfg *config) {
	if cfg.text == "" {
		handleError(ErrTextRequired, cfg.json)
		return
	}

	capability := getModelCapability(cfg.model)
	chunkSize := cfg.chunkSize
	if chunkSize == 0 {
		if cfg.budget > 0 {
			chunkSize = cfg.budget
		} else {
			chunkSize = capability.MaxContext
		}
	}

	if chunkSize <= 0 {
		chunkSize = defaultMaxContext
	}

	chunks := chunkText(cfg.text, chunkSize, cfg.model, cfg.encoding)

	result := TokenEstimate{
		Model:        cfg.model,
		Tokens:       estimateTokens(cfg.text),
		Budget:       chunkSize,
		Chunks:       chunks,
		WithinBudget: len(chunks) == 1,
		Encoding:     cfg.encoding,
	}

	outputResult(result, cfg.json)
}

func encodeWithModel(text string, model tokenizer.Model) ([]uint, []string, error) {
	codec, err := tokenizer.ForModel(model)
	if err != nil {
		return nil, nil, err
	}
	return codec.Encode(text)
}

func encodeWithEncoding(text string, encoding tokenizer.Encoding) ([]uint, []string, error) {
	codec, err := tokenizer.Get(encoding)
	if err != nil {
		return nil, nil, err
	}
	return codec.Encode(text)
}

func decodeWithModel(tokenIDs []uint, model tokenizer.Model) (string, error) {
	codec, err := tokenizer.ForModel(model)
	if err != nil {
		return "", err
	}
	return codec.Decode(tokenIDs)
}

func decodeWithEncoding(tokenIDs []uint, encoding tokenizer.Encoding) (string, error) {
	codec, err := tokenizer.Get(encoding)
	if err != nil {
		return "", err
	}
	return codec.Decode(tokenIDs)
}

func countTokensWithModel(text string, model tokenizer.Model) (int, error) {
	codec, err := tokenizer.ForModel(model)
	if err != nil {
		return 0, err
	}
	return codec.Count(text)
}

func countTokensWithEncoding(text string, encoding tokenizer.Encoding) (int, error) {
	codec, err := tokenizer.Get(encoding)
	if err != nil {
		return 0, err
	}
	return codec.Count(text)
}

func estimateTokens(text string) int {
	// Fallback to simple token estimation: ~4 characters per token on average
	// This is a rough approximation - real tokenization would be more accurate
	return len(text) / 4
}

func chunkText(text string, maxTokens int, model, encoding string) []ChunkInfo {
	// Use the tokenizer library for accurate chunking
	var t *aitokenizer.Tokenizer

	if encoding != "" {
		// Convert string to tokenizer.Encoding type
		tokenizerEncoding := tokenizer.Encoding(encoding)
		t = aitokenizer.NewTokenizerWithEncoding(tokenizerEncoding)
	} else {
		t = aitokenizer.NewTokenizer()
	}

	// Simple word-based chunking with token counting
	words := strings.Fields(text)
	var chunks []ChunkInfo
	currentChunk := ""
	chunkIndex := 0

	for _, word := range words {
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += " "
		}
		testChunk += word

		// Count tokens in the test chunk
		tokenCount := t.CountTokens(testChunk)

		if tokenCount > maxTokens && currentChunk != "" {
			// Current chunk is full, finalize it
			chunks = append(chunks, ChunkInfo{
				Index:  chunkIndex,
				Tokens: t.CountTokens(currentChunk),
				Text:   currentChunk,
			})
			currentChunk = word
			chunkIndex++
		} else {
			currentChunk = testChunk
		}
	}

	// Add the final chunk if there's remaining content
	if currentChunk != "" {
		chunks = append(chunks, ChunkInfo{
			Index:  chunkIndex,
			Tokens: t.CountTokens(currentChunk),
			Text:   currentChunk,
		})
	}

	return chunks
}

func getModelCapability(model string) ModelCapability {
	// Capability map for common models with their encodings
	capabilities := map[string]ModelCapability{
		"gpt-4":             {MaxContext: 8192, Vision: true, Encoding: "cl100k_base"},
		"gpt-4-32k":         {MaxContext: 32768, Vision: true, Encoding: "cl100k_base"},
		"gpt-4o":            {MaxContext: 128000, Vision: true, Encoding: "cl100k_base"},
		"gpt-3.5-turbo":     {MaxContext: 4096, Vision: false, Encoding: "cl100k_base"},
		"gpt-3.5-turbo-16k": {MaxContext: 16384, Vision: false, Encoding: "cl100k_base"},
		"gpt-2":             {MaxContext: 2048, Vision: false, Encoding: "gpt2"},
		"text-davinci-003":  {MaxContext: 4097, Vision: false, Encoding: "p50k_base"},
		"text-davinci-002":  {MaxContext: 4097, Vision: false, Encoding: "p50k_base"},
		"code-davinci-002":  {MaxContext: 8001, Vision: false, Encoding: "p50k_base"},
		"code-davinci-001":  {MaxContext: 8001, Vision: false, Encoding: "p50k_base"},
		"text-curie-001":    {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"text-babbage-001":  {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"text-ada-001":      {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"davinci":           {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"curie":             {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"babbage":           {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
		"ada":               {MaxContext: 2049, Vision: false, Encoding: "r50k_base"},
	}

	if capability, exists := capabilities[model]; exists {
		return capability
	}

	// Default capability for unknown models
	return ModelCapability{MaxContext: defaultMaxContext, Vision: false, Encoding: "cl100k_base"}
}

func outputResult(result TokenEstimate, useJSON bool) {
	if useJSON {
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Printf("Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	} else {
		fmt.Printf("Model: %s\n", result.Model)
		if result.Encoding != "" {
			fmt.Printf("Encoding: %s\n", result.Encoding)
		}
		fmt.Printf("Tokens: %d\n", result.Tokens)

		if result.TokenIDs != nil {
			fmt.Printf("Token IDs: %v\n", result.TokenIDs)
		}

		if result.DecodedText != "" {
			fmt.Printf("Decoded text: %s\n", result.DecodedText)
		}

		if result.Budget > 0 {
			fmt.Printf("Budget: %d tokens\n", result.Budget)
			if result.WithinBudget {
				fmt.Println("Status: ✓ Within budget")
			} else {
				fmt.Printf("Status: ✗ Exceeds budget by %d tokens\n", result.Tokens-result.Budget)
			}
		}

		if len(result.Chunks) > 1 {
			fmt.Printf("\nText split into %d chunks:\n", len(result.Chunks))
			for _, chunk := range result.Chunks {
				fmt.Printf("  Chunk %d: %d tokens\n", chunk.Index+1, chunk.Tokens)
			}
		}
	}
}

func handleError(err error, useJSON bool) {
	if useJSON {
		errorOutput := map[string]string{"error": err.Error()}
		output, _ := json.MarshalIndent(errorOutput, "", "  ")
		fmt.Println(string(output))
	} else {
		log.Printf("Error: %v\n", err)
	}
	os.Exit(1)
}

func showHelp() {
	fmt.Print(`AI Tokenizer - Standalone token budgeting and estimation service

Usage: ai-tokenizer [options]

Options:
  -text TEXT        Text content to analyze (required)
  -model NAME       Model name for capability lookup (default: gpt-4)
  -budget NUM       Token budget to check against
  -estimate         Estimate token count for the given text
  -chunk            Split text into chunks if over budget
  -chunk-size NUM   Custom chunk size (uses model max if not specified)
  -encode           Encode text to token IDs
  -decode           Decode token IDs to text
  -encoding NAME    Specific encoding to use (cl100k_base, gpt2, p50k_base, r50k_base)
  -json             Output results in JSON format
  -help             Show this help message

Examples:
  # Estimate tokens for text
  ai-tokenizer -estimate -text "Hello world"
  
  # Check if text fits within budget
  ai-tokenizer -estimate -text "Long text..." -budget 1000
  
  # Encode text to tokens
  ai-tokenizer -encode -text "Hello world"
  
  # Decode tokens to text
  ai-tokenizer -decode -text "15496,1917" -model gpt-4
  
  # Use specific encoding
  ai-tokenizer -encode -text "Hello world" -encoding cl100k_base
  
  # Chunk text for specific model
  ai-tokenizer -chunk -model gpt-4 -text "Very long document..."
  
  # Custom chunk size with JSON output
  ai-tokenizer -chunk -chunk-size 500 -json -text "Document content..."

Supported Models:
  - gpt-4, gpt-4-32k, gpt-4o (cl100k_base)
  - gpt-3.5-turbo, gpt-3.5-turbo-16k (cl100k_base)
  - gpt-2 (gpt2)
  - text-davinci-003, text-davinci-002 (p50k_base)
  - code-davinci-002, code-davinci-001 (p50k_base)
  - text-curie-001, text-babbage-001, text-ada-001 (r50k_base)
  - davinci, curie, babbage, ada (r50k_base)

Supported Encodings:
  - cl100k_base: GPT-4, GPT-3.5-turbo
  - gpt2: GPT-2
  - p50k_base: Code models
  - r50k_base: Older GPT models
  - p50k_edit: Edit models
  - o200k_base: Latest models

Exit codes:
  0  Success
  1  Error (invalid arguments, processing failed, etc.)`)
}
