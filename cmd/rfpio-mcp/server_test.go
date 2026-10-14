package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/bryanchrist/rfpio-go/internal/core"
	"github.com/bryanchrist/rfpio-go/internal/skills"
)

func TestParseAnswersArg(t *testing.T) {
	want := []core.AnswerUpdate{
		{Key: "Notes/Justification", Value: "n"},
		{Key: "Self Score", Value: "3 - Exceeds requirements"},
	}

	got, err := parseAnswersArg(map[string]any{"Self Score": "3 - Exceeds requirements", "Notes/Justification": "n"})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("object: got %v, %v", got, err)
	}

	got, err = parseAnswersArg(`{"Self Score": "3 - Exceeds requirements", "Notes/Justification": "n"}`)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("JSON string: got %v, %v", got, err)
	}

	for _, empty := range []any{nil, "", map[string]any{}} {
		if got, err := parseAnswersArg(empty); err != nil || len(got) != 0 {
			t.Errorf("%#v: got %v, %v", empty, got, err)
		}
	}

	bad := map[string]any{
		"non-string value": map[string]any{"Self Score": 3},
		"empty value":      map[string]any{"Self Score": ""},
		"empty key":        map[string]any{" ": "x"},
		"not an object":    []any{"x"},
		"bad JSON":         "{not json",
	}
	for name, arg := range bad {
		if _, err := parseAnswersArg(arg); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestUpdateAnswerToolSchema(t *testing.T) {
	for _, tool := range tools {
		if tool.Name != "update_answer" {
			continue
		}
		for _, p := range []string{"content_id", "answer", "answer_key", "answers", "allow_under_review"} {
			if _, ok := tool.InputSchema.Properties[p]; !ok {
				t.Errorf("update_answer missing parameter %q", p)
			}
		}
		for _, phrase := range []string{"search_content(ids=", "never adds a new answer slot", "<br>", "under review"} {
			if !strings.Contains(tool.Description, phrase) {
				t.Errorf("update_answer description missing %q", phrase)
			}
		}
		return
	}
	t.Fatal("update_answer tool not defined")
}

func TestSkillFileAppliedToTools(t *testing.T) {
	docs := skills.ParseSkillMarkdown("../../SKILL.md")
	if docs == nil {
		t.Fatal("SKILL.md not found")
	}
	applied := make([]mcp.Tool, len(tools))
	copy(applied, tools)
	for i := range applied {
		props := make(map[string]any, len(applied[i].InputSchema.Properties))
		for k, v := range applied[i].InputSchema.Properties {
			prop := make(map[string]any)
			for pk, pv := range v.(map[string]any) {
				prop[pk] = pv
			}
			props[k] = prop
		}
		applied[i].InputSchema.Properties = props
	}
	skills.ApplyToTools(applied, docs)

	byName := map[string]mcp.Tool{}
	for _, tool := range applied {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"add_answer", "update_answer", "delete_answers", "create_tag", "update_tag", "delete_tag"} {
		if !strings.Contains(byName[name].Description, "[WRITE OPERATION]") {
			t.Errorf("%s description lost its write marker", name)
		}
	}
	update := byName["update_answer"]
	for _, phrase := range []string{"search_content(ids=", "never adds a new answer slot", "answer_key", "<br>", "under review"} {
		if !strings.Contains(update.Description, phrase) {
			t.Errorf("applied update_answer description missing %q", phrase)
		}
	}
	for _, p := range []string{"answer", "answer_key", "answers", "allow_under_review"} {
		desc, _ := update.InputSchema.Properties[p].(map[string]any)["description"].(string)
		if desc == "" {
			t.Errorf("update_answer.%s has no description", p)
		}
	}
	if strings.Contains(byName["add_answer"].Description, "will cause shell errors") {
		t.Error("add_answer still carries the old entity-encoding advice")
	}
}
