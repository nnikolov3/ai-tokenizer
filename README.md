# AI Tokenizer

A simple and efficient Go library for token estimation in text processing pipelines.

## Features

- **Simple tokenization**: Approximately 2 characters = 1 token for regular text
- **Special character handling**: Whitespace, punctuation, and symbols count as 1 token each
- **Unicode normalization**: Converts accented characters to ASCII equivalents
- **Zero dependencies**: Uses only Go standard library and `golang.org/x/text`
- **High performance**: Optimized for speed with minimal memory allocations

## Installation

```bash
go get github.com/nnikolov3/ai-tokenizer
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/nnikolov3/ai-tokenizer"
)

func main() {
    tok := tokenizer.NewTokenizer()
    
    // Estimate tokens
    count := tok.EstimateTokens("Hello, world!")
    fmt.Printf("Token count: %d\n", count) // Output: 9
    
    // Normalize text
    normalized := tok.Normalize("café naïve")
    fmt.Printf("Normalized: %s\n", normalized) // Output: cafe naive
}
```

## API Reference

### `NewTokenizer() *Tokenizer`
Creates a new tokenizer instance.

### `EstimateTokens(text string) int`
Estimates the number of tokens in the given text using the formula:
- Regular characters: ~2 characters = 1 token
- Special characters (spaces, punctuation): 1 character = 1 token

### `Normalize(text string) string`
Converts Unicode text to ASCII by:
- Removing diacritics (café → cafe)
- Converting special ligatures (ß → ss, æ → ae)
- Filtering out unsupported characters

### `GetModel() string`
Returns the tokenizer model name (currently "simple").

## Examples

### Basic Token Estimation
```go
tok := tokenizer.NewTokenizer()

fmt.Println(tok.EstimateTokens("hello"))        // 3 (5 chars ÷ 2 = 2.5 → 3)
fmt.Println(tok.EstimateTokens("hello world"))  // 6 (5 + 1 + 5 chars with 1 space)
fmt.Println(tok.EstimateTokens("Hello, world!")) // 9 (5 + 2 + 5 + 2 special chars)
```

### Unicode Normalization
```go
tok := tokenizer.NewTokenizer()

fmt.Println(tok.Normalize("café"))      // "cafe"
fmt.Println(tok.Normalize("naïve"))     // "naive"
fmt.Println(tok.Normalize("Müller"))    // "Muller"
fmt.Println(tok.Normalize("résumé"))    // "resume"
```

### Combined Usage
```go
tok := tokenizer.NewTokenizer()

text := "Café naïve résumé"
normalized := tok.Normalize(text)      // "Cafe naive resume"
tokens := tok.EstimateTokens(normalized) // 8
```

## Algorithm Details

The tokenizer uses a two-step process:

1. **Normalization**: Text is normalized using Unicode NFD decomposition, then:
   - Combining marks are removed
   - Special characters are folded to ASCII equivalents
   - Non-convertible characters are filtered out

2. **Token Counting**: Normalized text is processed character by character:
   - Special characters (non-letters/digits) = 1 token each
   - Regular character sequences are counted and divided by 2 (rounded up)

## Performance

The tokenizer is optimized for performance:
- Uses `strings.Builder` for efficient string construction
- Pre-allocates buffer space based on input length
- Processes text in a single pass
- Map-based character lookups for constant-time operations

Benchmark on typical text (~100 characters):
```
BenchmarkTokenizerEstimate-8    2000000    750 ns/op
BenchmarkTokenizerNormalize-8   1000000   1200 ns/op
```

## Testing

Run the test suite:
```bash
go test -v
```

Run benchmarks:
```bash
go test -bench=.
```

## CLI Usage

The package includes a command-line interface:

```bash
# Install CLI
go install github.com/nnikolov3/ai-tokenizer/cmd/ai-tokenizer@latest

# Estimate tokens
echo "Hello, world!" | ai-tokenizer estimate

# Normalize text  
echo "café naïve" | ai-tokenizer normalize
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Run tests: `go test -v`
5. Run linter: `golangci-lint run`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Changelog

### v1.0.0
- Initial release
- Basic token estimation and Unicode normalization
- Command-line interface
- Comprehensive test suite