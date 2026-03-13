// Package skill provides skill management for icooclaw.
package skill

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Skill file patterns
var (
	// namePattern validates skill names: alphanumeric with hyphens
	namePattern = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)
	// reFrontmatter matches YAML/JSON frontmatter between --- markers
	reFrontmatter = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---`)
	// reStripFrontmatter removes frontmatter from content
	reStripFrontmatter = regexp.MustCompile(`(?s)^---(?:\r\n|\n|\r)(.*?)(?:\r\n|\n|\r)---(?:\r\n|\n|\r)*`)
)

// Validation constants
const (
	MaxNameLength        = 64
	MaxDescriptionLength = 1024
	MaxVersionLength     = 32
	MaxContentLength     = 100 * 1024 // 100KB
)

// ParsedSkill 解析后的技能，包含元数据和内容。
type ParsedSkill struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description"`
	Author      string `json:"author,omitempty"`
	Content     string `json:"content"`
	FilePath    string `json:"file_path,omitempty"`
}

// SkillFrontmatter 抽取技能文件的元数据。
type SkillFrontmatter struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version,omitempty" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author,omitempty" yaml:"author"`
}

// ParseError 解析技能时的错误。
type ParseError struct {
	Field   string
	Message string
}

func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("skill parse error [%s]: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("skill parse error: %s", e.Message)
}

// SkillParser 解析技能文件。
type SkillParser struct{}

// NewParser 创建一个新的技能解析器。
func NewParser() *SkillParser {
	return &SkillParser{}
}

// ParseFile 解析给定路径的技能文件。
func (p *SkillParser) ParseFile(path string) (*ParsedSkill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	if len(content) > MaxContentLength {
		return nil, &ParseError{Message: fmt.Sprintf("content exceeds %d bytes", MaxContentLength)}
	}

	skill, err := p.Parse(string(content))
	if err != nil {
		return nil, err
	}

	skill.FilePath = path
	return skill, nil
}

// Parse 解析技能内容，包含元数据和主体。
func (p *SkillParser) Parse(content string) (*ParsedSkill, error) {
	frontmatter, body := p.extractFrontmatter(content)

	skill := &ParsedSkill{
		Content: strings.TrimSpace(body),
	}

	// If no frontmatter, try to use first line as name
	if frontmatter == "" {
		return nil, &ParseError{Field: "frontmatter", Message: "missing frontmatter (expected --- delimited block)"}
	}

	// Parse frontmatter
	meta, err := p.parseFrontmatter(frontmatter)
	if err != nil {
		return nil, err
	}

	skill.Name = meta.Name
	skill.Version = meta.Version
	skill.Description = meta.Description
	skill.Author = meta.Author

	// Validate
	if err := p.Validate(skill); err != nil {
		return nil, err
	}

	return skill, nil
}

// ParseFrontmatterOnly 解析技能内容的元数据。
func (p *SkillParser) ParseFrontmatterOnly(content string) (*SkillFrontmatter, error) {
	frontmatter, _ := p.extractFrontmatter(content)
	if frontmatter == "" {
		return nil, &ParseError{Field: "frontmatter", Message: "missing frontmatter"}
	}
	return p.parseFrontmatter(frontmatter)
}

// Validate 验证解析后的技能是否有效。
func (p *SkillParser) Validate(skill *ParsedSkill) error {
	var errs error

	// Validate name
	if skill.Name == "" {
		errs = errors.Join(errs, &ParseError{Field: "name", Message: "name is required"})
	} else {
		if len(skill.Name) > MaxNameLength {
			errs = errors.Join(errs, &ParseError{Field: "name", Message: fmt.Sprintf("exceeds %d characters", MaxNameLength)})
		}
		if !namePattern.MatchString(skill.Name) {
			errs = errors.Join(errs, &ParseError{Field: "name", Message: "must be alphanumeric with hyphens (e.g., my-skill-name)"})
		}
	}

	// Validate version (optional but must be valid if present)
	if skill.Version != "" && len(skill.Version) > MaxVersionLength {
		errs = errors.Join(errs, &ParseError{Field: "version", Message: fmt.Sprintf("exceeds %d characters", MaxVersionLength)})
	}

	// Validate description
	if skill.Description == "" {
		errs = errors.Join(errs, &ParseError{Field: "description", Message: "description is required"})
	} else if len(skill.Description) > MaxDescriptionLength {
		errs = errors.Join(errs, &ParseError{Field: "description", Message: fmt.Sprintf("exceeds %d characters", MaxDescriptionLength)})
	}

	// Validate content
	if skill.Content == "" {
		errs = errors.Join(errs, &ParseError{Field: "content", Message: "skill content is required"})
	}

	return errs
}

// extractFrontmatter 提取技能内容的元数据和主体。
func (p *SkillParser) extractFrontmatter(content string) (frontmatter, body string) {
	match := reFrontmatter.FindStringSubmatch(content)
	if len(match) > 1 {
		frontmatter = match[1]
		body = reStripFrontmatter.ReplaceAllString(content, "")
	}
	return
}

// parseFrontmatter 解析技能文件的元数据。
func (p *SkillParser) parseFrontmatter(content string) (*SkillFrontmatter, error) {
	content = strings.TrimSpace(content)

	// Try JSON first
	if strings.HasPrefix(content, "{") {
		var meta SkillFrontmatter
		if err := json.Unmarshal([]byte(content), &meta); err != nil {
			return nil, &ParseError{Field: "frontmatter", Message: fmt.Sprintf("invalid JSON: %v", err)}
		}
		return &meta, nil
	}

	// Fall back to simple YAML parsing
	return p.parseSimpleYAML(content), nil
}

// parseSimpleYAML 解析简单的 YAML 格式。
func (p *SkillParser) parseSimpleYAML(content string) *SkillFrontmatter {
	meta := &SkillFrontmatter{}

	// Normalize line endings
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		switch key {
		case "name":
			meta.Name = value
		case "version":
			meta.Version = value
		case "description":
			meta.Description = value
		case "author":
			meta.Author = value
		}
	}

	return meta
}

// CreateSkillFile 创建一个新的技能文件。
func (p *SkillParser) CreateSkillFile(dir string, skill *ParsedSkill) error {
	if err := p.Validate(skill); err != nil {
		return err
	}

	// Create skill directory
	skillDir := filepath.Join(dir, skill.Name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	// Build file content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", skill.Name))
	if skill.Version != "" {
		sb.WriteString(fmt.Sprintf("version: %s\n", skill.Version))
	}
	sb.WriteString(fmt.Sprintf("description: %s\n", skill.Description))
	if skill.Author != "" {
		sb.WriteString(fmt.Sprintf("author: %s\n", skill.Author))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(skill.Content)

	// Write file
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}
