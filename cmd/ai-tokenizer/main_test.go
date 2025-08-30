package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// Common format strings and messages.
	fmtExpErr          = "tokenizeText(%q) expected error, got nil"
	fmtUnexpErr        = "tokenizeText(%q) unexpected error: %v"
	fmtNilResult       = "tokenizeText(%q) returned nil result"
	fmtTextMismatch    = "tokenizeText(%q) result.Text = %q, want %q"
	fmtNegCount        = "tokenizeText(%q) returned negative token count: %d"
	fmtEmptyModel      = "tokenizeText(%q) returned empty model"
	fmtOrigMismatch    = "tokenizeText(%q) result.OriginalText = %q, want %q"
	fmtEmptyNorm       = "tokenizeText(%q) returned empty normalized text"
	fmtOrigShouldEmpty = "tokenizeText(%q) result.OriginalText should be empty when showNormalized=false"
	fmtNormShouldEmpty = "tokenizeText(%q) result.NormalizedText should be empty when showNormalized=false"
	fmtTruncateText    = "truncateText(%q, %d) = %q, want %q"

	// File and IO constants.
	sampleFileContent = "This is a test file content.\nWith multiple lines!"
	invalidPath       = "/nonexistent/file.txt"
	testFileName      = "test_tokenizer_input.txt"

	// Skip messages.
	skipReadStdin = "readStdin() requires stdin mocking"

	// Read-file messages.
	fmtReadFileErr  = "readFile() error: %v"
	fmtReadFileWant = "readFile() = %q, want %q"
	fmtShouldErrNE  = "readFile() should return error for non-existent file"

	// JSON messages.
	fmtTokErr           = "tokenizeText() error: %v"
	fmtJSONUnmarshalErr = "JSON unmarshaling failed: %v"
	fmtRoundtripText    = "JSON roundtrip: Text = %q, want %q"
	fmtRoundtripTokens  = "JSON roundtrip: TokenCount = %d, want %d"
	fmtRoundtripModel   = "JSON roundtrip: Model = %q, want %q"
	fmtMissingField     = "JSON output missing field: %s\nFull JSON: %s"
	fmtJSONMarshalErr   = "json.Marshal error: %v"

	// Version logs.
	logEmptyVersion   = "Version is empty (expected in test environment)"
	logEmptyBuildTime = "BuildTime is empty (expected in test environment)"

	// Sample strings for tests - consolidated duplicates.
	hello      = "hello"
	helloWorld = "Hello, world!"
	simpleText = "simple"
	testValue  = "test" // unified constant

	sampleOrig = "tëst"

	// Benchmarks.
	benchInput = "This is a benchmark test for the tokenization function with some unicode characters like café and naïve."
	benchOrig  = "benchmark tëst"

	// JSON field expectations.
	jsonTextField       = `"text":"test"`
	jsonTokenCountField = `"tokenCount":5`
	jsonModelField      = `"model":"simple"`
	jsonOriginalText    = `"originalText":"tëst"`
	jsonNormalizedText  = `"normalizedText":"test"`

	// Test names.
	edgeCasePrefix = "edge_case"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// Simple validation functions with complexity <= 3.
func validateText(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.Text != input {
		t.Errorf(fmtTextMismatch, input, result.Text, input)
	}
}

func validateTokenCount(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.TokenCount < 0 {
		t.Errorf(fmtNegCount, input, result.TokenCount)
	}
}

func validateModel(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.Model == "" {
		t.Errorf(fmtEmptyModel, input)
	}
}

func validateOriginalText(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.OriginalText != input {
		t.Errorf(fmtOrigMismatch, input, result.OriginalText, input)
	}
}

func validateNormalizedNotEmpty(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if input != "" && result.NormalizedText == "" {
		t.Errorf(fmtEmptyNorm, input)
	}
}

func validateOriginalEmpty(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.OriginalText != "" {
		t.Errorf(fmtOrigShouldEmpty, input)
	}
}

func validateNormalizedEmpty(t *testing.T, input string, result *TokenResult) {
	t.Helper()

	if result.NormalizedText != "" {
		t.Errorf(fmtNormShouldEmpty, input)
	}
}

func assertTokenCommon(
	t *testing.T,
	input string,
	showNormalized bool,
	result *TokenResult,
) {
	t.Helper()

	if result == nil {
		t.Errorf(fmtNilResult, input)

		return
	}

	validateText(t, input, result)
	validateTokenCount(t, input, result)
	validateModel(t, input, result)

	if showNormalized {
		validateOriginalText(t, input, result)
		validateNormalizedNotEmpty(t, input, result)
	} else {
		validateOriginalEmpty(t, input, result)
		validateNormalizedEmpty(t, input, result)
	}
}

type tokenTestCase struct {
	name           string
	input          string
	showNormalized bool
	expectError    bool
}

func handleExpectedError(t *testing.T, input string, err error) bool {
	t.Helper()

	if err == nil {
		t.Errorf(fmtExpErr, input)
	}

	return true
}

func handleUnexpectedError(t *testing.T, input string, err error) bool {
	t.Helper()

	if err != nil {
		t.Errorf(fmtUnexpErr, input, err)

		return true
	}

	return false
}

func runTokenizeTest(t *testing.T, testCase tokenTestCase) {
	t.Helper()
	t.Parallel()

	result, err := tokenizeText(testCase.input, testCase.showNormalized)

	if testCase.expectError {
		handleExpectedError(t, testCase.input, err)

		return
	}

	if handleUnexpectedError(t, testCase.input, err) {
		return
	}

	assertTokenCommon(t, testCase.input, testCase.showNormalized, result)
}

func TestTokenizeText(t *testing.T) {
	t.Parallel()

	tests := []tokenTestCase{
		{
			name:           "simple text",
			input:          helloWorld,
			showNormalized: false,
			expectError:    false,
		},
		{
			name:           "empty text",
			input:          "",
			showNormalized: false,
			expectError:    false,
		},
		{
			name:           "unicode text with normalization",
			input:          "café",
			showNormalized: true,
			expectError:    false,
		},
		{
			name:           "special characters",
			input:          "!@#$%",
			showNormalized: false,
			expectError:    false,
		},
		{
			name:           "mixed content",
			input:          "Hello! How are you? 123",
			showNormalized: true,
			expectError:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runTokenizeTest(t, testCase)
		})
	}
}

func createTestFile(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, testFileName)

	err := os.WriteFile(tmpFile, []byte(sampleFileContent), 0o600)
	if err != nil {
		t.Fatalf(fmtReadFileErr, err)
	}

	return tmpFile
}

func validateFileContent(t *testing.T, content string) {
	t.Helper()

	if content != sampleFileContent {
		t.Errorf(fmtReadFileWant, content, sampleFileContent)
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	tmpFile := createTestFile(t)

	content, err := readFile(tmpFile)
	if err != nil {
		t.Errorf(fmtReadFileErr, err)

		return
	}

	validateFileContent(t, content)

	_, err = readFile(invalidPath)
	if err == nil {
		t.Error(fmtShouldErrNE)
	}
}

func TestReadStdin(t *testing.T) {
	t.Parallel()
	t.Skip(skipReadStdin)
}

type truncateTestCase struct {
	name   string
	input  string
	want   string
	maxLen int
}

func runTruncateTest(t *testing.T, testCase truncateTestCase) {
	t.Helper()
	t.Parallel()

	result := truncateText(testCase.input, testCase.maxLen)
	if result != testCase.want {
		t.Errorf(
			fmtTruncateText,
			testCase.input,
			testCase.maxLen,
			result,
			testCase.want,
		)
	}
}

func TestTruncateText(t *testing.T) {
	t.Parallel()

	tests := []truncateTestCase{
		{name: "short text", input: hello, maxLen: 10, want: hello},
		{name: "exact length", input: hello, maxLen: 5, want: hello},
		{
			name:   "truncate needed",
			input:  "hello world this is a long text",
			maxLen: 10,
			want:   "hello w...",
		},
		{name: "very short maxLen", input: hello, maxLen: 3, want: "..."},
		{name: "empty input", input: "", maxLen: 10, want: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runTruncateTest(t, testCase)
		})
	}
}

func getTestTokenResult(t *testing.T) *TokenResult {
	t.Helper()

	result, err := tokenizeText(helloWorld, false)
	if err != nil {
		t.Fatalf(fmtTokErr, err)
	}

	return result
}

func marshalResult(t *testing.T, result *TokenResult) []byte {
	t.Helper()

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf(fmtJSONMarshalErr, err)
	}

	return jsonData
}

func unmarshalResult(t *testing.T, jsonData []byte) *TokenResult {
	t.Helper()

	var unmarshaled TokenResult

	err := json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Errorf(fmtJSONUnmarshalErr, err)
	}

	return &unmarshaled
}

func validateTextField(t *testing.T, original, unmarshaled *TokenResult) {
	t.Helper()

	if unmarshaled.Text != original.Text {
		t.Errorf(fmtRoundtripText, unmarshaled.Text, original.Text)
	}
}

func validateTokenCountField(t *testing.T, original, unmarshaled *TokenResult) {
	t.Helper()

	if unmarshaled.TokenCount != original.TokenCount {
		t.Errorf(fmtRoundtripTokens, unmarshaled.TokenCount, original.TokenCount)
	}
}

func validateModelField(t *testing.T, original, unmarshaled *TokenResult) {
	t.Helper()

	if unmarshaled.Model != original.Model {
		t.Errorf(fmtRoundtripModel, unmarshaled.Model, original.Model)
	}
}

func TestJSONOutput(t *testing.T) {
	t.Parallel()

	result := getTestTokenResult(t)
	jsonData := marshalResult(t, result)
	unmarshaled := unmarshalResult(t, jsonData)

	validateTextField(t, result, unmarshaled)
	validateTokenCountField(t, result, unmarshaled)
	validateModelField(t, result, unmarshaled)
}

func createSampleTokenResult() *TokenResult {
	return &TokenResult{
		Text:           testValue,
		TokenCount:     5,
		Model:          simpleText,
		OriginalText:   sampleOrig,
		NormalizedText: testValue,
	}
}

func validateJSONField(t *testing.T, jsonStr, field string) {
	t.Helper()

	if !strings.Contains(jsonStr, field) {
		t.Errorf(fmtMissingField, field, jsonStr)
	}
}

func TestTokenResultStructure(t *testing.T) {
	t.Parallel()

	result := createSampleTokenResult()
	jsonData := marshalResult(t, result)
	jsonStr := string(jsonData)

	expectedFields := []string{
		jsonTextField,
		jsonTokenCountField,
		jsonModelField,
		jsonOriginalText,
		jsonNormalizedText,
	}

	for _, field := range expectedFields {
		validateJSONField(t, jsonStr, field)
	}
}

func TestVersionInfo(t *testing.T) {
	t.Parallel()

	if DefaultVersion == "" {
		t.Log(logEmptyVersion)
	}

	if DefaultBuildTime == "" {
		t.Log(logEmptyBuildTime)
	}

	_ = DefaultVersion
	_ = DefaultBuildTime
}

func BenchmarkTokenizeText(b *testing.B) {
	input := benchInput

	b.ResetTimer()

	for range b.N {
		_, err := tokenizeText(input, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenizeTextWithNormalization(b *testing.B) {
	input := benchInput

	b.ResetTimer()

	for range b.N {
		_, err := tokenizeText(input, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONMarshaling(b *testing.B) {
	result := &TokenResult{
		Text:           testValue,
		TokenCount:     10,
		Model:          simpleText,
		OriginalText:   benchOrig,
		NormalizedText: testValue,
	}

	b.ResetTimer()

	for range b.N {
		_, err := json.Marshal(result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createEdgeCaseName(input string) string {
	name := edgeCasePrefix
	if input != "" {
		name += "_" + input[:minInt(len(input), 10)]
	}

	return name
}

func runEdgeCaseTest(t *testing.T, input string) {
	t.Helper()
	t.Parallel()

	result, err := tokenizeText(input, true)
	if err != nil {
		t.Errorf(fmtUnexpErr, input, err)

		return
	}

	assertTokenCommon(t, input, true, result)
}

func TestErrorConditions(t *testing.T) {
	t.Parallel()

	edgeCases := []string{
		"",                         // empty
		" ",                        // single space
		"\n\t\r",                   // only whitespace
		"🌟🎉😀",                      // only emojis
		"café naïve résumé",        // unicode
		strings.Repeat("a", 10000), // very long string
	}

	for _, input := range edgeCases {
		name := createEdgeCaseName(input)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runEdgeCaseTest(t, input)
		})
	}
}
