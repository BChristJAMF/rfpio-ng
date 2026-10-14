package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

const sampleSkill = `# Title

Intro text that belongs to no tool.

---

## first_tool

Short summary (not part of the description).

### Description

**[WRITE OPERATION]**

First tool description.

1. **Note:** keep markdown.

### Parameters

**name** (string, required)
- The name
- Second line

**count** (integer, optional)
- How many

### Constraints

- Not part of the description

---

## second_tool

### Description

Second tool description.

---
`

func writeSkill(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseSkillMarkdown(t *testing.T) {
	docs := ParseSkillMarkdown(writeSkill(t, sampleSkill))

	first, ok := docs["first_tool"]
	if !ok {
		t.Fatalf("first_tool missing: %v", docs)
	}
	wantDesc := "**[WRITE OPERATION]**\n\nFirst tool description.\n\n1. **Note:** keep markdown."
	if first.Description != wantDesc {
		t.Errorf("first_tool description = %q, want %q", first.Description, wantDesc)
	}
	if got := first.Params["name"]; got != "The name Second line" {
		t.Errorf("name param = %q", got)
	}
	if got := first.Params["count"]; got != "How many" {
		t.Errorf("count param = %q", got)
	}
	if len(first.Params) != 2 {
		t.Errorf("params = %v, want only name and count", first.Params)
	}

	// Every section keeps its own description, not just the last one.
	if got := docs["second_tool"].Description; got != "Second tool description." {
		t.Errorf("second_tool description = %q", got)
	}

	if ParseSkillMarkdown(filepath.Join(t.TempDir(), "missing.md")) != nil {
		t.Error("missing file should return nil")
	}
}

func TestApplyToTools(t *testing.T) {
	tools := []mcp.Tool{
		mcp.NewTool("first_tool",
			mcp.WithDescription("built-in"),
			mcp.WithString("name", mcp.Description("built-in name")),
			mcp.WithString("other", mcp.Description("built-in other")),
		),
		mcp.NewTool("undocumented", mcp.WithDescription("keep me")),
	}
	n := ApplyToTools(tools, ParseSkillMarkdown(writeSkill(t, sampleSkill)))
	if n != 1 {
		t.Errorf("applied = %d, want 1", n)
	}
	if !strings.HasPrefix(tools[0].Description, "**[WRITE OPERATION]**") {
		t.Errorf("first_tool description = %q", tools[0].Description)
	}
	props := tools[0].InputSchema.Properties
	if got := props["name"].(map[string]any)["description"]; got != "The name Second line" {
		t.Errorf("name description = %v", got)
	}
	if got := props["other"].(map[string]any)["description"]; got != "built-in other" {
		t.Errorf("undocumented param changed: %v", got)
	}
	if _, ok := props["count"]; ok {
		t.Error("ApplyToTools added a parameter that is not in the schema")
	}
	if tools[1].Description != "keep me" {
		t.Errorf("undocumented tool changed: %q", tools[1].Description)
	}
}
