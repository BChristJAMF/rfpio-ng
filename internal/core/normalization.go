package core

import (
	"fmt"
	"regexp"
	"strings"
)

// Score/rating key groups for field-name fallbacks.
var (
	scoreKeys       = []string{"contentScore", "score", "content_score"}
	scoreKeysDeep   = []string{"contentScore", "score", "content_score", "relevanceScore", "relevance"}
	ratingKeys      = []string{"contentRating", "starRating", "rating", "star_rating"}
	backfillKeys    = []string{
		"contentScore", "score", "content_score",
		"contentRating", "starRating", "rating", "star_rating",
		"lastUsedDate", "updateDate",
	}
)

// brReplace splits on <br> tags for alternate question handling.
func brReplace(text string) string {
	return regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(text, "\n")
}

// TagName extracts the tag name from a tag item.
func tagName(tag any) string {
	switch v := tag.(type) {
	case map[string]any:
		if name, ok := v["tagName"]; ok {
			return fmt.Sprintf("%v", name)
		}
	}
	return ""
}

// resolveTagNames maps caller-supplied tag names to the exact stored tagName values.
// tagVocab is the complete tag list (limit=0).
func resolveTagNames(cfg *Config, names []string, tagVocab []map[string]any) ([]string, []string) {
	// Build case-insensitive index.
	byLower := make(map[string]string)
	for _, tag := range tagVocab {
		if name, ok := tag["tagName"]; ok {
			if nameStr := fmt.Sprintf("%v", name); nameStr != "" {
				lower := strings.ToLower(nameStr)
				if _, exists := byLower[lower]; !exists {
					byLower[lower] = nameStr
				}
			}
		}
	}

	resolved := make([]string, 0, len(names))
	unknown := make([]string, 0)

	for _, name := range names {
		match, exists := byLower[strings.ToLower(name)]
		if exists {
			resolved = append(resolved, match)
		} else {
			unknown = append(unknown, name)
		}
	}

	return resolved, unknown
}

// normalizeItem returns a search hit reshaped to the documented, normalized contract.
func normalizeItem(item any, dropBody bool) any {
	it, ok := item.(map[string]any)
	if !ok {
		return item
	}

	score := getFirst(it, scoreKeys)
	if score == nil {
		score = getFirstDeep(it, scoreKeysDeep)
	}
	rating := getFirst(it, ratingKeys)
	if rating == nil {
		rating = getFirstDeep(it, ratingKeys)
	}

	// Alternate questions.
	altRaw := it["alternateQuestions"]
	var alternateQuestions []string
	switch v := altRaw.(type) {
	case []any:
		for _, q := range v {
			if s := fmt.Sprintf("%v", q); strings.TrimSpace(s) != "" {
				alternateQuestions = append(alternateQuestions, strings.TrimSpace(s))
			}
		}
	case string:
		parts := brRe.ReplaceAllString(v, "\n")
		for _, p := range strings.Split(parts, "\n") {
			if strings.TrimSpace(p) != "" {
				alternateQuestions = append(alternateQuestions, strings.TrimSpace(p))
			}
		}
	}

	// Tags.
	tagsRaw := it["tags"]
	var tags []string
	switch v := tagsRaw.(type) {
	case []any:
		for _, t := range v {
			if s := fmt.Sprintf("%v", t); s != "" {
				tags = append(tags, s)
			}
		}
	case string:
		if v != "" {
			tags = append(tags, v)
		}
	}

	// Answers.
	answersOut := make([]any, 0)
	answersRaw := it["answers"]
	if answers, ok := answersRaw.([]any); ok {
		for _, a := range answers {
			var key, value, answerType string
			switch av := a.(type) {
			case map[string]any:
				if k, ok := av["key"]; ok {
					key = fmt.Sprintf("%v", k)
				}
				if t, ok := av["type"].(string); ok {
					answerType = t
				}
				if v, ok := av["value"]; ok {
					value = fmt.Sprintf("%v", v)
				}
			default:
				value = fmt.Sprintf("%v", a)
			}

			text := htmlToText(value)
			entry := map[string]any{
				"key":        key,
				"simhash":    simhash64(text),
				"word_count": wordCount(text),
			}
			if answerType != "" {
				entry["type"] = answerType
			}
			if !dropBody {
				entry["value"] = text
			}
			answersOut = append(answersOut, entry)
		}
	}

	normalized := map[string]any{
		"id": it["id"],
	}
	if score != nil {
		normalized["score"] = score
	}
	if rating != nil {
		normalized["rating"] = rating
	}
	if v, ok := it["lastUsedDate"]; ok && v != nil {
		normalized["last_used"] = v
	}
	if v, ok := it["updateDate"]; ok && v != nil {
		normalized["last_updated"] = v
	}

	// Provenance fields.
	if v, ok := it["projectName"]; ok && v != nil {
		normalized["project_name"] = v
	}
	if v, ok := it["project_name"]; ok && v != nil {
		normalized["project_name"] = v
	}
	if v, ok := it["sectionName"]; ok && v != nil {
		normalized["section_name"] = v
	}
	if v, ok := it["section_name"]; ok && v != nil {
		normalized["section_name"] = v
	}
	if v, ok := it["contentUrl"]; ok && v != nil {
		normalized["content_url"] = v
	}
	if v, ok := it["content_url"]; ok && v != nil {
		normalized["content_url"] = v
	}

	// Question.
	if v, ok := it["question"]; ok && v != nil {
		normalized["question"] = v
	}
	if v, ok := it["Question"]; ok && v != nil {
		normalized["question"] = v
	}
	if v, ok := it["primaryQuestion"]; ok && v != nil {
		normalized["question"] = v
	}

	normalized["alternate_questions"] = alternateQuestions
	normalized["tags"] = tags
	normalized["answers"] = answersOut

	return normalized
}

// backfillItemDetails fills in score/rating/dates from the full record.
func backfillItemDetails(cfg *Config, item map[string]any) {
	needScore := getFirst(item, scoreKeys) == nil && getFirstDeep(item, scoreKeys) == nil
	needRating := getFirst(item, ratingKeys) == nil && getFirstDeep(item, ratingKeys) == nil

	if !needScore && !needRating {
		return
	}

	ident, ok := item["id"]
	if !ok || ident == nil {
		return
	}

	idStr := fmt.Sprintf("%v", ident)
	fullRecord, err := getRecord(cfg, idStr)
	if err != nil {
		return
	}

	full, ok := fullRecord.(map[string]any)
	if !ok {
		return
	}

	for _, k := range backfillKeys {
		if v, exists := full[k]; exists && v != nil {
			if _, present := item[k]; !present {
				item[k] = v
			}
		}
	}
}
