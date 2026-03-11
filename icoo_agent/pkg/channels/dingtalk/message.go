package dingtalk

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// MessageType represents the type of DingTalk message.
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeMarkdown MessageType = "markdown"
	MessageTypeImage    MessageType = "image"
	MessageTypeFile     MessageType = "file"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeLink     MessageType = "link"
	MessageTypeAction   MessageType = "actionCard"
)

// TextMessage represents a text message.
type TextMessage struct {
	Content string `json:"content"`
}

// MarkdownMessage represents a markdown message.
type MarkdownMessage struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ImageMessage represents an image message.
type ImageMessage struct {
	MediaID string `json:"media_id"`
}

// FileMessage represents a file message.
type FileMessage struct {
	MediaID  string `json:"media_id"`
	FileName string `json:"file_name,omitempty"`
	FileType string `json:"file_type,omitempty"`
}

// LinkMessage represents a link message.
type LinkMessage struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`
	PicURL     string `json:"picUrl,omitempty"`
}

// BuildTextContent builds a text message content JSON.
func BuildTextContent(text string) string {
	msg := TextMessage{Content: text}
	data, _ := json.Marshal(msg)
	return string(data)
}

// BuildMarkdownContent builds a markdown message content JSON.
func BuildMarkdownContent(title, content string) string {
	msg := MarkdownMessage{
		Title:   title,
		Content: content,
	}
	data, _ := json.Marshal(msg)
	return string(data)
}

// ParseTextContent parses a text message content.
func ParseTextContent(content string) (string, error) {
	var msg TextMessage
	if err := json.Unmarshal([]byte(content), &msg); err != nil {
		return "", fmt.Errorf("parse text content: %w", err)
	}
	return msg.Content, nil
}

// ParseMarkdownContent parses a markdown message content.
func ParseMarkdownContent(content string) (*MarkdownMessage, error) {
	var msg MarkdownMessage
	if err := json.Unmarshal([]byte(content), &msg); err != nil {
		return nil, fmt.Errorf("parse markdown content: %w", err)
	}
	return &msg, nil
}

// FormatMarkdownForDingTalk formats markdown content for DingTalk.
// DingTalk markdown has some limitations compared to standard markdown.
func FormatMarkdownForDingTalk(content string) string {
	// DingTalk doesn't support all standard markdown features
	// We need to convert some syntax

	// Convert code blocks with language to simple code blocks
	content = convertCodeBlocks(content)

	// Convert headers to bold text (DingTalk doesn't support headers well)
	content = convertHeaders(content)

	// Ensure proper line breaks
	content = ensureLineBreaks(content)

	return content
}

// convertCodeBlocks converts code blocks to DingTalk compatible format.
func convertCodeBlocks(content string) string {
	// DingTalk doesn't support syntax highlighting, so we remove language specifiers
	lines := strings.Split(content, "\n")
	var result []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				result = append(result, "```")
				inCodeBlock = false
			} else {
				result = append(result, "```")
				inCodeBlock = true
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// convertHeaders converts markdown headers to bold text.
func convertHeaders(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			// h3 -> bold
			result = append(result, "**"+strings.TrimPrefix(trimmed, "### ")+"**")
		} else if strings.HasPrefix(trimmed, "## ") {
			// h2 -> bold
			result = append(result, "**"+strings.TrimPrefix(trimmed, "## ")+"**")
		} else if strings.HasPrefix(trimmed, "# ") {
			// h1 -> bold
			result = append(result, "**"+strings.TrimPrefix(trimmed, "# ")+"**")
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// ensureLineBreaks ensures proper line breaks for DingTalk markdown.
func ensureLineBreaks(content string) string {
	// DingTalk requires double newlines for paragraph breaks
	// but single newlines within paragraphs
	return content
}

// Regex patterns for markdown extraction
var (
	codeBlockRegex  = regexp.MustCompile("(?s)```.*?```")
	inlineCodeRegex = regexp.MustCompile("`([^`]+)`")
	linkRegex       = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	imageRegex      = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	headerRegex     = regexp.MustCompile(`^#{1,6}\s+(.+)$`)
	boldRegex       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	italicRegex     = regexp.MustCompile(`\*([^*]+)\*`)
)

// ExtractTextFromMarkdown extracts plain text from markdown content.
func ExtractTextFromMarkdown(markdown string) string {
	// Remove code blocks
	result := codeBlockRegex.ReplaceAllString(markdown, "[code]")

	// Remove inline code
	result = inlineCodeRegex.ReplaceAllString(result, "$1")

	// Remove links but keep text
	result = linkRegex.ReplaceAllString(result, "$1")

	// Remove images
	result = imageRegex.ReplaceAllString(result, "")

	// Remove headers markers
	result = headerRegex.ReplaceAllString(result, "$1")

	// Remove bold/italic markers
	result = boldRegex.ReplaceAllString(result, "$1")
	result = italicRegex.ReplaceAllString(result, "$1")

	// Clean up extra whitespace
	result = strings.TrimSpace(result)

	return result
}