package core

import (
	"fmt"
	"strings"
)

// APIResult is the common return type for API functions.
type APIResult map[string]any

// APIQueryUsersResult is the result of querying users.
type APIQueryUsersResult struct {
	Users []any `json:"users"`
}

// APISearchResult is the result of searching content.
type APISearchResult struct {
	Items      []any  `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	Omitted    []string `json:"omitted,omitempty"`
}

// APIAddResult is the result of adding content.
type APIAddResult struct {
	Success bool   `json:"success"`
	ID      *string `json:"id,omitempty"`
	Message string `json:"message"`
}

// APIUpdateResult is the result of updating content.
type APIUpdateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// APIDeleteResult is the result of deleting content.
type APIDeleteResult struct {
	Success     bool     `json:"success"`
	Deleted     []string `json:"deleted"`
	NotDeleted  []string `json:"not_deleted"`
	Message     string   `json:"message"`
}

// APITagResult is the result of tag operations.
type APITagResult struct {
	Success bool   `json:"success"`
	ID      *string `json:"id,omitempty"`
	Message string `json:"message"`
}

// APITagListResult is the result of listing tags.
type APITagListResult struct {
	Tags      []any `json:"tags"`
	Total     int   `json:"total"`
	Truncated bool  `json:"truncated"`
}

// APITagUsageResult is the result of counting tag usage.
type APITagUsageResult struct {
	TagName string `json:"tag_name"`
	Count   int    `json:"count"`
}

// APIListAnswersByTagResult is the result of listing answers by tag.
type APIListAnswersByTagResult struct {
	TagName   string   `json:"tag_name"`
	ContentIDs []string `json:"content_ids"`
	Count     int      `json:"count"`
}

// resolveEndpoint resolves a configured endpoint path to a full path string.
func resolveEndpoint(cfg *Config, configured, defaultPath string) string {
	path := configured
	if path == "" {
		path = defaultPath
	}
	return strings.TrimLeft(path, "/")
}

// answerGetEndpoint builds the GET-by-ID endpoint.
func answerGetEndpoint(cfg *Config, contentID string) string {
	getPath := resolveEndpoint(cfg, cfg.AnswerGetPath, DefaultAnswerGetPath)
	if strings.Contains(getPath, "{id}") {
		return strings.Replace(getPath, "{id}", contentID, 1)
	}
	return getPath + "/" + contentID
}

// answerUpdateEndpoint builds the update endpoint.
func answerUpdateEndpoint(cfg *Config, contentID string) string {
	updatePath := resolveEndpoint(cfg, cfg.AnswerUpdatePath, DefaultAnswerUpdatePath)
	if strings.Contains(updatePath, "{id}") {
		return strings.Replace(updatePath, "{id}", contentID, 1)
	}
	return updatePath + "/" + contentID
}

// answerAddEndpoint builds the add endpoint.
func answerAddEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.AnswerAddPath, DefaultAnswerAddPath)
}

// answerDeleteEndpoint builds the delete endpoint.
func answerDeleteEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.AnswerDeletePath, DefaultAnswerDeletePath)
}

// searchEndpoint builds the search endpoint.
func searchEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.SearchPath, DefaultSearchPath)
}

// usersEndpoint builds the users endpoint.
func usersEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.UsersPath, DefaultUsersPath)
}

// tagsListEndpoint builds the tags list endpoint.
func tagsListEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.TagsListPath, DefaultTagsListPath)
}

// tagsCreateEndpoint builds the tags create endpoint.
func tagsCreateEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.TagsCreatePath, DefaultTagsCreatePath)
}

// tagsDeleteEndpoint builds the tags delete endpoint.
func tagsDeleteEndpoint(cfg *Config) string {
	return resolveEndpoint(cfg, cfg.TagsDeletePath, DefaultTagsDeletePath)
}

// tagsUpdateEndpoint builds the tags update endpoint.
func tagsUpdateEndpoint(cfg *Config, tagID string) string {
	updatePath := resolveEndpoint(cfg, cfg.TagsUpdatePath, DefaultTagsUpdatePath)
	if strings.Contains(updatePath, "{id}") {
		return strings.Replace(updatePath, "{id}", tagID, 1)
	}
	return updatePath + "/" + tagID
}

// getRecord fetches the complete existing record for a content ID.
func getRecord(cfg *Config, contentID string) (any, error) {
	endpoint := answerGetEndpoint(cfg, contentID)
	responseText, err := httpGet(cfg, endpoint)
	if err != nil {
		return nil, err
	}

	responseData, err := parseResponse(responseText)
	if err != nil {
		return nil, err
	}

	// Extract the record.
	extracted := extractAnswerItems(responseData)
	if len(extracted) > 0 {
		return extracted[0], nil
	}
	if m, ok := responseData.(map[string]any); ok {
		return m, nil
	}
	return nil, &RFPIOAPIError{Message: fmt.Sprintf("Unexpected get-by-id response shape for %s", contentID)}
}

// postSearch posts to the search endpoint, handling INVALID_TAGS specially.
func postSearch(cfg *Config, endpoint string, payload map[string]any) (any, error) {
	responseText, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		if IsInvalidTagsError(err) {
			tagSearch, _ := payload["tagSearch"].([]string)
			if len(tagSearch) == 1 {
				// Single resolved tag with zero members -> empty envelope.
				return map[string]any{"results": []any{}, "totalRecords": 0}, nil
			}
		}
		return nil, err
	}
	return parseResponse(responseText)
}

// ---------------------------------------------------------------------------
// Public API functions
// ---------------------------------------------------------------------------

// APIQueryUsers queries RFPIO users.
func APIQueryUsers(cfg *Config, limit int) (*APIQueryUsersResult, error) {
	endpoint := usersEndpoint(cfg)
	responseText, err := httpGet(cfg, endpoint)
	if err != nil {
		return nil, err
	}

	responseData, err := parseResponse(responseText)
	if err != nil {
		return nil, err
	}

	// Extract users list.
	users := extractUsers(responseData)
	if users == nil {
		users = []any{}
	}

	// Apply limit.
	if limit > 0 && limit < len(users) {
		users = users[:limit]
	}

	return &APIQueryUsersResult{Users: users}, nil
}

func extractUsers(data any) []any {
	switch v := data.(type) {
	case []any:
		return v
	case map[string]any:
		// Try common envelope keys.
		for _, key := range []string{"users", "data", "items", "results", "userVOList"} {
			if val, ok := v[key]; ok {
				if list, ok := val.([]any); ok {
					return list
				}
			}
		}
		// Check if it's a single user dict.
		if _, hasEmail := v["email"]; hasEmail {
			return []any{v}
		}
	}
	return nil
}

// APISearch searches RFPIO content library.
func APISearch(cfg *Config, keyword *string, ids []string, tags []string, digest bool, limit int, cursor string) (*APISearchResult, error) {
	// ID lookup takes precedence.
	if len(ids) > 0 {
		return searchByID(cfg, ids, digest)
	}

	// Keyword/tag search.
	endpoint := searchEndpoint(cfg)
	payload := map[string]any{
		"limit": clamp(limit, 0, 100),
	}

	if keyword != nil && *keyword != "" {
		payload["keyword"] = *keyword
	}
	if len(tags) > 0 {
		// Resolve tag names.
		// First fetch the full tag list.
		tagList, err := APITagList(cfg, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve tag names: %w", err)
		}
		tagVocab := make([]map[string]any, 0)
		for _, t := range tagList.Tags {
			if m, ok := t.(map[string]any); ok {
				tagVocab = append(tagVocab, m)
			}
		}
		resolved, unknown := resolveTagNames(cfg, tags, tagVocab)
		if len(unknown) > 0 {
			return nil, fmt.Errorf("unknown tag name(s): %s", strings.Join(unknown, ", "))
		}
		payload["tagSearch"] = resolved
	}
	if cursor != "" {
		payload["cursor"] = cursor
	}

	responseData, err := postSearch(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	// Extract items.
	rawItems := extractAnswerItems(responseData)
	if len(rawItems) == 0 {
		if list := extractFirstList(responseData); list != nil {
			rawItems = listToMaps(list)
		}
	}

	// Back-fill missing details.
	for i := range rawItems {
		backfillItemDetails(cfg, rawItems[i])
	}

	// Normalize items.
	items := make([]any, 0, len(rawItems))
	for _, item := range rawItems {
		items = append(items, normalizeItem(item, digest))
	}

	result := &APISearchResult{Items: items}

	// Surface cursor for paging.
	if data, ok := responseData.(map[string]any); ok {
		if cursorMark, ok := data["nextCursorMark"]; ok && cursorMark != nil {
			if cm, ok := cursorMark.(string); ok && cm != "" {
				result.NextCursor = cm
			}
		}
	}

	return result, nil}

func listToMaps(list []any) []map[string]any {
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// searchByID fetches exact records by ID.
func searchByID(cfg *Config, ids []string, digest bool) (*APISearchResult, error) {
	items := make([]any, 0, len(ids))
	omitted := make([]string, 0)

	for _, contentID := range ids {
		endpoint := answerGetEndpoint(cfg, contentID)
		responseText, err := httpGet(cfg, endpoint)
		if err != nil {
			// Non-existent ID: omit it, don't fail.
			omitted = append(omitted, contentID)
			continue
		}

		responseData, err := parseResponse(responseText)
		if err != nil {
			omitted = append(omitted, contentID)
			continue
		}

		// Extract items.
		extracted := extractAnswerItems(responseData)
		if len(extracted) > 0 {
			for _, item := range extracted {
				items = append(items, normalizeItem(item, digest))
			}
		} else if list := extractFirstList(responseData); list != nil {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					items = append(items, normalizeItem(m, digest))
				}
			}
		} else if m, ok := responseData.(map[string]any); ok {
			items = append(items, normalizeItem(m, digest))
		} else {
			omitted = append(omitted, contentID)
		}
	}

	result := &APISearchResult{Items: items}
	if len(omitted) > 0 {
		result.Omitted = omitted
	}

	return result, nil
}

// APIAdd adds a new Q/A pair.
func APIAdd(cfg *Config, question, answer string, alternateQuestions *string, tags *[]string) (*APIAddResult, error) {
	endpoint := answerAddEndpoint(cfg)

	payload := map[string]any{
		"question": question,
		"answers":  []any{map[string]any{"key": "Default", "value": answer}},
	}

	if alternateQuestions != nil && *alternateQuestions != "" {
		payload["alternateQuestions"] = *alternateQuestions
	}
	if tags != nil && len(*tags) > 0 {
		payload["tags"] = *tags
	}

	responseText, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		return &APIAddResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	responseData, err := parseResponse(responseText)
	if err != nil {
		return &APIAddResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Extract ID.
	var contentID string
	if m, ok := responseData.(map[string]any); ok {
		if id, ok := m["id"]; ok {
			contentID = fmt.Sprintf("%v", id)
		} else if cid, ok := m["contentId"]; ok {
			contentID = fmt.Sprintf("%v", cid)
		}
	}

	return &APIAddResult{
		Success: true,
		ID:      &contentID,
		Message: fmt.Sprintf("Successfully added content (ID: %s)", contentID),
	}, nil
}

// serverComputedFields are read-only record fields the server computes itself.
// An answer write that carried the old answerMatch (a plain-text copy of the
// answers) and the old contentScore came back with both recomputed, so the
// values read before an edit are stale after it. They are left out of answer
// writes rather than echoed back.
var serverComputedFields = []string{"answerMatch", "contentScore"}

// checkAnswerSlots verifies that an update keeps the record's answer slots
// exactly: RFPIO maps answers to the entry's answer columns by position, so a
// changed count or order would write values into the wrong column (or fail
// with "Index N out of bounds").
func checkAnswerSlots(before, after []any) error {
	beforeKeys, afterKeys := answerKeys(before), answerKeys(after)
	if len(before) != len(after) || strings.Join(beforeKeys, "\x00") != strings.Join(afterKeys, "\x00") {
		return &RFPIOAPIError{Message: fmt.Sprintf(
			"Internal check failed: update would change the answer slots from %s to %s. Nothing was written.",
			quoteKeys(beforeKeys), quoteKeys(afterKeys))}
	}
	return nil
}

// APIUpdate updates an existing Q/A pair using read-modify-write.
//
// The update endpoint replaces the stored record, so this GETs the full
// record, overlays only the supplied fields, and POSTs the whole object back;
// everything not supplied (other answers, customFields, owners, starRating,
// reviewStatus, projectName, sectionName, ...) is preserved. Each entry in
// answers replaces the value of an existing answer (see mergeAnswers); no
// answer slot is ever added, and serverComputedFields are left for the server
// to recompute. Merge errors are returned before anything is written.
//
// Entries with review settings switched on (see underReviewReasons) are
// refused unless allowUnderReview is set, because it is not known that a
// replace-style update preserves those settings.
func APIUpdate(cfg *Config, contentID string, question *string, answers []AnswerUpdate, alternateQuestions, tags *string, allowUnderReview bool) (*APIUpdateResult, error) {
	// Validate inputs.
	if question == nil && len(answers) == 0 && alternateQuestions == nil && tags == nil {
		return &APIUpdateResult{
			Success: false,
			Message: "No updates provided (question, answers, alternate_questions, tags all empty)",
		}, nil
	}

	// GET the existing record.
	existingRecord, err := getRecord(cfg, contentID)
	if err != nil {
		return nil, err
	}

	record, ok := existingRecord.(map[string]any)
	if !ok {
		return nil, &RFPIOAPIError{Message: fmt.Sprintf("Unexpected record shape for %s", contentID)}
	}

	if !allowUnderReview {
		if reasons := underReviewReasons(record); len(reasons) > 0 {
			return nil, NewEntryUnderReview(contentID, reasons)
		}
	}

	// Overlay changes.
	if question != nil {
		record["question"] = *question
	}

	if len(answers) > 0 {
		existingAnswers, _ := record["answers"].([]any)
		newAnswers, err := mergeAnswers(existingAnswers, answers)
		if err != nil {
			return nil, err
		}
		if err := checkAnswerSlots(existingAnswers, newAnswers); err != nil {
			return nil, err
		}
		record["answers"] = newAnswers
		for _, field := range serverComputedFields {
			delete(record, field)
		}
	}

	if alternateQuestions != nil && *alternateQuestions != "" {
		record["alternateQuestions"] = normalizeAlternateQuestions(*alternateQuestions)
	}

	if tags != nil {
		// The API stores tags as a list of names, not a comma-separated string.
		if names := normalizeTags(*tags); len(names) > 0 {
			record["tags"] = names
		}
	}

	// POST the full record back.
	endpoint := answerUpdateEndpoint(cfg, contentID)
	_, err = httpPostJSON(cfg, endpoint, record)
	if err != nil {
		return nil, err
	}

	return &APIUpdateResult{
		Success: true,
		Message: fmt.Sprintf("Successfully updated content (ID: %s)", contentID),
	}, nil
}

// APIDelete deletes content by ID(s).
func APIDelete(cfg *Config, ids []string, moveToTrash bool) (*APIDeleteResult, error) {
	endpoint := answerDeleteEndpoint(cfg)

	payload := map[string]any{
		"idList":        ids,
		"isMoveToTrash": moveToTrash,
	}

	responseText, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	// Parse response to find deleted records.
	responseData, _ := parseResponse(responseText)

	deleted := collectDeletedIDs(responseData, ids)
	deletedSet := make(map[string]bool)
	for _, id := range deleted {
		deletedSet[id] = true
	}

	notDeleted := make([]string, 0)
	for _, id := range ids {
		if !deletedSet[id] {
			notDeleted = append(notDeleted, id)
		}
	}

	success := len(deleted) > 0
	msg := fmt.Sprintf("Deleted %d of %d items", len(deleted), len(ids))
	if !success {
		msg = "No items were deleted"
	}

	return &APIDeleteResult{
		Success:    success,
		Deleted:    deleted,
		NotDeleted: notDeleted,
		Message:    msg,
	}, nil
}

func collectDeletedIDs(data any, requested []string) []string {
	deleted := make([]string, 0)
	collectDeleted(data, &deleted)
	return deleted
}

func collectDeleted(data any, deleted *[]string) {
	switch v := data.(type) {
	case map[string]any:
		for k, val := range v {
			lower := strings.ToLower(k)
			if strings.Contains(lower, "deleted") && strings.Contains(lower, "record") {
				if list, ok := val.([]any); ok {
					for _, item := range list {
						switch el := item.(type) {
						case string:
							*deleted = append(*deleted, el)
						case map[string]any:
							for _, idKey := range []string{"id", "ID", "Id"} {
								if idVal, ok := el[idKey]; ok && idVal != nil {
									*deleted = append(*deleted, fmt.Sprintf("%v", idVal))
								}
							}
						}
					}
				}
			}
			// Recurse into nested structures.
			if _, isPrimitive := val.(string); !isPrimitive {
				if _, isFloat := val.(float64); !isFloat {
					if _, isBool := val.(bool); !isBool {
						collectDeleted(val, deleted)
					}
				}
			}
		}
	case []any:
		for _, item := range v {
			collectDeleted(item, deleted)
		}
	}
}

// APITagList lists tags in the content library.
func APITagList(cfg *Config, limit int) (*APITagListResult, error) {
	endpoint := tagsListEndpoint(cfg)

	// POST-based pagination.
	pageSize := 100
	offset := 0
	allTags := make([]any, 0)
	total := 0
	allBodies := make([]string, 0)

	for {
		payload := map[string]any{
			"limit":       pageSize,
			"offset":      offset,
			"primaryBU":   []any{},
			"searchTerm":  "",
		}

		responseText, err := httpPostJSON(cfg, endpoint, payload)
		if err != nil {
			return nil, err
		}
		allBodies = append(allBodies, responseText)

		responseData, err := parseResponse(responseText)
		if err != nil {
			return nil, err
		}

		// Extract tag list.
		var pageTags []any
		if m, ok := responseData.(map[string]any); ok {
			if tagList, ok := m["tagList"]; ok {
				if list, ok := tagList.([]any); ok {
					pageTags = list
				}
			} else {
				list := extractFirstList(responseData)
				if list != nil {
					pageTags = list
				}
			}
			if total == 0 {
				if tr, ok := m["totalRecords"]; ok {
					if t, ok := tr.(float64); ok {
						total = int(t)
					}
				}
			}
		} else if list, ok := responseData.([]any); ok {
			pageTags = list
		}

		allTags = append(allTags, pageTags...)

		// Stop conditions.
		if len(pageTags) == 0 || len(pageTags) < pageSize {
			break
		}
		if total > 0 && len(allTags) >= total {
			break
		}
		offset += len(pageTags)
	}

	// Apply limit.
	totalMatched := len(allTags)
	if limit > 0 && limit < len(allTags) {
		allTags = allTags[:limit]
	}

	// Convert to maps.
	tags := make([]any, 0, len(allTags))
	for _, t := range allTags {
		if m, ok := t.(map[string]any); ok {
			tags = append(tags, m)
		}
	}

	return &APITagListResult{
		Tags:      tags,
		Total:     totalMatched,
		Truncated: limit > 0 && totalMatched > limit,
	}, nil
}

// APITagCreate creates a new tag.
func APITagCreate(cfg *Config, name string) (*APITagResult, error) {
	endpoint := tagsCreateEndpoint(cfg)

	payload := map[string]any{
		"tagName": name,
	}

	responseText, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	responseData, err := parseResponse(responseText)
	if err != nil {
		return nil, err
	}

	var tagID string
	if m, ok := responseData.(map[string]any); ok {
		if id, ok := m["id"]; ok {
			tagID = fmt.Sprintf("%v", id)
		}
	}

	return &APITagResult{
		Success: true,
		ID:      &tagID,
		Message: fmt.Sprintf("Successfully created tag %q (ID: %s)", name, tagID),
	}, nil
}

// APITagDelete deletes a tag by ID.
func APITagDelete(cfg *Config, tagID string) (*APITagResult, error) {
	endpoint := tagsDeleteEndpoint(cfg)

	payload := map[string]any{
		"id": tagID,
	}

	_, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	return &APITagResult{
		Success: true,
		Message: fmt.Sprintf("Successfully deleted tag (ID: %s)", tagID),
	}, nil
}

// APITagUpdate updates a tag's name.
func APITagUpdate(cfg *Config, tagID, newName string) (*APITagResult, error) {
	endpoint := tagsUpdateEndpoint(cfg, tagID)

	payload := map[string]any{
		"id":      tagID,
		"tagName": newName,
	}

	_, err := httpPostJSON(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	return &APITagResult{
		Success: true,
		Message: fmt.Sprintf("Successfully updated tag (ID: %s) to %q", tagID, newName),
	}, nil
}

// APITagUsageCount counts entries with a given tag.
func APITagUsageCount(cfg *Config, name string) (*APITagUsageResult, error) {
	// Fetch the full tag list to resolve the name.
	tagList, err := APITagList(cfg, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tag name: %w", err)
	}

	// Find the canonical tag name.
	canonical := ""
	for _, t := range tagList.Tags {
		if m, ok := t.(map[string]any); ok {
			if tn, ok := m["tagName"]; ok {
				if tnStr := fmt.Sprintf("%v", tn); strings.EqualFold(tnStr, name) {
					canonical = tnStr
					break
				}
			}
		}
	}

	if canonical == "" {
		return nil, fmt.Errorf("unknown tag name: %q", name)
	}

	// Search with limit=1 to get totalRecords.
	endpoint := searchEndpoint(cfg)
	payload := map[string]any{
		"tagSearch": []string{canonical},
		"limit":     1,
	}

	responseData, err := postSearch(cfg, endpoint, payload)
	if err != nil {
		return nil, err
	}

	var total int
	if data, ok := responseData.(map[string]any); ok {
		if tr, ok := data["totalRecords"]; ok {
			if t, ok := tr.(float64); ok {
				total = int(t)
			}
		}
	}

	return &APITagUsageResult{
		TagName: canonical,
		Count:   total,
	}, nil
}

// APIListAnswersByTag enumerates all content IDs for a given tag.
func APIListAnswersByTag(cfg *Config, name string) (*APIListAnswersByTagResult, error) {
	// Fetch the full tag list to resolve the name.
	tagList, err := APITagList(cfg, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tag name: %w", err)
	}

	// Find the canonical tag name.
	canonical := ""
	for _, t := range tagList.Tags {
		if m, ok := t.(map[string]any); ok {
			if tn, ok := m["tagName"]; ok {
				if tnStr := fmt.Sprintf("%v", tn); strings.EqualFold(tnStr, name) {
					canonical = tnStr
					break
				}
			}
		}
	}

	if canonical == "" {
		return nil, fmt.Errorf("unknown tag name: %q", name)
	}

	// Cursor-walk the tag membership.
	endpoint := searchEndpoint(cfg)
	cursor := ""
	contentIDs := make([]string, 0)
	pages := 0

	for {
		payload := map[string]any{
			"tagSearch": []string{canonical},
			"limit":     100,
		}
		if cursor != "" {
			payload["cursor"] = cursor
		}

		responseData, err := postSearch(cfg, endpoint, payload)
		if err != nil {
			break
		}
		pages++

		items := extractAnswerItems(responseData)
		if len(items) == 0 {
			if list := extractFirstList(responseData); list != nil {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						if id, ok := m["id"]; ok {
							contentIDs = append(contentIDs, fmt.Sprintf("%v", id))
						}
					}
				}
			}
		} else {
			for _, item := range items {
				if id, ok := item["id"]; ok {
					contentIDs = append(contentIDs, fmt.Sprintf("%v", id))
				}
			}
		}

		// Check termination.
		nextCursor := ""
		if data, ok := responseData.(map[string]any); ok {
			if nc, ok := data["nextCursorMark"]; ok {
				nextCursor, _ = nc.(string)
			}
		}
		if len(contentIDs) == pages*100 || nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
		if pages >= 1000 {
			break
		}
	}

	return &APIListAnswersByTagResult{
		TagName:    canonical,
		ContentIDs: contentIDs,
		Count:      len(contentIDs),
	}, nil
}
