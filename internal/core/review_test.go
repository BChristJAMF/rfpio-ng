package core

import (
	"encoding/json"
	"os"
	"testing"
)

func loadFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return record
}

func TestUnderReviewReasons(t *testing.T) {
	cases := []struct {
		name   string
		record map[string]any
		want   string
	}{
		{"review on", map[string]any{"reviewStatus": true}, "review is turned on"},
		{"flagged", map[string]any{"reviewFlag": true}, "flagged for review"},
		{"due date", map[string]any{"dueDate": "2026-10-16"}, "review due date 2026-10-16"},
		{"cycle", map[string]any{"customReviewCycleInDays": float64(90)}, "review cycle of 90 days"},
		{"workflow", map[string]any{"reviewWorkflow": "ANY"}, `review workflow "ANY"`},
		{"follow-up dates", map[string]any{"followUpDates": []any{"2026-11-01"}}, "review follow-up dates set"},
		{"last reviewed", map[string]any{"lastReviewedOn": "2026-07-20"}, "last reviewed on 2026-07-20"},
	}
	for _, c := range cases {
		got := underReviewReasons(c.record)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%s: got %v, want [%s]", c.name, got, c.want)
		}
	}
}

func TestUnderReviewReasonsIgnoresEmptyValues(t *testing.T) {
	// What the API returns for an entry with no review set up.
	none := map[string]any{
		"reviewStatus": false, "reviewFlag": false, "dueDate": "", "customReviewCycleInDays": float64(-1),
		"reviewWorkflow": "", "followUpDates": []any{}, "lastReviewedOn": "",
	}
	if got := underReviewReasons(none); len(got) != 0 {
		t.Errorf("empty review settings reported as %v", got)
	}
	if got := underReviewReasons(map[string]any{}); len(got) != 0 {
		t.Errorf("missing review settings reported as %v", got)
	}
}

func TestUnderReviewReasonsOnLiveShapedRecords(t *testing.T) {
	if got := underReviewReasons(loadFixture(t, "../../fixtures/answer_get_named_answers.json")); len(got) != 0 {
		t.Errorf("entry without review reported as under review: %v", got)
	}
	// A sanitized record from the live API: review on, due date, 90-day cycle, workflow, last reviewed.
	if got := underReviewReasons(loadFixture(t, "../../fixtures/answer_get_under_review.json")); len(got) != 5 {
		t.Errorf("under-review entry reasons = %v, want 5", got)
	}
}
