package core

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Match HTML tags.
	tagRe = regexp.MustCompile(`(?s)<[^>]*>`)
	// Match <br> tags.
	brRe = regexp.MustCompile(`(?i)<br\s*/?>`)
	// Match common block-level tags that add paragraph breaks.
	blockBreakRe = regexp.MustCompile(`(?i)</?\s*(p|div|section|/?)\s*>`)
	// Match list markers.
	listRe = regexp.MustCompile(`^[\s]*[-*•]\s+`)
)

// htmlToText converts an HTML string to Markdown-ish plain text.
func htmlToText(s string) string {
	if s == "" {
		return ""
	}

	// Unescape HTML entities first.
	text := html.UnescapeString(s)

	// Remove script and style elements.
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	text = scriptRe.ReplaceAllString(text, "")
	text = styleRe.ReplaceAllString(text, "")

	// Replace <br> with newlines.
	text = brRe.ReplaceAllString(text, "\n")

	// Handle block-level tags.
	// Replace <p>, <div>, <section> opening tags with double newlines.
	openBlockRe := regexp.MustCompile(`(?i)<\s*(p|div|section)\b[^>]*>`)
	text = openBlockRe.ReplaceAllStringFunc(text, func(match string) string {
		return "\n\n"
	})
	// Replace closing block tags with newlines.
	closeBlockRe := regexp.MustCompile(`(?i)</\s*(p|div|section)\s*>`)
	text = closeBlockRe.ReplaceAllString(text, "\n")

	// Handle list items.
	// Replace <li> with bullet + newline.
	openLiRe := regexp.MustCompile(`(?i)<\s*li\b[^>]*>`)
	text = openLiRe.ReplaceAllStringFunc(text, func(match string) string {
		return "\n• "
	})
	closeLiRe := regexp.MustCompile(`(?i)</\s*li\s*>`)
	text = closeLiRe.ReplaceAllString(text, "\n")

	// Handle <ul> and <ol> tags.
	text = regexp.MustCompile(`(?i)<\s*/?\s*ul\s*>`).ReplaceAllString(text, "")
	text = regexp.MustCompile(`(?i)<\s*/?\s*ol\s*>`).ReplaceAllString(text, "")

	// Handle bold/italic.
	text = regexp.MustCompile(`(?i)<\s*/?\s*(strong|b)\s*>`).ReplaceAllString(text, "**")
	text = regexp.MustCompile(`(?i)<\s*/?\s*(em|i)\s*>`).ReplaceAllString(text, "*")

	// Handle code blocks.
	text = regexp.MustCompile(`(?i)<\s*pre\s*>`).ReplaceAllString(text, "```\n")
	text = regexp.MustCompile(`(?i)</\s*pre\s*>`).ReplaceAllString(text, "\n```")
	text = regexp.MustCompile(`(?i)<\s*code\s*>`).ReplaceAllString(text, "`")
	text = regexp.MustCompile(`(?i)</\s*code\s*>`).ReplaceAllString(text, "`")

	// Handle links: [text](url)
	// This is simplified; captures simple cases.
	linkRe := regexp.MustCompile(`(?i)<a\s+href="([^"]*)"[^>]*>(.*?)</a>`)
	text = linkRe.ReplaceAllStringFunc(text, func(match string) string {
		submatches := linkRe.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			url := submatches[1]
			linkText := submatches[2]
			// Strip any nested HTML from link text.
			linkText = tagRe.ReplaceAllString(linkText, "")
			return "[" + linkText + "](" + url + ")"
		}
		return match
	})

	// Remove all remaining HTML tags.
	text = tagRe.ReplaceAllString(text, "")

	// Normalize whitespace: collapse multiple spaces/lines.
	lines := strings.Split(text, "\n")
	var result []string
	prevBlank := false
	for _, line := range lines {
		cleaned := strings.Join(strings.Fields(line), " ")
		if cleaned == "" {
			if !prevBlank {
				result = append(result, "")
			}
			prevBlank = true
		} else {
			result = append(result, cleaned)
			prevBlank = false
		}
	}

	// Join and trim.
	out := strings.Join(result, "\n")
	out = strings.TrimFunc(out, unicode.IsSpace)

	// Clean up excessive blank lines.
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}

	return out
}
