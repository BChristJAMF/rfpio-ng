package core

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// namedAnswers mirrors the answers in fixtures/answer_get_named_answers.json.
func namedAnswers() []any {
	return []any{
		map[string]any{"answerSourceType": "", "key": "Self Score", "type": "SINGLE", "value": "3 - Exceeds requirements"},
		map[string]any{"answerSourceType": "", "key": "Notes/Justification", "type": "RICH_TEXT", "value": "<div>old notes</div>"},
	}
}

func answerAt(t *testing.T, answers []any, i int) map[string]any {
	t.Helper()
	m, ok := answers[i].(map[string]any)
	if !ok {
		t.Fatalf("answers[%d] is %T, want map", i, answers[i])
	}
	return m
}

func TestMergeAnswersNamedKeyExact(t *testing.T) {
	existing := namedAnswers()
	got, err := mergeAnswers(existing, []AnswerUpdate{{Key: "Notes/Justification", Value: "<div>new notes</div>"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(answers) = %d, want 2 (no slot may be added)", len(got))
	}
	if v := answerAt(t, got, 1)["value"]; v != "<div>new notes</div>" {
		t.Errorf("Notes/Justification = %q", v)
	}
	if !reflect.DeepEqual(got[0], existing[0]) {
		t.Errorf("Self Score changed: %v", got[0])
	}
	// Order and the other fields of the replaced entry are preserved.
	notes := answerAt(t, got, 1)
	if notes["key"] != "Notes/Justification" || notes["type"] != "RICH_TEXT" || notes["answerSourceType"] != "" {
		t.Errorf("replaced entry lost fields: %v", notes)
	}
}

func TestMergeAnswersCaseInsensitive(t *testing.T) {
	got, err := mergeAnswers(namedAnswers(), []AnswerUpdate{{Key: "  notes/JUSTIFICATION ", Value: "<div>x</div>"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v := answerAt(t, got, 1)["value"]; v != "<div>x</div>" {
		t.Errorf("Notes/Justification = %q", v)
	}
	if k := answerAt(t, got, 1)["key"]; k != "Notes/Justification" {
		t.Errorf("stored key rewritten to %q; the record's own key must be kept", k)
	}
}

func TestMergeAnswersExactMatchBeatsCaseInsensitive(t *testing.T) {
	existing := []any{
		map[string]any{"key": "response", "value": "a"},
		map[string]any{"key": "Response", "value": "b"},
	}
	got, err := mergeAnswers(existing, []AnswerUpdate{{Key: "Response", Value: "new"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answerAt(t, got, 0)["value"] != "a" || answerAt(t, got, 1)["value"] != "new" {
		t.Errorf("got %v", got)
	}
}

func TestMergeAnswersAmbiguousCaseInsensitive(t *testing.T) {
	existing := []any{
		map[string]any{"key": "response", "value": "a"},
		map[string]any{"key": "RESPONSE", "value": "b"},
	}
	_, err := mergeAnswers(existing, []AnswerUpdate{{Key: "Response", Value: "new"}})
	if err == nil || !strings.Contains(err.Error(), "more than one answer") {
		t.Fatalf("want ambiguity error, got %v", err)
	}
}

func TestMergeAnswersUnknownKeyListsKeys(t *testing.T) {
	existing := namedAnswers()
	_, err := mergeAnswers(existing, []AnswerUpdate{{Key: "Justification", Value: "x"}})
	var nf *AnswerKeyNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("want *AnswerKeyNotFound, got %T %v", err, err)
	}
	if !reflect.DeepEqual(nf.AvailableKeys, []string{"Self Score", "Notes/Justification"}) {
		t.Errorf("AvailableKeys = %v", nf.AvailableKeys)
	}
	msg := err.Error()
	for _, want := range []string{`"Justification"`, `"Self Score"`, `"Notes/Justification"`, "Nothing was written"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q missing %s", msg, want)
		}
	}
	if !reflect.DeepEqual(existing, namedAnswers()) {
		t.Errorf("input mutated: %v", existing)
	}
}

func TestMergeAnswersMissingDefaultFails(t *testing.T) {
	_, err := mergeAnswers(namedAnswers(), []AnswerUpdate{{Value: "x"}})
	var nf *AnswerKeyNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("want *AnswerKeyNotFound, got %T %v", err, err)
	}
	msg := err.Error()
	for _, want := range []string{`no "Default" answer`, "answer_key", `"Self Score"`, `"Notes/Justification"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q missing %s", msg, want)
		}
	}
}

func TestMergeAnswersEmptyRecordFails(t *testing.T) {
	_, err := mergeAnswers(nil, []AnswerUpdate{{Value: "x"}})
	if err == nil || !strings.Contains(err.Error(), "(none)") {
		t.Fatalf("want error listing (none), got %v", err)
	}
}

func TestMergeAnswersDefaultAmongSeveral(t *testing.T) {
	existing := []any{
		map[string]any{"key": "Spanish", "value": "hola"},
		map[string]any{"key": "Default", "value": "hello"},
	}
	got, err := mergeAnswers(existing, []AnswerUpdate{{Value: "hi"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || answerAt(t, got, 0)["value"] != "hola" || answerAt(t, got, 1)["value"] != "hi" {
		t.Errorf("got %v", got)
	}
}

func TestMergeAnswersSingleNonDefaultAnswer(t *testing.T) {
	// Most library entries have one answer keyed e.g. "Response"; an update
	// without answer_key keeps targeting it, as before.
	existing := []any{map[string]any{"key": "Response", "type": "RICH_TEXT", "value": "<div>old</div>"}}
	got, err := mergeAnswers(existing, []AnswerUpdate{{Value: "<div>new</div>"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || answerAt(t, got, 0)["key"] != "Response" || answerAt(t, got, 0)["value"] != "<div>new</div>" {
		t.Errorf("got %v", got)
	}
}

func TestMergeAnswersMultipleAtomic(t *testing.T) {
	got, err := mergeAnswers(namedAnswers(), []AnswerUpdate{
		{Key: "Notes/Justification", Value: "<div>n</div>"},
		{Key: "self score", Value: "2 - Meets requirements"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answerAt(t, got, 0)["key"] != "Self Score" || answerAt(t, got, 0)["value"] != "2 - Meets requirements" {
		t.Errorf("Self Score = %v", got[0])
	}
	if answerAt(t, got, 1)["value"] != "<div>n</div>" {
		t.Errorf("Notes = %v", got[1])
	}

	// One bad key rejects the whole set.
	got, err = mergeAnswers(namedAnswers(), []AnswerUpdate{
		{Key: "Notes/Justification", Value: "<div>n</div>"},
		{Key: "Score", Value: "1"},
	})
	if err == nil || got != nil {
		t.Fatalf("want error and nil result, got %v, %v", got, err)
	}
}

func TestMergeAnswersDuplicateTargetFails(t *testing.T) {
	_, err := mergeAnswers(namedAnswers(), []AnswerUpdate{
		{Key: "Notes/Justification", Value: "a"},
		{Key: "notes/justification", Value: "b"},
	})
	if err == nil || !strings.Contains(err.Error(), "both target") {
		t.Fatalf("want duplicate-target error, got %v", err)
	}
}

func TestFormatAnswerValue(t *testing.T) {
	cases := []struct {
		name, value, typ, want string
	}{
		{"plain rich text is wrapped", "Uses fuzzy matching (edit distance) and isn't slow.", "RICH_TEXT",
			"<div>Uses fuzzy matching (edit distance) and isn't slow.</div>"},
		{"plain rich text escapes markup chars", "A & B < C", "RICH_TEXT", "<div>A &amp; B &lt; C</div>"},
		{"plain rich text newlines become br", "line 1\nline 2\r\nline 3", "RICH_TEXT", "<div>line 1<br>line 2<br>line 3</div>"},
		{"html passes through", "<p>Already <b>HTML</b></p>", "RICH_TEXT", "<p>Already <b>HTML</b></p>"},
		{"br-only text passes through", "one<br>two", "RICH_TEXT", "one<br>two"},
		{"entity-encoded text passes through", "a &#40;b&#41; c&#39;s", "RICH_TEXT", "a &#40;b&#41; c&#39;s"},
		{"single is verbatim", "3 - Exceeds requirements", "SINGLE", "3 - Exceeds requirements"},
		{"untyped is verbatim", "A & B", "", "A & B"},
	}
	for _, c := range cases {
		var typ any
		if c.typ != "" {
			typ = c.typ
		}
		if got := formatAnswerValue(c.value, typ); got != c.want {
			t.Errorf("%s: formatAnswerValue(%q) = %q, want %q", c.name, c.value, got, c.want)
		}
	}
}

func TestMergeAnswersFormatsByEntryType(t *testing.T) {
	got, err := mergeAnswers(namedAnswers(), []AnswerUpdate{{Key: "Notes/Justification", Value: "plain text"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v := answerAt(t, got, 1)["value"]; v != "<div>plain text</div>" {
		t.Errorf("value = %q", v)
	}
}

func TestMergeAnswersDuplicateKeyIsAmbiguous(t *testing.T) {
	// Imported entries can repeat a key (e.g. "DESCRIPTION" on every column).
	existing := []any{
		map[string]any{"key": "DESCRIPTION", "type": "RICH_TEXT", "value": "a"},
		map[string]any{"key": "Notes", "type": "RICH_TEXT", "value": "b"},
		map[string]any{"key": "DESCRIPTION", "type": "RICH_TEXT", "value": "c"},
	}
	got, err := mergeAnswers(existing, []AnswerUpdate{{Key: "DESCRIPTION", Value: "x"}})
	if err == nil || got != nil {
		t.Fatalf("want error and nil result, got %v, %v", got, err)
	}
	for _, want := range []string{`"DESCRIPTION"`, "2 times", "answers 1, 3", "counting from 1", "Nothing was written"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}

	// A key that is unique on the same record is still addressable.
	got, err = mergeAnswers(existing, []AnswerUpdate{{Key: "Notes", Value: "<div>n</div>"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answerAt(t, got, 1)["value"] != "<div>n</div>" || answerAt(t, got, 0)["value"] != "a" || answerAt(t, got, 2)["value"] != "c" {
		t.Errorf("got %v", got)
	}
}

func TestMergeAnswersDuplicateKeyRejectsWholeMultiUpdate(t *testing.T) {
	existing := []any{
		map[string]any{"key": "DESCRIPTION", "value": "a"},
		map[string]any{"key": "Notes", "value": "b"},
		map[string]any{"key": "DESCRIPTION", "value": "c"},
	}
	got, err := mergeAnswers(existing, []AnswerUpdate{{Key: "Notes", Value: "n"}, {Key: "DESCRIPTION", Value: "x"}})
	if err == nil || got != nil || !strings.Contains(err.Error(), "2 times") {
		t.Fatalf("want ambiguity error and nil result, got %v, %v", got, err)
	}
}

func TestMergeAnswersDuplicateDefaultIsAmbiguous(t *testing.T) {
	existing := []any{
		map[string]any{"key": "Default", "value": "a"},
		map[string]any{"key": "Default", "value": "b"},
	}
	_, err := mergeAnswers(existing, []AnswerUpdate{{Value: "x"}})
	if err == nil || !strings.Contains(err.Error(), "2 times") {
		t.Fatalf("want ambiguity error, got %v", err)
	}
}
