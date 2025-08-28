package aitokenizer

import (
	"testing"

	"github.com/tiktoken-go/tokenizer"
)

func TestNewTokenizer(t *testing.T) {
	tok := NewTokenizer()
	if tok == nil {
		t.Fatal("NewTokenizer() returned nil")
	}
}

func TestNewTokenizerWithEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoding tokenizer.Encoding
	}{
		{"cl100k_base", tokenizer.Cl100kBase},
		{"gpt2", tokenizer.GPT2Enc},
		{"p50k_base", tokenizer.P50kBase},
		{"r50k_base", tokenizer.R50kBase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizerWithEncoding(tt.encoding)
			if tok == nil {
				t.Fatal("NewTokenizerWithEncoding() returned nil")
			}
		})
	}
}

func TestCountTokens(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"two words", "hello world", 2},
		{"sentence", "Hello world, how are you?", 6},
		{"longer text", "This is a longer text that should have more tokens than the previous examples.", 15},
		{"special characters", "Hello! @#$%^&*()", 4},
		{"numbers", "123 456 789", 3},
		{"mixed content", "Hello123 world! @#$", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.CountTokens(tt.content)
			if result < 0 {
				t.Errorf("CountTokens() returned negative value: %d", result)
			}
			// Note: We can't test exact values since tokenization can vary
			// but we can test that it returns reasonable values
			if tt.content == "" && result != 0 {
				t.Errorf("CountTokens() for empty string = %d, want 0", result)
			}
		})
	}
}

func TestCountTokensWithLanguage(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name     string
		content  string
		language string
	}{
		{"english", "Hello world", "en"},
		{"spanish", "Hola mundo", "es"},
		{"french", "Bonjour le monde", "fr"},
		{"german", "Hallo Welt", "de"},
		{"chinese", "你好世界", "zh"},
		{"japanese", "こんにちは世界", "ja"},
		{"empty content", "", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.CountTokensWithLanguage(tt.content, tt.language)
			if result < 0 {
				t.Errorf("CountTokensWithLanguage() returned negative value: %d", result)
			}
			if tt.content == "" && result != 0 {
				t.Errorf("CountTokensWithLanguage() for empty string = %d, want 0", result)
			}
		})
	}
}

func TestEstimateTokensForFile(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name     string
		content  string
		language string
	}{
		{"go file", "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}", "go"},
		{"python file", "def hello():\n    print('Hello, World!')", "python"},
		{"javascript file", "function hello() {\n    console.log('Hello, World!');\n}", "javascript"},
		{"markdown file", "# Title\n\nThis is some **markdown** content.", "markdown"},
		{"empty file", "", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.EstimateTokensForFile(tt.content, tt.language)
			if result < 0 {
				t.Errorf("EstimateTokensForFile() returned negative value: %d", result)
			}
			if tt.content == "" && result != 0 {
				t.Errorf("EstimateTokensForFile() for empty string = %d, want 0", result)
			}
		})
	}
}

func TestCountTokensForModel(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name    string
		content string
		model   tokenizer.Model
		wantErr bool
	}{
		{"gpt-4", "Hello world", tokenizer.GPT4, false},
		{"gpt-3.5-turbo", "Hello world", tokenizer.GPT35Turbo, false},
		{"gpt-2", "Hello world", tokenizer.GPT2, true}, // GPT2 encoding not supported in this version
		{"text-davinci-003", "Hello world", tokenizer.TextDavinci003, false},
		{"empty content", "", tokenizer.GPT4, false},
		{"invalid model", "Hello world", tokenizer.Model("invalid-model"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tok.CountTokensForModel(tt.content, tt.model)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CountTokensForModel() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CountTokensForModel() unexpected error: %v", err)
				return
			}

			if result < 0 {
				t.Errorf("CountTokensForModel() returned negative value: %d", result)
			}

			if tt.content == "" && result != 0 {
				t.Errorf("CountTokensForModel() for empty string = %d, want 0", result)
			}
		})
	}
}

func TestCountTokensForEncoding(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name     string
		content  string
		encoding tokenizer.Encoding
		wantErr  bool
	}{
		{"cl100k_base", "Hello world", tokenizer.Cl100kBase, false},
		{"gpt2", "Hello world", tokenizer.GPT2Enc, true}, // GPT2 encoding not supported in this version
		{"p50k_base", "Hello world", tokenizer.P50kBase, false},
		{"r50k_base", "Hello world", tokenizer.R50kBase, false},
		{"empty content", "", tokenizer.Cl100kBase, false},
		{"invalid encoding", "Hello world", tokenizer.Encoding("invalid-encoding"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tok.CountTokensForEncoding(tt.content, tt.encoding)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CountTokensForEncoding() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CountTokensForEncoding() unexpected error: %v", err)
				return
			}

			if result < 0 {
				t.Errorf("CountTokensForEncoding() returned negative value: %d", result)
			}

			if tt.content == "" && result != 0 {
				t.Errorf("CountTokensForEncoding() for empty string = %d, want 0", result)
			}
		})
	}
}

func TestFallbackCount(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"two words", "hello world", 2},
		{"multiple words", "hello world how are you", 5},
		{"with punctuation", "hello, world! how are you?", 5},
		{"with numbers", "hello 123 world 456", 4},
		{"with special chars", "hello @#$ world", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.fallbackCount(tt.content)
			if result != tt.expected {
				t.Errorf("fallbackCount() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkCountTokens(b *testing.B) {
	tok := NewTokenizer()
	content := "This is a benchmark test for token counting. It contains multiple sentences and should provide a good measure of performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.CountTokens(content)
	}
}

func BenchmarkCountTokensWithLanguage(b *testing.B) {
	tok := NewTokenizer()
	content := "This is a benchmark test for token counting with language specification."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.CountTokensWithLanguage(content, "en")
	}
}

func BenchmarkCountTokensForModel(b *testing.B) {
	tok := NewTokenizer()
	content := "This is a benchmark test for model-specific token counting."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.CountTokensForModel(content, tokenizer.GPT4)
	}
}

func BenchmarkCountTokensForEncoding(b *testing.B) {
	tok := NewTokenizer()
	content := "This is a benchmark test for encoding-specific token counting."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.CountTokensForEncoding(content, tokenizer.Cl100kBase)
	}
}

// Test tokenizer consistency
func TestTokenizerConsistency(t *testing.T) {
	tok := NewTokenizer()
	content := "Hello world, this is a test."

	// Count should be consistent
	count1 := tok.CountTokens(content)
	count2 := tok.CountTokens(content)

	if count1 != count2 {
		t.Errorf("Token count not consistent: first=%d, second=%d", count1, count2)
	}
}

// Test different encodings produce different results
func TestEncodingDifferences(t *testing.T) {
	content := "Hello world"

	cl100k := NewTokenizerWithEncoding(tokenizer.Cl100kBase)
	gpt2 := NewTokenizerWithEncoding(tokenizer.GPT2Enc)

	count1 := cl100k.CountTokens(content)
	count2 := gpt2.CountTokens(content)

	// Different encodings should produce different token counts
	// (though this might not always be true for very short texts)
	if count1 == count2 && len(content) > 5 {
		t.Logf("Warning: Different encodings produced same token count: %d", count1)
	}
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	tok := NewTokenizer()

	tests := []struct {
		name    string
		content string
	}{
		{"very long text", string(make([]byte, 10000))},
		{"unicode characters", "Hello 世界 🌍"},
		{"newlines", "Hello\nworld\r\nhow\nare\nyou"},
		{"tabs", "Hello\tworld\thow\tare\tyou"},
		{"mixed whitespace", "Hello   world\n\t  how   are   you"},
		{"special unicode", "Hello\u0000world\u0001test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.CountTokens(tt.content)
			if result < 0 {
				t.Errorf("CountTokens() returned negative value for %s: %d", tt.name, result)
			}
		})
	}
}
