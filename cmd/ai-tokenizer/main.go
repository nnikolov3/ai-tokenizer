// Package main provides a CLI for estimating AI token counts with optional
// normalization and JSON output.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	tokenizer "github.com/nnikolov3/ai-tokenizer"
)

// TokenResult is the output payload for tokenization results.
type TokenResult struct {
	Text           string `json:"text"`
	Model          string `json:"model"`
	OriginalText   string `json:"originalText,omitempty"`
	NormalizedText string `json:"normalizedText,omitempty"`
	TokenCount     int    `json:"tokenCount"`
}

const (
	// Defaults for build metadata.
	DefaultVersion   = "dev"
	DefaultBuildTime = "unknown"

	// Output/format strings.
	MsgVersionFmt    = "AI Tokenizer %s (built %s)\n"
	MsgTextFmt       = "Text: %s\n"
	MsgTokenCountFmt = "Token Count: %d\n"
	MsgModelFmt      = "Model: %s\n"
	MsgNormalizedFmt = "Normalized: %s\n"
	MsgJSONIndent    = "  "
	FmtGenericErr    = "%v"

	// Error wrappers/messages.
	ErrWrapTokenize   = "tokenize: %w"
	ErrWrapEncodeJSON = "encode json: %w"
	ErrWrapReadStdin  = "read stdin: %w"
	ErrOpenFileFmt    = "failed to open file %q: %w"
	ErrReadFileFmt    = "failed to read file %q: %w"
	ErrNoInputMsg     = "no input"

	// Flag names and help strings.
	FlagNameVersion    = "version"
	FlagNameJSON       = "json"
	FlagNameFile       = "file"
	FlagNameText       = "text"
	FlagNameNormalized = "normalized"

	FlagHelpVersion    = "Show version information"
	FlagHelpJSON       = "Output in JSON format"
	FlagHelpInputFile  = "Input file path (default: stdin)"
	FlagHelpText       = "Text to tokenize"
	FlagHelpNormalized = "Show normalized text in output"

	// Usage text (lines wrapped to meet 80-char limit).
	UsageHeader = "" +
		"AI Tokenizer - Simple token estimation tool\n\n"
	UsageUsageFmt = "" +
		"Usage: %s [options] [text]\n\n"
	UsageRules = "" +
		"Tokenization Rules:\n" +
		"  - 2 regular characters = 1 token\n" +
		"  - 1 special character = 1 token\n" +
		"  - Non-ASCII chars converted to ASCII equivalents\n\n"
	UsageOptions     = "Options:\n"
	UsageExamplesFmt = "" +
		"\nExamples:\n" +
		"  %s \"Hello, world!\"\n" +
		"  %s -json \"Hello, world!\"\n" +
		"  %s -file input.txt\n" +
		"  echo \"Hello, world!\" | %s\n" +
		"  %s -text \"café\" -normalized\n"

	// CLI preview defaults and constants for helpers.
	DefaultPreviewMax = 100
	ExecutableDefault = "ai-tokenizer"
	EllipsisLen       = 3

	// Initial capacity guess for build settings map.
	SettingsInitCap = 8
)

type VersionInfo struct {
	Revision       string
	BuildTimestamp string
}

// ErrNoInput is returned when no input text is provided.
var ErrNoInput = errors.New(ErrNoInputMsg)

// cliFlags collects parsed CLI flags for the CLI program.
type cliFlags struct {
	inputFile      string
	text           string
	showVersion    bool
	outputJSON     bool
	showNormalized bool
}

func main() {
	err := run()
	if err != nil {
		printError(FmtGenericErr+"\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := parseFlags()
	// Handle --version early to keep branching
	if flags.showVersion {
		printVersion()

		return nil
	}
	//
	input, err := requireInput(flags)
	if err != nil {
		return err
	}

	return process(flags, input)
}

func requireInput(flags *cliFlags) (string, error) {
	textInput, err := obtainInput(flags)
	if err != nil {
		return "", err
	}

	err = ensureNonEmpty(textInput)
	if err != nil {
		printError(FmtGenericErr+"\n", err)
		printUsage()

		return "", err
	}

	return textInput, nil
}

func resolveVersionAndTime() VersionInfo {
	info := readBuildInfo()
	settings := buildSettingsMap(info)

	revision := settings["vcs.revision"]
	if revision == "" {
		revision = DefaultVersion
	}

	buildTimestamp := settings["vcs.time"]
	if buildTimestamp == "" {
		buildTimestamp = DefaultBuildTime
	}

	return VersionInfo{Revision: revision, BuildTimestamp: buildTimestamp}
}

// obtainInput resolves the input text using flags, args, or stdin.
func obtainInput(flags *cliFlags) (string, error) {
	return readInputNonText(flags)
}

func ensureNonEmpty(inputStr string) error {
	if strings.TrimSpace(inputStr) == "" {
		return ErrNoInput
	}

	return nil
}

// New helper: the pipeline after we have validated input.
func process(flags *cliFlags, input string) error {
	result, err := buildResult(flags, input)
	if err != nil {
		return fmt.Errorf(ErrWrapTokenize, err)
	}

	return emitResult(flags, result)
}

// parseFlags defines and parses CLI flags, returning a structured result.
func parseFlags() *cliFlags {
	showVersion := flag.Bool(FlagNameVersion, false, FlagHelpVersion)
	outputJSON := flag.Bool(FlagNameJSON, false, FlagHelpJSON)
	inputFile := flag.String(FlagNameFile, "", FlagHelpInputFile)
	text := flag.String(FlagNameText, "", FlagHelpText)
	showNormalized := flag.Bool(FlagNameNormalized, false, FlagHelpNormalized)

	flag.Usage = func() { printUsage() }
	flag.Parse()

	return &cliFlags{
		showVersion:    *showVersion,
		outputJSON:     *outputJSON,
		inputFile:      *inputFile,
		text:           *text,
		showNormalized: *showNormalized,
	}
}

// buildResult selects tokenization mode based on flags and returns a result.
func buildResult(flags *cliFlags, input string) (*TokenResult, error) {
	if flags.showNormalized {
		return tokenizeNormalized(input)
	}

	return tokenize(input)
}

// emitResult chooses output mode based on flags and writes the result.
func emitResult(flags *cliFlags, r *TokenResult) error {
	if flags.outputJSON {
		return writeJSON(r)
	}

	writePlain(r)

	return nil
}

// printVersion prints version metadata derived from embedded build info.
func printVersion() {
	versionInfo := resolveVersionAndTime()
	printOutput(MsgVersionFmt, versionInfo.Revision, versionInfo.BuildTimestamp)
}

// readBuildInfo retrieves build info if available.
func readBuildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}

	return info
}

// buildSettingsMap converts build settings to a map for quick lookup.
func buildSettingsMap(info *debug.BuildInfo) map[string]string {
	settingsMap := make(map[string]string, SettingsInitCap)
	if info == nil {
		return settingsMap
	}

	for _, s := range info.Settings {
		settingsMap[s.Key] = s.Value
	}

	return settingsMap
}

// readInputNonText considers file, args, then stdin in that order.

func readInputNonText(flags *cliFlags) (string, error) {
	if flags.inputFile != "" {
		return readFile(flags.inputFile)
	}

	joined := strings.Join(flag.Args(), " ")
	if joined != "" {
		return joined, nil
	}

	return readStdin()
}

// tokenize returns a TokenResult without normalization.
func tokenize(text string) (*TokenResult, error) {
	tok := tokenizer.NewTokenizer()

	return &TokenResult{
		Text:           text,
		Model:          tok.GetModel(),
		OriginalText:   "",
		NormalizedText: "",
		TokenCount:     tok.EstimateTokens(text),
	}, nil
}

// tokenizeNormalized returns a TokenResult with normalization.
func tokenizeNormalized(text string) (*TokenResult, error) {
	tok := tokenizer.NewTokenizer()
	norm := tok.Normalize(text)

	return &TokenResult{
		Text:           text,
		Model:          tok.GetModel(),
		OriginalText:   text,
		NormalizedText: norm,
		TokenCount:     tok.EstimateTokens(text),
	}, nil
}

// tokenizeText is kept for test compatibility; delegates to explicit variants.
//

func tokenizeText(text string, showNormalized bool) (*TokenResult, error) {
	if showNormalized {
		return tokenizeNormalized(text)
	}

	return tokenize(text)
}

// readFile reads the entire file content after sanitizing the provided path.
func readFile(filename string) (string, error) {
	clean := filepath.Clean(filename)
	// #nosec G304 — path cleaned; CLI tool intended to read user-provided files.
	data, err := os.ReadFile(clean)
	if err != nil {
		return "", fmt.Errorf(ErrOpenFileFmt, filename, err)
	}

	return string(data), nil
}

// readStdin reads all data from standard input and wraps errors with context.
func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf(ErrWrapReadStdin, err)
	}

	return string(data), nil
}

// printError writes formatted output to stderr; critical failure if writing fails.
func printError(format string, args ...any) {
	_, e := fmt.Fprintf(os.Stderr, format, args...)
	if e != nil {
		// If unable to write to stderr, terminate loudly.
		panic(e)
	}
}

// printOutput writes formatted output to stdout; critical failure if writing fails.
func printOutput(format string, args ...any) {
	_, e := fmt.Fprintf(os.Stdout, format, args...)
	if e != nil {
		// If unable to write to stdout, terminate loudly.
		panic(e)
	}
}

// printUsage prints the CLI usage text with examples and flag defaults.
func printUsage() {
	exe := ExecutableDefault

	path, execErr := os.Executable()
	if execErr == nil && path != "" {
		exe = filepath.Base(path)
	}

	printOutput(UsageHeader)
	printOutput(UsageUsageFmt, exe)
	printOutput(UsageRules)
	printOutput(UsageOptions)
	flag.PrintDefaults()
	printOutput(UsageExamplesFmt, exe, exe, exe, exe, exe)
}

// truncateText returns a shortened representation with ellipsis if needed.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	if maxLen <= EllipsisLen {
		return strings.Repeat(".", maxLen)
	}

	return text[:maxLen-EllipsisLen] + "..."
}

// writeJSON pretty-prints the TokenResult as JSON to stdout and wraps errors.
func writeJSON(result *TokenResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", MsgJSONIndent)

	err := enc.Encode(result)
	if err != nil {
		return fmt.Errorf(ErrWrapEncodeJSON, err)
	}

	return nil
}

// writePlain prints a human-friendly representation to stdout.
func writePlain(result *TokenResult) {
	printOutput(MsgTextFmt, truncateText(result.Text, DefaultPreviewMax))
	printOutput(MsgTokenCountFmt, result.TokenCount)
	printOutput(MsgModelFmt, result.Model)

	if result.NormalizedText != "" {
		printOutput(
			MsgNormalizedFmt,
			truncateText(result.NormalizedText, DefaultPreviewMax),
		)
	}
}
