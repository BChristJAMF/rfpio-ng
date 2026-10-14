package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolDoc is the documentation SKILL.md provides for one tool.
type ToolDoc struct {
	Description string
	Params      map[string]string
}

var (
	toolHeadingRe = regexp.MustCompile(`^## +(\w+)`)
	paramHeadRe   = regexp.MustCompile(`^\*\*(\w+)\*\*`)
)

// ParseSkillMarkdown parses a SKILL.md file into per-tool documentation,
// matching the Python rfpio_skill_loader. Expected format:
//
//	## tool_name
//
//	### Description
//
//	Description text (kept verbatim, markdown included)...
//
//	### Parameters
//
//	**param_name** (type, ...)
//	- Description line
//	- Another line
//
// A tool's description is the "### Description" subsection; each parameter's
// description is its "- " lines joined with spaces. Other subsections
// (Constraints, Important Notes, Returns) are not used. Returns nil if the
// file cannot be read.
func ParseSkillMarkdown(path string) map[string]ToolDoc {
	content, err := os.ReadFile(expandTilde(path))
	if err != nil {
		return nil
	}

	docs := make(map[string]ToolDoc)
	var tool, section, param string
	var desc []string
	var params map[string]string
	var paramLines []string

	flushParam := func() {
		if param != "" {
			params[param] = strings.Join(paramLines, " ")
		}
		param, paramLines = "", nil
	}
	flushTool := func() {
		flushParam()
		if tool != "" {
			docs[tool] = ToolDoc{
				Description: strings.TrimSpace(strings.Join(desc, "\n")),
				Params:      params,
			}
		}
	}

	for _, line := range strings.Split(string(content), "\n") {
		if m := toolHeadingRe.FindStringSubmatch(line); m != nil {
			flushTool()
			tool, section, desc, params = m[1], "", nil, make(map[string]string)
			continue
		}
		if tool == "" {
			continue
		}
		if strings.HasPrefix(line, "###") {
			flushParam()
			section = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}

		switch section {
		case "Description":
			if strings.TrimSpace(line) != "---" {
				desc = append(desc, line)
			}
		case "Parameters":
			trimmed := strings.TrimSpace(line)
			if m := paramHeadRe.FindStringSubmatch(trimmed); m != nil {
				flushParam()
				param = m[1]
				continue
			}
			if param == "" || trimmed == "" || trimmed == "---" {
				continue
			}
			paramLines = append(paramLines, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
		}
	}
	flushTool()

	return docs
}

// ApplyToTools overwrites tool and parameter descriptions with the SKILL.md
// text where it exists, leaving the built-in description otherwise. Only
// parameters already in a tool's schema are touched. Returns the number of
// tools whose description was replaced.
func ApplyToTools(tools []mcp.Tool, docs map[string]ToolDoc) int {
	applied := 0
	for i := range tools {
		doc, ok := docs[tools[i].Name]
		if !ok {
			continue
		}
		if doc.Description != "" {
			tools[i].Description = doc.Description
			applied++
		}
		for name, text := range doc.Params {
			if prop, ok := tools[i].InputSchema.Properties[name].(map[string]any); ok && text != "" {
				prop["description"] = text
			}
		}
	}
	return applied
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		if home == "" {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		return os.Getenv("HOME")
	}
	return path
}
