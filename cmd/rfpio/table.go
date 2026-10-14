package main

import (
	"fmt"
	"strings"
)

const maxWidth = 80

func wrapText(text string, width int) []string {
	if text == "" {
		return []string{""}
	}
	if width <= 0 {
		width = maxWidth
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		if len(lines[len(lines)-1])+1+len(word) <= width {
			lines[len(lines)-1] += " " + word
		} else {
			lines = append(lines, word)
		}
	}
	return lines
}

func drawUserTable(users []any) string {
	type colInfo struct {
		name string
		wid  int
	}
	cols := []colInfo{{"Name", 30}, {"Email", 38}, {"Active", 8}}

	// Collect rows.
	var rows []sortRow
	for _, u := range users {
		m, ok := u.(map[string]any)
		if !ok {
			continue
		}

		// Name: firstName + lastName.
		name := getUserField(m, "firstName", "first_name", "given_name")
		last := getUserField(m, "lastName", "last_name", "family_name")
		displayName := strings.TrimSpace(name + " " + last)

		if displayName == "" {
			displayName = getUserField(m, "name", "full_name", "display_name", "username")
		}
		if displayName == "" {
			displayName = getUserField(m, "email", "mail", "userName", "username")
		}

		email := getUserField(m, "email", "mail", "userName", "username")

		// Active status.
		active := determineActive(m)
		activeStr := ""
		if active != nil && *active {
			activeStr = "Yes"
		} else if active != nil && !*active {
			activeStr = "No"
		}

		rows = append(rows, sortRow{Name: displayName, Email: email, Active: activeStr})
	}

	// Sort: active first, then inactive, then unknown.
	sortRows(rows)

	// Calculate column widths.
	colWidths := make([]int, len(cols))
	for i, c := range cols {
		colWidths[i] = c.wid
	}
	for _, r := range rows {
		if len(r.Name) > colWidths[0] {
			colWidths[0] = len(r.Name)
		}
		if len(r.Email) > colWidths[1] {
			colWidths[1] = len(r.Email)
		}
		if len(r.Active) > colWidths[2] {
			colWidths[2] = len(r.Active)
		}
	}

	// Cap widths to maxWidth.
	contentTotal := 0
	separators := len(cols) + 1
	for _, w := range colWidths {
		contentTotal += w
	}
	lineLen := contentTotal + separators
	if lineLen > maxWidth {
		overflow := lineLen - maxWidth
		for i := range colWidths {
			if overflow <= 0 {
				break
			}
			reduction := colWidths[i] - 6
			if reduction < 0 {
				reduction = 0
			}
			colWidths[i] -= reduction
			overflow -= reduction
		}
	}

	// Build the table.
	var sb strings.Builder

	hLine := func(ch string) string {
		if ch == "=" {
			return "┏" + strings.Repeat("━", contentTotal-separators+1) + "┓"
		}
		return "├" + strings.Repeat("─", contentTotal-separators+1) + "┤"
	}

	sb.WriteString(hLine("=") + "\n")

	// Header.
	var headerParts []string
	for i, c := range cols {
		headerParts = append(headerParts, leftPad(c.name, colWidths[i]))
	}
	sb.WriteString("│" + strings.Join(headerParts, "│") + "│" + "\n")
	sb.WriteString(hLine("-") + "\n")

	// Rows.
	for _, r := range rows {
		var parts []string
		parts = append(parts, leftPad(r.Name, colWidths[0]))
		parts = append(parts, leftPad(r.Email, colWidths[1]))
		parts = append(parts, leftPad(r.Active, colWidths[2]))
		sb.WriteString("│" + strings.Join(parts, "│") + "│" + "\n")
		sb.WriteString(hLine("-") + "\n")
	}

	return sb.String()
}

func leftPad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func getUserField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func determineActive(m map[string]any) *bool {
	// Check is_active.
	if v, ok := m["is_active"]; ok {
		switch val := v.(type) {
		case bool:
			return &val
		case string:
			b := strings.ToLower(val) == "true" || val == "1"
			return &b
		}
	}

	// Check active.
	if v, ok := m["active"]; ok {
		switch val := v.(type) {
		case bool:
			return &val
		}
	}

	// Check status.
	if v, ok := m["status"]; ok {
		if s, ok := v.(string); ok {
			lower := strings.ToLower(strings.TrimSpace(s))
			if lower == "active" {
				b := true
				return &b
			}
			if lower == "inactive" {
				b := false
				return &b
			}
		}
	}

	return nil
}

type sortRow struct {
	Name   string
	Email  string
	Active string
}

func sortRows(rows []sortRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if shouldSwap(rows[i], rows[j]) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func shouldSwap(a, b sortRow) bool {
	aActive := sortKey(a.Active)
	bActive := sortKey(b.Active)
	if aActive != bActive {
		return aActive > bActive // 1 (Yes) before 2 (No) before 3 (unknown)
	}
	return strings.ToLower(a.Name) > strings.ToLower(b.Name)
}

func sortKey(active string) int {
	switch active {
	case "Yes":
		return 1
	case "No":
		return 2
	default:
		return 3
	}
}
