package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// normalizeAlternateQuestions normalizes alternate questions to a newline-delimited string.
func normalizeAlternateQuestions(value any) string {
	switch v := value.(type) {
	case []any:
		var parts []string
		for _, q := range v {
			s := strings.TrimSpace(fmt.Sprintf("%v", q))
			if s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	case string:
		text := brRe.ReplaceAllString(v, "\n")
		var parts []string
		for _, p := range strings.Split(text, "\n") {
			if strings.TrimSpace(p) != "" {
				parts = append(parts, strings.TrimSpace(p))
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// normalizeTags normalizes tags to a list of tag-name strings.
func normalizeTags(value any) []string {
	switch v := value.(type) {
	case []any:
		var items []string
		for _, t := range v {
			s := strings.TrimSpace(fmt.Sprintf("%v", t))
			if s != "" {
				items = append(items, s)
			}
		}
		return items
	case string:
		var items []string
		for _, t := range strings.Split(v, ",") {
			s := strings.TrimSpace(t)
			if s != "" {
				items = append(items, s)
			}
		}
		return items
	default:
		return nil
	}
}

// DefaultAnswerKey is the answer key targeted when an update names no key.
const DefaultAnswerKey = "Default"

// AnswerUpdate is one answer value to write. An empty Key targets the
// "Default" answer (or the record's only answer, if it has exactly one).
type AnswerUpdate struct {
	Key   string
	Value string
}

var (
	htmlTagRe    = regexp.MustCompile(`<[a-zA-Z/!][^>]*>`)
	htmlEntityRe = regexp.MustCompile(`&(#[0-9]+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);`)
	plainEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
)

// answerKeys returns the keys of the answer entries, in record order.
func answerKeys(answers []any) []string {
	keys := make([]string, 0, len(answers))
	for _, e := range answers {
		if m, ok := e.(map[string]any); ok {
			keys = append(keys, fmt.Sprintf("%v", m["key"]))
		}
	}
	return keys
}

// findAnswerIndex locates the answer entry keyed key: an exact match wins,
// otherwise a unique case-insensitive match. Returns -1 when nothing matches.
// Keys are not unique within an entry (imported entries can repeat one, e.g.
// "DESCRIPTION" on every column), so a key that matches several answers is an
// error rather than a silent pick of the first.
func findAnswerIndex(answers []any, key string) (int, error) {
	var exact []int
	folded := -1
	foldedCount := 0
	for i, e := range answers {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		k := fmt.Sprintf("%v", m["key"])
		if k == key {
			exact = append(exact, i)
		} else if strings.EqualFold(k, key) {
			if folded < 0 {
				folded = i
			}
			foldedCount++
		}
	}
	if len(exact) > 1 {
		return -1, &RFPIOAPIError{Message: fmt.Sprintf(
			"Answer key %q appears %d times on this record (answers %s, counting from 1), so it cannot be told apart by key. Edit it in the RFPIO app. Nothing was written.",
			key, len(exact), positionList(exact))}
	}
	if len(exact) == 1 {
		return exact[0], nil
	}
	if foldedCount > 1 {
		return -1, &RFPIOAPIError{Message: fmt.Sprintf(
			"Answer key %q matches more than one answer case-insensitively; use the exact key. Available answer keys: %s. Nothing was written.",
			key, quoteKeys(answerKeys(answers)))}
	}
	return folded, nil
}

// positionList renders 0-based answer indexes as 1-based positions.
func positionList(indexes []int) string {
	parts := make([]string, len(indexes))
	for i, n := range indexes {
		parts[i] = strconv.Itoa(n + 1)
	}
	return strings.Join(parts, ", ")
}

// looksLikeHTML reports whether s already carries markup or entity references.
func looksLikeHTML(s string) bool {
	return htmlTagRe.MatchString(s) || htmlEntityRe.MatchString(s)
}

// formatAnswerValue shapes a new value to match how RFPIO stores the target
// answer type. RICH_TEXT answers are stored as HTML wrapped in <div>, so plain
// text is escaped, newlines become <br>, and the result is wrapped. Input that
// already contains HTML or entities, and every other answer type (e.g. SINGLE),
// is sent unchanged.
func formatAnswerValue(value string, answerType any) string {
	if fmt.Sprintf("%v", answerType) != "RICH_TEXT" || looksLikeHTML(value) {
		return value
	}
	text := strings.ReplaceAll(value, "\r\n", "\n")
	text = plainEscaper.Replace(text)
	text = strings.ReplaceAll(text, "\n", "<br>")
	return "<div>" + text + "</div>"
}

// mergeAnswers returns a copy of existingAnswers with each update's value
// written into the matching existing entry. It never appends an entry: an
// unknown key fails with AnswerKeyNotFound listing the record's real keys.
// Every other entry, every other field of a matched entry, and the order of
// the list are preserved. The input list is not mutated.
func mergeAnswers(existingAnswers []any, updates []AnswerUpdate) ([]any, error) {
	keys := answerKeys(existingAnswers)
	targets := make(map[int]string, len(updates))
	values := make(map[int]string, len(updates))

	for _, u := range updates {
		requested := strings.TrimSpace(u.Key)
		var idx int
		if requested == "" {
			found, err := findAnswerIndex(existingAnswers, DefaultAnswerKey)
			if err != nil {
				return nil, err
			}
			idx = found
			if idx < 0 && len(existingAnswers) == 1 {
				// Single-answer entry with a non-Default key (e.g. "Response"):
				// the one answer is unambiguous.
				if _, ok := existingAnswers[0].(map[string]any); ok {
					idx = 0
				}
			}
			if idx < 0 {
				return nil, NewMissingDefaultAnswer(len(existingAnswers), keys)
			}
		} else {
			found, err := findAnswerIndex(existingAnswers, requested)
			if err != nil {
				return nil, err
			}
			if found < 0 {
				return nil, NewAnswerKeyNotFound(requested, keys)
			}
			idx = found
		}

		if prev, dup := targets[idx]; dup {
			return nil, &RFPIOAPIError{Message: fmt.Sprintf(
				"Answer keys %q and %q both target the answer keyed %q; supply each answer once. Nothing was written.",
				prev, u.Key, keys[idx])}
		}
		targets[idx] = u.Key
		values[idx] = u.Value
	}

	result := make([]any, 0, len(existingAnswers))
	for i, entry := range existingAnswers {
		m, ok := entry.(map[string]any)
		if !ok {
			result = append(result, entry)
			continue
		}
		entryCopy := make(map[string]any, len(m))
		for k, v := range m {
			entryCopy[k] = v
		}
		if value, ok := values[i]; ok {
			entryCopy["value"] = formatAnswerValue(value, m["type"])
		}
		result = append(result, entryCopy)
	}
	return result, nil
}
