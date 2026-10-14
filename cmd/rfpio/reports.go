package main

import (
	"fmt"
	"strings"
)

func drawSearchReport(items []any, digest bool) string {
	var sb strings.Builder

	if len(items) == 0 {
		sb.WriteString("┏" + strings.Repeat("━", maxWidth-2) + "┓\n")
		sb.WriteString("│ No results " + strings.Repeat(" ", maxWidth-14) + "│\n")
		sb.WriteString("├" + strings.Repeat("─", maxWidth-2) + "┤")
		return sb.String()
	}

	for idx, item := range items {
		it, ok := item.(map[string]any)
		if !ok {
			continue
		}

		sb.WriteString(drawSingleReport(it, digest))

		if idx < len(items)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func drawSingleReport(it map[string]any, digest bool) string {
	var sb strings.Builder
	w := maxWidth

	// Header.
	sb.WriteString("┏" + strings.Repeat("━", w-2) + "┓\n")

	// ID.
	sb.WriteString(kvLine("ID", fmt.Sprintf("%v", it["id"]), 0))

	// Score, Rating, Last Used on one line.
	var trio []string
	if v := it["score"]; v != nil {
		trio = append(trio, fmt.Sprintf("Score: %v", v))
	}
	if v := it["rating"]; v != nil {
		trio = append(trio, fmt.Sprintf("Rating: %v", v))
	}
	if v := it["last_used"]; v != nil {
		trio = append(trio, fmt.Sprintf("Last Used: %v", v))
	}
	if len(trio) > 0 {
		sb.WriteString(kvLine(" ", "   "+strings.Join(trio, "   "), 2))
	}

	// Last Updated.
	if v := it["last_updated"]; v != nil {
		sb.WriteString(kvLine("Last Updated", fmt.Sprintf("%v", v), 0))
	}

	// Question.
	if v := it["question"]; v != nil {
		sb.WriteString(kvLine("Question", fmt.Sprintf("%v", v), 0))
	}

	// Alternate Questions.
	if alts, ok := it["alternate_questions"].([]any); ok && len(alts) > 0 {
		var parts []string
		for _, a := range alts {
			parts = append(parts, fmt.Sprintf("%v", a))
		}
		sb.WriteString(kvLine("Alternate Question", strings.Join(parts, "; "), 0))
	}

	// Tags.
	var tags []string
	if t, ok := it["tags"].([]string); ok {
		tags = t
	} else if t, ok := it["tags"].([]any); ok {
		for _, tag := range t {
			tags = append(tags, fmt.Sprintf("%v", tag))
		}
	}
	if len(tags) > 0 {
		sb.WriteString(kvLine("Tags", strings.Join(tags, ", "), 0))
	}

	// SimHash and Word Count.
	firstHash, firstWc := true, true
	var answerTexts []struct {
		key string
		text string
	}

	if answers, ok := it["answers"].([]any); ok {
		for _, a := range answers {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%v", am["key"])
			value := ""
			if v, ok := am["value"]; ok {
				value = fmt.Sprintf("%v", v)
			}
			text := value
			if sh, ok := am["simhash"]; ok {
				label := "SimHash"
				if !firstHash {
					label = " "
				}
				firstHash = false
				sb.WriteString(kvLine(label, key+": "+fmt.Sprintf("%v", sh), 2))
			}
			if wc, ok := am["word_count"]; ok {
				label := "Word Count"
				if !firstWc {
					label = " "
				}
				firstWc = false
				sb.WriteString(kvLine(label, key+": "+fmt.Sprintf("%v", wc), 2))
			}

			answerTexts = append(answerTexts, struct {
				key  string
				text string
			}{key, text})
		}
	}

	// Answers (body).
	if !digest {
		sb.WriteString("├" + strings.Repeat("─", w-2) + "┤\n")
		sb.WriteString("│ " + strings.Repeat(" ", w-4) + "Answers " + strings.Repeat(" ", w-4-7) + "│\n")
		for _, at := range answerTexts {
			prefix := "- " + at.key + ": "
			avail := w - 2 - 2 - len(prefix)
			wrapped := wrapText(at.text, avail)
			for i, line := range wrapped {
				content := line
				if i == 0 {
					content = prefix + line
				} else {
					content = strings.Repeat(" ", len(prefix)) + line
				}
				sb.WriteString("│  " + leftPad(content, w-4) + " │\n")
			}
		}
		if len(answerTexts) == 0 {
			sb.WriteString("│  (no answers)" + strings.Repeat(" ", w-15) + "│\n")
		}
		sb.WriteString("├" + strings.Repeat("─", w-2) + "┤")
	} else {
		sb.WriteString("├" + strings.Repeat("─", w-2) + "┤")
	}

	return sb.String()
}

func kvLine(label string, value string, indent int) string {
	labelText := label + ": "
	avail := maxWidth - 2 - indent - len(labelText)
	wrapped := wrapText(value, avail)
	var sb strings.Builder
	for i, line := range wrapped {
		prefix := labelText
		if i > 0 {
			prefix = strings.Repeat(" ", len(labelText))
		}
		content := prefix + line
		sb.WriteString("│" + strings.Repeat(" ", indent) + leftPad(content, maxWidth-2-indent) + "│\n")
	}
	if len(wrapped) == 0 {
		sb.WriteString("│" + strings.Repeat(" ", indent) + leftPad(labelText, maxWidth-2-indent) + "│\n")
	}
	return sb.String()
}

func drawTagTable(tags []any) string {
	type colInfo struct {
		name string
		wid  int
	}
	cols := []colInfo{{"Tag", 50}, {"ID", 26}}

	var rows []struct{ tag, id string }
	for _, t := range tags {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		tag := ""
		for _, k := range []string{"tagName", "name", "tag"} {
			if v, ok := m[k]; ok && v != nil {
				tag = fmt.Sprintf("%v", v)
				break
			}
		}
		id := ""
		if v, ok := m["id"]; ok && v != nil {
			id = fmt.Sprintf("%v", v)
		}
		rows = append(rows, struct{ tag, id string }{tag, id})
	}

	// Sort by tag name (case-insensitive).
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if strings.ToLower(rows[i].tag) > strings.ToLower(rows[j].tag) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}

	// Calculate widths.
	colWidths := []int{cols[0].wid, cols[1].wid}
	for _, r := range rows {
		if len(r.tag) > colWidths[0] {
			colWidths[0] = len(r.tag)
		}
		if len(r.id) > colWidths[1] {
			colWidths[1] = len(r.id)
		}
	}

	// Cap to maxWidth.
	contentTotal := colWidths[0] + colWidths[1]
	lineLen := contentTotal + 3
	if lineLen > maxWidth {
		overflow := lineLen - maxWidth
		for i := range colWidths {
			reduction := colWidths[i] - 6
			if reduction < 0 {
				reduction = 0
			}
			colWidths[i] -= reduction
			overflow -= reduction
			if overflow <= 0 {
				break
			}
		}
		contentTotal = colWidths[0] + colWidths[1]
	}

	var sb strings.Builder

	hLine := func(ch string) string {
		if ch == "=" {
			return "┏" + strings.Repeat("━", contentTotal+1) + "┓"
		}
		return "├" + strings.Repeat("─", contentTotal+1) + "┤"
	}

	sb.WriteString(hLine("=") + "\n")
	sb.WriteString("│" + leftPad(cols[0].name, colWidths[0]) + "│" + leftPad(cols[1].name, colWidths[1]) + "│" + "\n")
	sb.WriteString(hLine("-") + "\n")

	for _, r := range rows {
		sb.WriteString("│" + leftPad(r.tag, colWidths[0]) + "│" + leftPad(r.id, colWidths[1]) + "│" + "\n")
		sb.WriteString(hLine("-") + "\n")
	}

	return sb.String()
}

func drawDeleteReport(ids, deleted, notDeleted []string) string {
	w := maxWidth
	var sb strings.Builder

	sb.WriteString("┏" + strings.Repeat("━", w-2) + "┓\n")
	sb.WriteString(kvLine("Action", "Delete Answer(s)", 0))
	sb.WriteString(kvLine("Move To Trash", "Yes", 0))
	sb.WriteString("├" + strings.Repeat("─", w-2) + "┤\n")
	sb.WriteString("│ " + strings.Repeat(" ", w-4) + "IDs " + strings.Repeat(" ", w-4-3) + "│\n")

	for _, id := range ids {
		sb.WriteString("│  - " + leftPad(id, w-6) + " │\n")
	}
	if len(ids) == 0 {
		sb.WriteString("│  (no ids)" + strings.Repeat(" ", w-10) + "│\n")
	}

	sb.WriteString("├" + strings.Repeat("─", w-2) + "┤\n")
	sb.WriteString("│ " + strings.Repeat(" ", w-4) + "Deleted " + strings.Repeat(" ", w-4-7) + "│\n")
	if len(deleted) > 0 {
		for _, id := range deleted {
			sb.WriteString("│  • " + leftPad(id, w-6) + " │\n")
		}
	} else {
		sb.WriteString("│  (none)" + strings.Repeat(" ", w-8) + "│\n")
	}

	sb.WriteString("├" + strings.Repeat("─", w-2) + "┤\n")
	sb.WriteString("│ " + strings.Repeat(" ", w-4) + "Not Deleted " + strings.Repeat(" ", w-4-11) + "│\n")
	if len(notDeleted) > 0 {
		for _, id := range notDeleted {
			sb.WriteString("│  • " + leftPad(id, w-6) + " │\n")
		}
	} else {
		sb.WriteString("│  (none)" + strings.Repeat(" ", w-8) + "│\n")
	}

	sb.WriteString("├" + strings.Repeat("─", w-2) + "┤")

	return sb.String()
}
