# task-fix.md — Differences Found in Go Version (vs Python CLI)

Last compared: 2026-10-07

## Bug Fixes Applied

### 1. Users query returned empty results
- **Root cause**: `extractUsers` didn't check for `userVOList` key. The users API returns `{"totalResultsCount": N, "userVOList": [...]}`.
- **Fix**: Added `"userVOList"` to the key list in `extractUsers()`.

### 2. Search answers showed "(no answers)"
- **Root cause**: `answersOut` was declared as `[]map[string]any` but the report type-asserted `it["answers"].([]any)`. In Go, `[]map[string]any` is a different type from `[]any` — the assertion failed silently.
- **Fix**: Changed `answersOut` type from `[]map[string]any` to `[]any` in `normalizeItem()`.

### 3. Tags not displayed in search results
- **Root cause**: `normalizeItem` produces `tags` as `[]string`, but the report type-asserted `it["tags"].([]any)`. Same Go type mismatch issue.
- **Fix**: Updated report to accept both `[]string` and `[]any` for tags.

### 4. Answer text contained raw HTML
- **Root cause**: The `value` stored in the answer entry was the raw HTML string from the API. The cleaned text (`htmlToText(value)`) was only used for SimHash computation, not stored.
- **Fix**: Store `text` (cleaned) instead of `value` (raw HTML) in the answer entry.

### 5. Word count computed on raw HTML
- **Root cause**: `wordCount(value)` was called with raw HTML, counting HTML tags as part of words. Python computes word count on cleaned text: `len(html_to_text(value).split())`.
- **Fix**: Changed to `wordCount(text)` where `text` is already cleaned. Also updated `wordCount()` to not call `htmlToText` again (avoid double-processing).

### 6. Double htmlToText in simhash64
- **Root cause**: `simhash64()` internally called `htmlToText(text)` even though `normalizeItem` already passed cleaned text.
- **Fix**: Removed the `htmlToText` call from inside `simhash64()` — it now assumes pre-cleaned text.

## Remaining Differences

### 1. Word count off by 1 (items 0, 2)

| Item | Python | Go |
|------|--------|-----|
| 0 (security IAM) | 553 | 552 |
| 1 (dev best practice) | 260 | 260 ✓ |
| 2 (SOC2 security) | 215 | 214 |

- **Cause**: Go's regex-based `htmlToText` and Python's `HTMLParser`-based `html_to_text` produce slightly different text for some HTML inputs. The difference is typically 1 whitespace or newline, which affects word count.
- **Impact**: Minor — metadata value only, does not affect content display.

### 2. SimHash differs for item 2

| Item | Python | Go |
|------|--------|-----|
| 0 | 15239154359329701686 | 15239154359329701686 ✓ |
| 1 | 14986952779920793255 | 14986952779920793255 ✓ |
| 2 | 14408417288098689390 | 10949657172358345070 |

- **Cause**: The SimHash algorithm is identical in both implementations (tokenize → count → 64-bit vector → fingerprint). The difference on item 2 is caused by the slightly different text produced by the HTML-to-text functions (same root as #1 above).
- **Impact**: Minor — SimHash is used for deduplication/similarity. The hash values are still valid fingerprints; they just won't match cross-language for borderline cases.

### 3. Column alignment cosmetic difference

Python uses left-aligned headers with even padding:
```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┳━━━━━━━━┓
│Name                          │Email                                 │Active  │
```

Go uses right-aligned label headers with uneven padding:
```
┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
│                          Name│                                 Email│  Active│
```

- **Cause**: Different table header formatting logic. Go puts labels at the right edge of each column; Python puts them left-aligned.
- **Impact**: Cosmetic only. Both show the same data with the same box-drawing characters.

### 4. Next cursor shown in Go only

Go shows `[next_cursor: ...]` at the bottom of search results. Python does not.

- **Impact**: Minor — Go provides more paging information.

### 5. Link formatting

Go converts `<a href="...">text</a>` to `[text](url)` Markdown format. Python shows plain text (just the link text).

- **Impact**: Cosmetic — Go's version is arguably more useful (preserves the URL).

## Test Matrix

| Test Case | Python | Go | Status |
|-----------|--------|-----|--------|
| `q users --limit 3` | 3 users shown | 3 users shown | ✅ Fixed |
| `q search --keyword "security" --limit 3` | Full output with answers | Full output with answers | ✅ Fixed |
| `q search --keyword "security" --digest` | Metadata only, no body | Metadata only, no body | ✅ OK |
| `q search --id <id>` | Full record fetched | Full record fetched | ✅ OK |
| `q tags --limit 5` | Tags listed with counts | Tags listed with counts | ✅ OK |
| Tags in search results | Displayed | Displayed | ✅ Fixed |
| Answer text display | Clean text | Clean text | ✅ Fixed |
| Word count values | Correct | ±1 on some items | ⚠️ Minor |
| SimHash values | Correct | Match on 2/3 items | ⚠️ Minor |

## Action Items

1. **[Low]** Match Go's `htmlToText` output more closely to Python's `HTMLParser`-based version to eliminate word count / SimHash differences. Key areas to align: `<br>` handling, whitespace normalization, entity handling.
2. **[Low]** Align Go column header alignment with Python's left-aligned style for visual consistency.
3. **[Done]** Users query returns data.
4. **[Done]** Answers display correctly.
5. **[Done]** Tags display in search results.
6. **[Done]** HTML stripped from answer display text.
