package aitokenizer

import (
	"fmt"
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

// Tokenizer provides functionality to count tokens in text content.
type Tokenizer struct {
	// Default tokenizer encoding to use
	defaultEncoding tokenizer.Encoding
}

// NewTokenizer creates a new tokenizer instance with default GPT-4 tokenizer.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		defaultEncoding: tokenizer.Cl100kBase,
	}
}

// NewTokenizerWithEncoding creates a new tokenizer instance with specified encoding.
func NewTokenizerWithEncoding(encoding tokenizer.Encoding) *Tokenizer {
	return &Tokenizer{
		defaultEncoding: encoding,
	}
}

// CountTokens estimates the number of tokens in the given content using tiktoken.
func (t *Tokenizer) CountTokens(content string) int {
	if content == "" {
		return 0
	}

	// Use tiktoken for accurate token counting
	tiktoken, err := tokenizer.Get(t.defaultEncoding)
	if err != nil {
		// Fallback to simple estimation if tiktoken fails
		return t.fallbackCount(content)
	}

	tokens, _, err := tiktoken.Encode(content)
	if err != nil {
		// Fallback to simple estimation if encoding fails
		return t.fallbackCount(content)
	}

	return len(tokens)
}

// CountTokensWithLanguage estimates tokens with language-specific considerations.
func (t *Tokenizer) CountTokensWithLanguage(content, language string) int {
	// For now, use the same tokenizer for all languages
	// In the future, we could use different tokenizers for different languages
	return t.CountTokens(content)
}

// EstimateTokensForFile estimates tokens for a file with given language.
func (t *Tokenizer) EstimateTokensForFile(content, language string) int {
	return t.CountTokensWithLanguage(content, language)
}

// CountTokensForModel counts tokens using a specific OpenAI model.
func (t *Tokenizer) CountTokensForModel(content string, model tokenizer.Model) (int, error) {
	if content == "" {
		return 0, nil
	}

	tiktoken, err := tokenizer.ForModel(model)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokenizer for model %s: %w", model, err)
	}

	tokens, _, err := tiktoken.Encode(content)
	if err != nil {
		return 0, fmt.Errorf("failed to encode content with model %s: %w", model, err)
	}

	return len(tokens), nil
}

// CountTokensForEncoding counts tokens using a specific encoding.
func (t *Tokenizer) CountTokensForEncoding(content string, encoding tokenizer.Encoding) (int, error) {
	if content == "" {
		return 0, nil
	}

	tiktoken, err := tokenizer.Get(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokenizer for encoding %s: %w", encoding, err)
	}

	tokens, _, err := tiktoken.Encode(content)
	if err != nil {
		return 0, fmt.Errorf("failed to encode content with encoding %s: %w", encoding, err)
	}

	return len(tokens), nil
}

// fallbackCount provides a simple fallback token counting method.
func (t *Tokenizer) fallbackCount(content string) int {
	// Simple word-based estimation as fallback
	words := strings.Fields(content)
	return len(words)
}
