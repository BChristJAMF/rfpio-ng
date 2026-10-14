package core

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

const fixtureID = "62a000000000000000000002"

// fakeRFPIO serves a fixture record on GET and records POST bodies.
type fakeRFPIO struct {
	record map[string]any
	posts  []map[string]any
	paths  []string
}

func newFakeRFPIO(t *testing.T) (*fakeRFPIO, *Config) {
	t.Helper()
	return newFakeRFPIOFor(t, "../../fixtures/answer_get_named_answers.json", fixtureID)
}

// newFakeRFPIOFor serves the record in fixture under id.
func newFakeRFPIOFor(t *testing.T, fixture, id string) (*fakeRFPIO, *Config) {
	t.Helper()
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	f := &fakeRFPIO{}
	if err := json.Unmarshal(raw, &f.record); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/answer-lib/get/"+id:
			w.Write(raw)
		case r.Method == http.MethodPost && r.URL.Path == "/answer-lib/update/"+id:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Errorf("update body is not JSON: %v", err)
			}
			f.posts = append(f.posts, payload)
			f.paths = append(f.paths, r.URL.Path)
			w.Write([]byte(`{}`))
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	return f, &Config{APIToken: "test", SiteURL: srv.URL}
}

func strPtr(s string) *string { return &s }

func TestAPIUpdateNamedAnswerPreservesRecord(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	res, err := APIUpdate(cfg, fixtureID, nil,
		[]AnswerUpdate{{Key: "notes/justification", Value: "Sample notes (with parentheses)."}}, nil, nil, false)
	if err != nil {
		t.Fatalf("APIUpdate: %v", err)
	}
	if !res.Success {
		t.Fatalf("APIUpdate failed: %s", res.Message)
	}
	if len(f.posts) != 1 {
		t.Fatalf("POSTs = %d, want 1", len(f.posts))
	}
	sent := f.posts[0]

	// Every field except answers and the server-computed fields is sent back
	// untouched; the computed fields are left for the server to recompute.
	derived := map[string]bool{}
	for _, k := range serverComputedFields {
		derived[k] = true
		if _, ok := f.record[k]; !ok {
			t.Fatalf("fixture lacks server-computed field %q", k)
		}
		if _, ok := sent[k]; ok {
			t.Errorf("stale server-computed field %q was sent", k)
		}
	}
	for k, want := range f.record {
		if k == "answers" || derived[k] {
			continue
		}
		if !reflect.DeepEqual(sent[k], want) {
			t.Errorf("field %q changed: sent %v, want %v", k, sent[k], want)
		}
	}
	if len(sent) != len(f.record)-len(serverComputedFields) {
		t.Errorf("sent %d fields, want %d", len(sent), len(f.record)-len(serverComputedFields))
	}

	answers, _ := sent["answers"].([]any)
	if len(answers) != 2 {
		t.Fatalf("answers = %d, want 2", len(answers))
	}
	orig := f.record["answers"].([]any)
	if !reflect.DeepEqual(answers[0], orig[0]) {
		t.Errorf("Self Score changed: %v", answers[0])
	}
	notes := answers[1].(map[string]any)
	wantNotes := map[string]any{
		"answerSourceType": "",
		"key":              "Notes/Justification",
		"type":             "RICH_TEXT",
		"value":            "<div>Sample notes (with parentheses).</div>",
	}
	if !reflect.DeepEqual(notes, wantNotes) {
		t.Errorf("Notes/Justification = %v, want %v", notes, wantNotes)
	}
}

func TestAPIUpdateMultipleAnswersSingleWrite(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	_, err := APIUpdate(cfg, fixtureID, nil, []AnswerUpdate{
		{Key: "Self Score", Value: "2 - Meets requirements"},
		{Key: "Notes/Justification", Value: "<div>n</div>"},
	}, nil, nil, false)
	if err != nil {
		t.Fatalf("APIUpdate: %v", err)
	}
	if len(f.posts) != 1 {
		t.Fatalf("POSTs = %d, want 1", len(f.posts))
	}
	answers := f.posts[0]["answers"].([]any)
	if a := answers[0].(map[string]any); a["key"] != "Self Score" || a["value"] != "2 - Meets requirements" {
		t.Errorf("answers[0] = %v", a)
	}
	if a := answers[1].(map[string]any); a["key"] != "Notes/Justification" || a["value"] != "<div>n</div>" {
		t.Errorf("answers[1] = %v", a)
	}
}

func TestAPIUpdateUnknownKeyWritesNothing(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	_, err := APIUpdate(cfg, fixtureID, nil, []AnswerUpdate{{Key: "Justification", Value: "x"}}, nil, nil, false)
	if err == nil || !strings.Contains(err.Error(), `"Self Score", "Notes/Justification"`) {
		t.Fatalf("want key-listing error, got %v", err)
	}
	if len(f.posts) != 0 {
		t.Errorf("POSTs = %d, want 0", len(f.posts))
	}
}

func TestAPIUpdateMissingDefaultWritesNothing(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	_, err := APIUpdate(cfg, fixtureID, nil, []AnswerUpdate{{Value: "x"}}, nil, nil, false)
	if err == nil || !strings.Contains(err.Error(), `no "Default" answer`) {
		t.Fatalf("want missing-Default error, got %v", err)
	}
	if len(f.posts) != 0 {
		t.Errorf("POSTs = %d, want 0 (the old code appended a third answer here and got HTTP 400)", len(f.posts))
	}
}

func TestAPIUpdateTagsSentAsList(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	_, err := APIUpdate(cfg, fixtureID, nil, nil, nil, strPtr("Cloud, SLA ,"), false)
	if err != nil {
		t.Fatalf("APIUpdate: %v", err)
	}
	if got := f.posts[0]["tags"]; !reflect.DeepEqual(got, []any{"Cloud", "SLA"}) {
		t.Errorf("tags = %#v, want [Cloud SLA]", got)
	}
	// Nothing answer-related changes on a tags-only update, so the computed
	// fields are current and are sent back as read.
	for _, k := range append([]string{"answers"}, serverComputedFields...) {
		if !reflect.DeepEqual(f.posts[0][k], f.record[k]) {
			t.Errorf("%s changed on a tags-only update", k)
		}
	}
}

func TestCheckAnswerSlots(t *testing.T) {
	before := namedAnswers()
	same, _ := mergeAnswers(before, []AnswerUpdate{{Key: "Self Score", Value: "1"}})
	if err := checkAnswerSlots(before, same); err != nil {
		t.Errorf("same slots rejected: %v", err)
	}
	appended := append(namedAnswers(), map[string]any{"key": "Default", "value": "x"})
	reordered := []any{before[1], before[0]}
	renamed := []any{before[0], map[string]any{"key": "Notes", "value": "x"}}
	for name, after := range map[string][]any{"appended": appended, "reordered": reordered, "renamed": renamed} {
		err := checkAnswerSlots(before, after)
		if err == nil || !strings.Contains(err.Error(), "Nothing was written") {
			t.Errorf("%s: want slot error, got %v", name, err)
		}
	}
}

func TestAPIUpdateNoChanges(t *testing.T) {
	f, cfg := newFakeRFPIO(t)

	res, err := APIUpdate(cfg, fixtureID, nil, nil, nil, nil, false)
	if err != nil || res.Success {
		t.Fatalf("want unsuccessful result, got %v, %v", res, err)
	}
	if len(f.posts) != 0 {
		t.Errorf("POSTs = %d, want 0", len(f.posts))
	}
}

const underReviewID = "62a000000000000000000001"

func TestAPIUpdateRefusesEntryUnderReview(t *testing.T) {
	f, cfg := newFakeRFPIOFor(t, "../../fixtures/answer_get_under_review.json", underReviewID)

	// The guard runs before the answer merge, so even a bad key reports the review.
	for _, answers := range [][]AnswerUpdate{{{Value: "new text"}}, {{Key: "No such key", Value: "x"}}} {
		_, err := APIUpdate(cfg, underReviewID, nil, answers, nil, nil, false)
		var ur *EntryUnderReview
		if !errors.As(err, &ur) {
			t.Fatalf("want *EntryUnderReview, got %T %v", err, err)
		}
		for _, want := range []string{"review is turned on", "review due date 2026-10-16", "review cycle of 90 days", `review workflow "ANY"`, "last reviewed on 2026-07-20", "allow_under_review=true", "Nothing was written"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q missing %q", err.Error(), want)
			}
		}
	}
	if len(f.posts) != 0 {
		t.Errorf("POSTs = %d, want 0", len(f.posts))
	}
}

func TestAPIUpdateUnderReviewOptIn(t *testing.T) {
	f, cfg := newFakeRFPIOFor(t, "../../fixtures/answer_get_under_review.json", underReviewID)

	// The entry's single "Response" answer is the target when no key is given.
	res, err := APIUpdate(cfg, underReviewID, nil, []AnswerUpdate{{Value: "new text"}}, nil, nil, true)
	if err != nil || !res.Success {
		t.Fatalf("APIUpdate with opt-in: %v, %v", res, err)
	}
	if len(f.posts) != 1 {
		t.Fatalf("POSTs = %d, want 1", len(f.posts))
	}
	sent := f.posts[0]
	for _, k := range []string{"reviewStatus", "reviewFlag", "reviewWorkflow", "dueDate", "customReviewCycleInDays", "lastReviewedOn", "followUpDates"} {
		if !reflect.DeepEqual(sent[k], f.record[k]) {
			t.Errorf("review field %q changed: sent %v, read %v", k, sent[k], f.record[k])
		}
	}
	answer := sent["answers"].([]any)[0].(map[string]any)
	if answer["key"] != "Response" || answer["value"] != "<div>new text</div>" {
		t.Errorf("answer = %v", answer)
	}
}
