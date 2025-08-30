// Package tokenizer provides simple token estimation functionality for text processing.
// It implements a basic tokenization strategy where approximately 2 characters equal 1
// token, and special characters (whitespace, punctuation, symbols) count as 1 token each.
package tokenizer

import (
	"math"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Tokenizer implements simple token estimation.
type Tokenizer struct {
	model string
}

const (
	// DefaultModel is the default tokenizer model name.
	DefaultModel = "simple"
	// CharsPerToken defines the approximate character-to-token ratio for regular
	// characters.
	CharsPerToken      = 2.0
	maxASCII      rune = 0x7F

	// Special character folding constants.
	ligatureSharpS = "ss"
	ligatureAE     = "ae"
	ligatureOE     = "oe"
	ligatureO      = "o"
	ligatureTH     = "th"
	ligatureD      = "d"
)

// NewTokenizer creates a new simple tokenizer instance.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{model: DefaultModel}
}

// EstimateTokens estimates tokens using: 2 chars = 1 token, special chars = 1 token each.
func (t *Tokenizer) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	normalized := t.Normalize(text)

	return t.countTokensFromNormalizedText(normalized)
}

// Normalize converts non-ASCII characters to their ASCII equivalents.
func (t *Tokenizer) Normalize(text string) string {
	if text == "" {
		return ""
	}

	return t.processText(norm.NFD.String(text))
}

// GetModel returns the tokenizer model name.
func (t *Tokenizer) GetModel() string {
	return t.model
}

// processText handles the main normalization logic.
func (t *Tokenizer) processText(nfd string) string {
	var builder strings.Builder
	builder.Grow(len(nfd))

	for _, r := range nfd {
		if out := normalizeRune(r); out != "" {
			builder.WriteString(out)
		}
	}

	return builder.String()
}

func (t *Tokenizer) countTokensFromNormalizedText(normalized string) int {
	tokenCount := 0
	charCount := 0

	for _, r := range normalized {
		if isSpecialChar(r) {
			tokenCount += t.addAccumulatedCharTokens(charCount)

			charCount = 0
			tokenCount++

			continue
		}

		charCount++
	}

	tokenCount += t.addAccumulatedCharTokens(charCount)

	return tokenCount
}

func (t *Tokenizer) addAccumulatedCharTokens(charCount int) int {
	if charCount <= 0 {
		return 0
	}

	return int(math.Ceil(float64(charCount) / CharsPerToken))
}

// normalizeRune converts a rune to its ASCII representation.
func normalizeRune(inputRune rune) string {
	if unicode.Is(unicode.Mn, inputRune) {
		return ""
	}

	if inputRune <= maxASCII {
		return string(inputRune)
	}

	return foldSpecialRune(inputRune)
}

// foldSpecialRune handles Unicode character folding to ASCII.
func foldSpecialRune(inputRune rune) string {
	// specialRuneMap maps Unicode characters to their ASCII equivalents.
	specialRuneMap := map[rune]string{
		'ß': ligatureSharpS,
		'Æ': ligatureAE,
		'æ': ligatureAE,
		'Œ': ligatureOE,
		'œ': ligatureOE,
		'Ø': ligatureO,
		'ø': ligatureO,
		'Þ': ligatureTH,
		'þ': ligatureTH,
		'Ð': ligatureD,
		'ð': ligatureD,
	}

	if replacement, exists := specialRuneMap[inputRune]; exists {
		return replacement
	}

	return ""
}

func isSpecialChar(inputRune rune) bool {
	return !unicode.IsLetter(inputRune) && !unicode.IsDigit(inputRune)
}
