// Package channels provides channel management for icooclaw.
package channels

import (
	"math"
	"strings"
	"unicode/utf8"
)

// Default message length limits per channel.
var channelMaxLen = map[string]int{
	"dingtalk": 4096,
	"feishu":   4096,
	"telegram": 4096,
	"discord":  2000,
	"slack":    40000,
	"web":      100000,
}

// GetMaxMessageLength returns the max message length for a channel.
func GetMaxMessageLength(channel string) int {
	if maxLen, ok := channelMaxLen[channel]; ok {
		return maxLen
	}
	return 4096 // default
}

// SplitMessage splits a message into chunks that fit within maxLen.
// It tries to preserve code block integrity.
func SplitMessage(content string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = 4096
	}

	if utf8.RuneCountInString(content) <= maxLen {
		return []string{content}
	}

	runes := []rune(content)
	var chunks []string

	for len(runes) > 0 {
		chunkEnd := min(maxLen, len(runes))
		chunk := runes[:chunkEnd]

		// Check for unclosed code blocks
		if hasUnclosedCodeBlock(string(chunk)) {
			// Try to extend to include closing ```
			closingIdx := findClosingCodeBlock(runes, chunkEnd)
			if closingIdx > 0 && closingIdx <= maxLen*3/2 {
				chunkEnd = closingIdx
				chunk = runes[:chunkEnd]
			} else {
				// Inject closing and reopening code blocks
				chunkStr := strings.TrimRight(string(chunk), " \t\n\r") + "\n```\n"
				chunks = append(chunks, chunkStr)
				runes = append([]rune("```\n"), runes[chunkEnd:]...)
				continue
			}
		}

		// Try to break at a newline
		if breakIdx := findBreakPoint(chunk); breakIdx > 0 {
			chunkEnd = breakIdx
			chunk = runes[:chunkEnd]
		}

		chunks = append(chunks, string(chunk))
		runes = runes[chunkEnd:]
	}

	return chunks
}

// hasUnclosedCodeBlock checks if there's an unclosed code block.
func hasUnclosedCodeBlock(s string) bool {
	count := strings.Count(s, "```")
	return count%2 != 0
}

// findClosingCodeBlock finds the closing ``` after start.
func findClosingCodeBlock(runes []rune, start int) int {
	for i := start; i < len(runes)-2; i++ {
		if runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			return i + 3
		}
	}
	return -1
}

// findBreakPoint finds a good break point in the chunk.
func findBreakPoint(chunk []rune) int {
	// Prefer breaking at double newline
	for i := len(chunk) - 1; i >= 0; i-- {
		if i > 0 && chunk[i] == '\n' && chunk[i-1] == '\n' {
			return i + 1
		}
	}

	// Then single newline
	for i := len(chunk) - 1; i >= 0; i-- {
		if chunk[i] == '\n' {
			return i + 1
		}
	}

	// Then space
	for i := len(chunk) - 1; i >= 0; i-- {
		if chunk[i] == ' ' || chunk[i] == '\t' {
			return i + 1
		}
	}

	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TruncateMessage truncates a message to maxLen with ellipsis.
func TruncateMessage(content string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 4096
	}

	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}

	// Reserve space for ellipsis
	maxLen -= 3
	if maxLen < 0 {
		maxLen = 0
	}

	return string(runes[:maxLen]) + "..."
}

// EstimateTokens estimates the number of tokens in a message.
func EstimateTokens(content string) int {
	// Rough estimation: ~4 characters per token for English
	// ~2 characters per token for Chinese/Japanese
	charCount := utf8.RuneCountInString(content)

	// Count CJK characters
	cjkCount := 0
	for _, r := range content {
		if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3040 && r <= 0x309F || // Hiragana
			r >= 0x30A0 && r <= 0x30FF || // Katakana
			r >= 0xAC00 && r <= 0xD7AF { // Hangul
			cjkCount++
		}
	}

	nonCjkCount := charCount - cjkCount
	return int(math.Ceil(float64(cjkCount)/2 + float64(nonCjkCount)/4))
}
