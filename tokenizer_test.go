package tokenizer_test

import (
	"strings"
	"testing"

	tokenizer "github.com/nnikolov3/ai-tokenizer"
)

// toASCII is a test-only helper that mirrors ASCII folding using the production
// Normalize.
// It returns 0 when the rune cannot be represented after normalization.
func toASCII(r rune) rune {
	tok := tokenizer.NewTokenizer()

	out := tok.Normalize(string(r))
	if out == "" {
		return 0
	}

	rs := []rune(out)

	return rs[0]
}

// Test constants to eliminate duplication and magic numbers.
const (
	// Error message formats.
	EstimateTokensErrorFormat = "EstimateTokens(%q) = %d, want %d"
	NormalizeErrorFormat      = "Normalize(%q) = %q, want %q"
	ToASCIIErrorFormat        = "toASCII(%c) = %c, want %c"
	GetModelErrorFormat       = "GetModel() = %q, want %q"

	// Common test strings.
	EmptyString     = ""
	SingleChar      = "a"
	TwoChars        = "ab"
	ThreeChars      = "abc"
	FourChars       = "abcd"
	HelloWorld      = "hello world"
	CafeUnicode     = "café"
	NaiveUnicode    = "naïve"
	MullerUmlaut    = "Müller"
	HanScriptText   = "Hello\u4e16\u754c"                                    // Han characters for testing (世界)
	MixedScriptText = "café\u4e16\u754c\u043f\u0440\u0438\u0432\u0435\u0442" // Mixed scripts for testing (世界привет)

	// Benchmark strings.
	BenchmarkEstimateText = "This is a benchmark test for the tokenizer estimation " +
		"functionality. It should measure the performance of token counting " +
		"with special characters!"
	BenchmarkNormalizeText = "This is a tëst with spéciål chäracters and unicode " +
		"tëxt for nørmalization benchmarking."

	// Error messages.
	GetModelEmptyError   = "GetModel() returned empty string"
	NewTokenizerNilError = "tokenizer.NewTokenizer() returned nil"
	EmptyEstimateError   = "EstimateTokens(\"\") = %d, want 0"
	EmptyNormalizeError  = "Normalize(\"\") = %q, want \"\""
	NegativeTokensError  = "EstimateTokens returned negative value: %d for input %q"
	NonASCIIResultError  = "Normalize returned non-ASCII character %c in %q from input %q"
)

type TokenEstimateTestCase struct {
	name     string
	input    string
	expected int
}

type NormalizeTestCase struct {
	name     string
	input    string
	expected string
}

type ToASCIITestCase struct {
	name     string
	input    rune
	expected rune
}

func getTokenEstimateTestCases() []TokenEstimateTestCase {
	return []TokenEstimateTestCase{
		{"empty string", EmptyString, 0},
		{"single character", SingleChar, 1},
		{"two characters", TwoChars, 1},
		{"three characters", ThreeChars, 2},
		{"four characters", FourChars, 2},
		{"special character alone", "!", 1},
		{"multiple special characters", "!@#", 3},
		{"mixed text and special chars", "hello!world", 7},
		{"text with punctuation", "Hello, world!", 9},
		{"non-ASCII characters (unicode)", CafeUnicode, 2},
		{"mixed unicode and special chars", "naïve!", 4},
		{"whitespace", "   ", 3}, // three spaces -> three special tokens
		{"newlines and tabs", "\n\t", 2},
	}
}

func TestTokenizerEstimate(t *testing.T) {
	t.Parallel()

	tests := getTokenEstimateTestCases()
	tok := tokenizer.NewTokenizer()

	runTokenEstimateTests(t, tok, tests)
}

func runTokenEstimateTests(
	t *testing.T,
	tok *tokenizer.Tokenizer,
	tests []TokenEstimateTestCase,
) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			validateTokenEstimate(t, tok, testCase)
		})
	}
}

func validateTokenEstimate(
	t *testing.T,
	tok *tokenizer.Tokenizer,
	testCase TokenEstimateTestCase,
) {
	t.Helper()

	result := tok.EstimateTokens(testCase.input)
	if result != testCase.expected {
		t.Errorf(
			EstimateTokensErrorFormat,
			testCase.input,
			result,
			testCase.expected,
		)
	}
}

func getNormalizeTestCases() []NormalizeTestCase {
	return []NormalizeTestCase{
		{"ASCII text unchanged", HelloWorld, HelloWorld},
		{"accented characters", CafeUnicode, "cafe"},
		{"various diacritics", "naïve résumé", "naive resume"},
		{"German umlauts", MullerUmlaut, "Muller"},
		{"mixed ASCII and Unicode", "Hello café!", "Hello cafe!"},
		{"empty string", EmptyString, EmptyString},
	}
}

func TestTokenizerNormalize(t *testing.T) {
	t.Parallel()

	tests := getNormalizeTestCases()
	tok := tokenizer.NewTokenizer()

	runNormalizeTests(t, tok, tests)
}

func runNormalizeTests(
	t *testing.T,
	tok *tokenizer.Tokenizer,
	tests []NormalizeTestCase,
) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			validateNormalize(t, tok, testCase)
		})
	}
}

func validateNormalize(
	t *testing.T,
	tok *tokenizer.Tokenizer,
	testCase NormalizeTestCase,
) {
	t.Helper()

	result := tok.Normalize(testCase.input)
	if result != testCase.expected {
		t.Errorf(
			NormalizeErrorFormat,
			testCase.input,
			result,
			testCase.expected,
		)
	}
}

func TestTokenizerGetModel(t *testing.T) {
	t.Parallel()

	tok := tokenizer.NewTokenizer()

	model := tok.GetModel()
	if model == EmptyString {
		t.Error(GetModelEmptyError)
	}

	if model != tokenizer.DefaultModel {
		t.Errorf(GetModelErrorFormat, model, tokenizer.DefaultModel)
	}
}

func BenchmarkTokenizerEstimate(b *testing.B) {
	tok := tokenizer.NewTokenizer()
	text := BenchmarkEstimateText

	b.ResetTimer()

	for range b.N {
		_ = tok.EstimateTokens(text)
	}
}

func BenchmarkTokenizerNormalize(b *testing.B) {
	tok := tokenizer.NewTokenizer()
	text := BenchmarkNormalizeText

	b.ResetTimer()

	for range b.N {
		_ = tok.Normalize(text)
	}
}

func getEdgeCaseEstimateTestCases() []TokenEstimateTestCase {
	return []TokenEstimateTestCase{
		{"very long text", strings.Repeat(TwoChars, 1000), 1000},
		{"only special characters", "!@#$%^&*()", 10},
		{
			"mixed languages",
			HanScriptText,
			3,
		}, // Hello -> 3 tokens (5 letters -> ceil(5/2)=3), 世界 dropped by Normalize
		{"emoji and symbols", "😀👍🎉", 0}, // dropped by Normalize
		{"numbers and letters", "abc123def", 5},
		{"tabs and newlines mixed", "a\tb\nc\rd", 7},
		{"repeated spaces", " hello world ", 9}, // spaces are special tokens
		{"mixed case with accents", "CAFÉ café", 5},
	}
}

func TestTokenizerEstimateEdgeCases(t *testing.T) {
	t.Parallel()

	tests := getEdgeCaseEstimateTestCases()
	tok := tokenizer.NewTokenizer()

	runTokenEstimateTests(t, tok, tests)
}

func getEdgeCaseNormalizeTestCases() []NormalizeTestCase {
	return []NormalizeTestCase{
		{"complex diacritics", "àáâãäåçèéêë", "aaaaaaceeee"},
		{"uppercase with diacritics", "ÀÁÂÃÄÅÇÈÉÊË", "AAAAAACEEEE"},
		{"mixed scripts", MixedScriptText, "cafe"},
		{"special ligatures", "ﬁﬂ", EmptyString},
		{"already ASCII", "abcDEF123!@#", "abcDEF123!@#"},
		{"combining characters", "a\u0301b\u0302c\u0308", "abc"},
	}
}

func TestTokenizerNormalizeEdgeCases(t *testing.T) {
	t.Parallel()

	tests := getEdgeCaseNormalizeTestCases()
	tok := tokenizer.NewTokenizer()

	runNormalizeTests(t, tok, tests)
}

func getToASCIITestCases() []ToASCIITestCase {
	return []ToASCIITestCase{
		{"ASCII a", 'a', 'a'},
		{"ASCII Z", 'Z', 'Z'},
		{"digit 5", '5', '5'},
		{"accented a", 'à', 'a'},
		{"accented A", 'À', 'A'},
		{"non-convertible", '世', 0},
		{"non-ASCII symbol", '€', 0},
	}
}

func TestToASCII(t *testing.T) {
	t.Parallel()

	tests := getToASCIITestCases()
	runToASCIITests(t, tests)
}

func runToASCIITests(t *testing.T, tests []ToASCIITestCase) {
	t.Helper()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			validateToASCII(t, testCase)
		})
	}
}

func validateToASCII(t *testing.T, testCase ToASCIITestCase) {
	t.Helper()

	result := toASCII(testCase.input)
	if result != testCase.expected {
		t.Errorf(
			ToASCIIErrorFormat,
			testCase.input,
			result,
			testCase.expected,
		)
	}
}

func testTokenizerCreation(t *testing.T) *tokenizer.Tokenizer {
	t.Helper()

	tok := tokenizer.NewTokenizer()
	if tok == nil {
		t.Error(NewTokenizerNilError)
	}

	return tok
}

func testEmptyStringEstimate(t *testing.T, tok *tokenizer.Tokenizer) {
	t.Helper()

	result := tok.EstimateTokens(EmptyString)
	if result != 0 {
		t.Errorf(EmptyEstimateError, result)
	}
}

func testEmptyStringNormalize(t *testing.T, tok *tokenizer.Tokenizer) {
	t.Helper()

	normalized := tok.Normalize(EmptyString)
	if normalized != EmptyString {
		t.Errorf(EmptyNormalizeError, normalized)
	}
}

func TestTokenizerNilSafety(t *testing.T) {
	t.Parallel()

	tok := testTokenizerCreation(t)
	testEmptyStringEstimate(t, tok)
	testEmptyStringNormalize(t, tok)
}
