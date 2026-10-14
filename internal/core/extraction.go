package core

// getFirst returns the first present, non-None value among keys in mapping.
func getFirst(mapping map[string]any, keys []string) any {
	for _, k := range keys {
		if v, ok := mapping[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

// getFirstDeep searches recursively for the first present, non-None value among keys.
func getFirstDeep(obj any, keys []string) any {
	switch v := obj.(type) {
	case map[string]any:
		for _, k := range keys {
			if val, ok := v[k]; ok && val != nil {
				return val
			}
		}
		for _, val := range v {
			if found := getFirstDeep(val, keys); found != nil {
				return found
			}
		}
	case []any:
		for _, item := range v {
			if found := getFirstDeep(item, keys); found != nil {
				return found
			}
		}
	}
	return nil
}

// extractFirstList returns the first list found in a response envelope.
func extractFirstList(d any) []any {
	switch v := d.(type) {
	case []any:
		return v
	case map[string]any:
		// Check common envelope keys first.
		for _, key := range []string{"items", "results", "data", "content", "docs", "list", "userVOList"} {
			if val, ok := v[key]; ok {
				if list, ok := val.([]any); ok {
					return list
				}
			}
		}
		// Fallback: any list value.
		for _, val := range v {
			if list, ok := val.([]any); ok {
				return list
			}
		}
	}
	return nil
}

// extractAnswerItems recursively collects dicts that contain an 'answers' list.
func extractAnswerItems(obj any) []map[string]any {
	found := make([]map[string]any, 0)
	walk(obj, &found)
	return found
}

func walk(obj any, found *[]map[string]any) {
	switch v := obj.(type) {
	case map[string]any:
		if _, hasAnswers := v["answers"]; hasAnswers {
			*found = append(*found, v)
			return
		}
		for _, val := range v {
			walk(val, found)
		}
	case []any:
		for _, item := range v {
			walk(item, found)
		}
	}
}
