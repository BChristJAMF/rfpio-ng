package core

import "fmt"

// underReviewReasons lists the review settings a record has switched on, worded
// for a user. An empty result means the entry is not under review.
//
// The update endpoint replaces the whole record, and it is not known that an
// update preserves these settings: the published update type names the review
// switch enableReview while records carry reviewStatus. APIUpdate therefore
// refuses entries with any of them set unless the caller opts in.
func underReviewReasons(record map[string]any) []string {
	var reasons []string
	if on, _ := record["reviewStatus"].(bool); on {
		reasons = append(reasons, "review is turned on")
	}
	if on, _ := record["reviewFlag"].(bool); on {
		reasons = append(reasons, "flagged for review")
	}
	if s, _ := record["dueDate"].(string); s != "" {
		reasons = append(reasons, "review due date "+s)
	}
	if days, _ := record["customReviewCycleInDays"].(float64); days > 0 {
		reasons = append(reasons, fmt.Sprintf("review cycle of %d days", int(days)))
	}
	if s, _ := record["reviewWorkflow"].(string); s != "" {
		reasons = append(reasons, fmt.Sprintf("review workflow %q", s))
	}
	if dates, _ := record["followUpDates"].([]any); len(dates) > 0 {
		reasons = append(reasons, "review follow-up dates set")
	}
	if s, _ := record["lastReviewedOn"].(string); s != "" {
		reasons = append(reasons, "last reviewed on "+s)
	}
	return reasons
}
