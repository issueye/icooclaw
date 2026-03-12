package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkillParser_Parse(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantName    string
		wantDesc    string
		wantContent string
	}{
		{
			name: "YAML frontmatter",
			content: `---
name: test-skill
description: A test skill
version: 1.0.0
---

# Test Skill Content

This is the skill body.`,
			wantErr:     false,
			wantName:    "test-skill",
			wantDesc:    "A test skill",
			wantContent: "# Test Skill Content\n\nThis is the skill body.",
		},
		{
			name: "JSON frontmatter",
			content: `---
{
  "name": "json-skill",
  "description": "JSON formatted skill",
  "version": "2.0.0"
}
---

# JSON Skill Content`,
			wantErr:     false,
			wantName:    "json-skill",
			wantDesc:    "JSON formatted skill",
			wantContent: "# JSON Skill Content",
		},
		{
			name: "No frontmatter",
			content: `# Plain Skill

This skill has no frontmatter.`,
			wantErr: true, // frontmatter is required
		},
		{
			name: "With author",
			content: `---
name: authored-skill
description: Skill with author
author: test-author
---

# Authored Skill`,
			wantErr:  false,
			wantName: "authored-skill",
			wantDesc: "Skill with author",
		},
		{
			name: "Windows line endings",
			content: "---\r\nname: windows-skill\r\ndescription: Windows line endings\r\n---\r\n\r\n# Content",
			wantErr:  false,
			wantName: "windows-skill",
			wantDesc: "Windows line endings",
		},
		{
			name: "Empty content",
			content: `---
name: empty-skill
description: Empty
---

`,
			wantErr: true, // Content is empty
		},
		{
			name: "Invalid name with special chars",
			content: `---
name: "invalid name!"
description: Invalid name
---

# Content`,
			wantErr: true, // Invalid name pattern
		},
		{
			name: "Missing name",
			content: `---
description: No name provided
---

# Content`,
			wantErr: true, // name is required
		},
		{
			name: "Missing description",
			content: `---
name: no-desc
---

# Content`,
			wantErr: true, // description is required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill, err := parser.Parse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if skill.Name != tt.wantName {
				t.Errorf("Parse() name = %q, want %q", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("Parse() description = %q, want %q", skill.Description, tt.wantDesc)
			}
			if tt.wantContent != "" && skill.Content != tt.wantContent {
				t.Errorf("Parse() content = %q, want %q", skill.Content, tt.wantContent)
			}
		})
	}
}

func TestSkillParser_ParseFile(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewParser()

	t.Run("Valid skill file", func(t *testing.T) {
		skillDir := filepath.Join(tmpDir, "test-skill")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("failed to create skill directory: %v", err)
		}

		skillFile := filepath.Join(skillDir, "SKILL.md")
		content := `---
name: file-skill
description: A skill from file
version: 1.0.0
---

# File Skill Content`
		if err := os.WriteFile(skillFile, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}

		skill, err := parser.ParseFile(skillFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		if skill.Name != "file-skill" {
			t.Errorf("ParseFile() name = %q, want %q", skill.Name, "file-skill")
		}
		if skill.FilePath != skillFile {
			t.Errorf("ParseFile() filePath = %q, want %q", skill.FilePath, skillFile)
		}
	})

	t.Run("Non-existent file", func(t *testing.T) {
		_, err := parser.ParseFile("/nonexistent/SKILL.md")
		if err == nil {
			t.Error("ParseFile() expected error for non-existent file")
		}
	})
}

func TestSkillParser_Validate(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		skill   *ParsedSkill
		wantErr bool
	}{
		{
			name: "Valid skill",
			skill: &ParsedSkill{
				Name:        "valid-skill",
				Description: "A valid skill",
				Content:     "Some content",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			skill: &ParsedSkill{
				Content:     "Some content",
				Description: "A skill",
			},
			wantErr: true,
		},
		{
			name: "Empty content",
			skill: &ParsedSkill{
				Name:        "empty-content",
				Description: "A skill",
			},
			wantErr: true,
		},
		{
			name: "Empty description",
			skill: &ParsedSkill{
				Name:    "no-desc",
				Content: "Some content",
			},
			wantErr: true,
		},
		{
			name: "Name too long",
			skill: &ParsedSkill{
				Name:        string(make([]byte, 100)),
				Description: "A skill",
				Content:     "Some content",
			},
			wantErr: true,
		},
		{
			name: "Invalid name pattern",
			skill: &ParsedSkill{
				Name:        "invalid name!",
				Description: "A skill",
				Content:     "Some content",
			},
			wantErr: true,
		},
		{
			name: "Description too long",
			skill: &ParsedSkill{
				Name:        "long-desc",
				Description: string(make([]byte, 1500)),
				Content:     "Some content",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Validate(tt.skill)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSkillParser_CreateSkillFile(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewParser()

	skill := &ParsedSkill{
		Name:        "created-skill",
		Version:     "1.0.0",
		Description: "A created skill",
		Author:      "test-author",
		Content:     "# Created Skill\n\nThis is the content.",
	}

	err := parser.CreateSkillFile(tmpDir, skill)
	if err != nil {
		t.Fatalf("CreateSkillFile() error = %v", err)
	}

	// Verify file was created
	skillFile := filepath.Join(tmpDir, "created-skill", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("skill file not created: %v", err)
	}

	// Read and verify content
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("failed to read skill file: %v", err)
	}

	content := string(data)
	if !contains(content, "name: created-skill") {
		t.Error("skill file missing name in frontmatter")
	}
	if !contains(content, "# Created Skill") {
		t.Error("skill file missing content")
	}

	// Parse the created file to verify round-trip
	parsed, err := parser.ParseFile(skillFile)
	if err != nil {
		t.Fatalf("failed to parse created file: %v", err)
	}

	if parsed.Name != skill.Name {
		t.Errorf("round-trip name = %q, want %q", parsed.Name, skill.Name)
	}
	if parsed.Description != skill.Description {
		t.Errorf("round-trip description = %q, want %q", parsed.Description, skill.Description)
	}
}

func TestSkillParser_ParseFrontmatterOnly(t *testing.T) {
	parser := NewParser()

	content := `---
name: meta-only
description: Just metadata
version: 2.0.0
---

# Body content`

	meta, err := parser.ParseFrontmatterOnly(content)
	if err != nil {
		t.Fatalf("ParseFrontmatterOnly() error = %v", err)
	}

	if meta.Name != "meta-only" {
		t.Errorf("ParseFrontmatterOnly() name = %q, want %q", meta.Name, "meta-only")
	}
	if meta.Version != "2.0.0" {
		t.Errorf("ParseFrontmatterOnly() version = %q, want %q", meta.Version, "2.0.0")
	}
}

func TestParseError(t *testing.T) {
	err := &ParseError{Field: "name", Message: "is required"}
	expected := "skill parse error [name]: is required"
	if err.Error() != expected {
		t.Errorf("ParseError.Error() = %q, want %q", err.Error(), expected)
	}

	err2 := &ParseError{Message: "generic error"}
	expected2 := "skill parse error: generic error"
	if err2.Error() != expected2 {
		t.Errorf("ParseError.Error() = %q, want %q", err2.Error(), expected2)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}