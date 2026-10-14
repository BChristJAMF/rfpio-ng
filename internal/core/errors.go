package core

import "fmt"

// RFPIOAPIError represents an error from the RFPIO API.
type RFPIOAPIError struct {
	Message string
	Code    int
}

func (e *RFPIOAPIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("RFPIO API error %d: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("RFPIO API error: %s", e.Message)
}

// IsInvalidTagsError returns true if the error message contains "INVALID_TAGS".
func IsInvalidTagsError(err error) bool {
	if apiErr, ok := err.(*RFPIOAPIError); ok {
		return containsStr(apiErr.Message, "INVALID_TAGS")
	}
	return containsStr(err.Error(), "INVALID_TAGS")
}

// AnswerKeyNotFound is raised when a named-answer update targets a key not on the record.
type AnswerKeyNotFound struct {
	RequestedKey   string
	AvailableKeys  []string
	RFPIOAPIError
}

func NewAnswerKeyNotFound(requestedKey string, availableKeys []string) *AnswerKeyNotFound {
	return &AnswerKeyNotFound{
		RequestedKey:  requestedKey,
		AvailableKeys: availableKeys,
		RFPIOAPIError: RFPIOAPIError{
			Message: fmt.Sprintf("No answer keyed %q on this record; available answer keys: %s. Nothing was written. To add a new named answer, first add it as a response header in the RFPIO app.", requestedKey, quoteKeys(availableKeys)),
		},
	}
}

// EntryUnderReview is raised when an update targets an entry that has review
// settings switched on and the caller has not opted in.
type EntryUnderReview struct {
	ContentID string
	Reasons   []string
	RFPIOAPIError
}

func NewEntryUnderReview(contentID string, reasons []string) *EntryUnderReview {
	return &EntryUnderReview{
		ContentID: contentID,
		Reasons:   reasons,
		RFPIOAPIError: RFPIOAPIError{
			Message: fmt.Sprintf("Entry %s is under review (%s). Updating it through the API may reset its review settings; that has not been tested. Nothing was written. Ask the user first; if they agree, retry with allow_under_review=true (CLI: --allow-under-review).", contentID, joinStrings(reasons, "; ")),
		},
	}
}

// NewMissingDefaultAnswer is raised when an update names no answer key but the
// record has no "Default" answer and more than one answer to choose from.
func NewMissingDefaultAnswer(answerCount int, availableKeys []string) *AnswerKeyNotFound {
	return &AnswerKeyNotFound{
		RequestedKey:  "Default",
		AvailableKeys: availableKeys,
		RFPIOAPIError: RFPIOAPIError{
			Message: fmt.Sprintf("This record has no \"Default\" answer and has %d answers; pass answer_key to choose one. Available answer keys: %s. Nothing was written. To add a \"Default\" answer, first add it as a response header in the RFPIO app.", answerCount, quoteKeys(availableKeys)),
		},
	}
}

// quoteKeys renders answer keys as a quoted, comma-separated list.
func quoteKeys(keys []string) string {
	if len(keys) == 0 {
		return "(none)"
	}
	quoted := make([]string, len(keys))
	for i, k := range keys {
		quoted[i] = fmt.Sprintf("%q", k)
	}
	return joinStrings(quoted, ", ")
}

// containsStr is a simple string contains check (avoid import cycle).
func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// joinStrings joins strings with a separator.
func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
