# RFPIO-Go

A Go port of the [RFPIO](https://www.rfpio.com/) CLI and MCP server.  
Provides a command-line interface and Model Context Protocol (MCP) server for interacting with the RFPIO (Responsive) content library API.

## Features

- **Content Search** — Keyword search, ID lookup, tag filtering
- **Q&A Management** — Add, update, delete Q&A pairs
- **Tag Management** — Create, update, delete, enumerate tags
- **User Management** — List system users
- **SimHash Fingerprinting** — 64-bit LSH for answer similarity detection
- **Non-destructive Updates** — Read-modify-write preserves all record fields
- **Unicode Box Drawing** — Terminal-friendly formatted output
- **MCP Server** — AI-assisted content library interaction

## Quick Start

### Prerequisites

- Go 1.21+

### Build

```bash
./build.sh
```

This produces two binaries in `bin/`:

| Binary | Description |
|--------|-------------|
| `bin/rfpio` | CLI tool |
| `bin/rfpio-mcp` | MCP server (stdio) |

### Configuration

Create a config file with:

```bash
./bin/rfpio config init
```

This creates `~/.config/rfpio/rfpio-config.json`:

```json
{
  "api_token": "your_api_token_here",
  "site_url": "https://your-rfpio-instance.com"
}
```

Alternatively, set environment variables:

```bash
export RFPIO_API_TOKEN="your_api_token"
export RFPIO_SITE_URL="https://your-rfpio-instance.com"
```

## CLI Usage

### Query Operations

```bash
# List users
./bin/rfpio q users --limit 50

# Search by keyword
./bin/rfpio q search --keyword "security" --limit 10

# Search by tag
./bin/rfpio q search --tags "Cloud,HIPAA" --limit 10

# Search with digest (metadata only)
./bin/rfpio q search --keyword "compliance" --digest

# Fetch by ID
./bin/rfpio q search --id "abc123,def456"

# List tags
./bin/rfpio q tags --limit 100
```

### Action Operations

```bash
# Add a Q&A pair
./bin/rfpio a add \
  --question "What is your SLA?" \
  --answer "We provide 99.9% uptime." \
  --tags "Cloud,SLA"

# Update an existing entry (its "Default" answer, or its only answer)
./bin/rfpio a update \
  --id "abc123" \
  --answer "Updated answer text."

# Update one named answer; find the keys with: ./bin/rfpio q search --id "abc123"
./bin/rfpio a update \
  --id "abc123" \
  --answer-key "Notes/Justification" \
  --answer "Updated notes."

# Entries under review (review on, due date, cycle or workflow) are refused;
# add --allow-under-review only if you accept that RFPIO may reset their review
./bin/rfpio a update --id "abc123" --answer "Updated answer text." --allow-under-review

# Delete entries
./bin/rfpio a delete --id "abc123,def456"

# Permanent deletion
./bin/rfpio a delete --id "abc123" --permanent
```

## MCP Server

The MCP server exposes all RFPIO operations as tools for AI assistants, using the [mcp-go](https://github.com/mark3labs/mcp-go) library:

```bash
./bin/rfpio-mcp
```

Tools:
- `query_users` — List system users
- `search_content` — Search by keyword, ID, or tags
- `add_answer` — Add a new Q&A pair
- `update_answer` — Update an existing entry: replace a named answer (`answer` + `answer_key`) or several at once (`answers`), plus question, alternate questions, tags
- `delete_answers` — Delete Q&A pairs
- `list_tags` — List content library tags
- `create_tag` — Create a new tag
- `delete_tag` — Delete a tag
- `update_tag` — Rename a tag
- `tag_usage_count` — Count entries per tag
- `list_answers_by_tag` — Enumerate entries by tag

## RFPIO API Notes

Behavior of the RFPIO update endpoint that the code works around:

- **Answers are named, and most entries have no `Default`.** An entry holds one or more answers, each with a `key` and a `type` (`RICH_TEXT` for HTML text, `SINGLE` for a pick-list value), e.g. `Response`, or `Self Score` + `Notes/Justification`. `update_answer` replaces the value of an existing answer and never adds one.
- **The answers list must keep its shape.** Sending an extra answer fails with `HTTP 400 ... Index 2 out of bounds for length 2` on a two-answer entry. The server appears to map answers to the entry's answer slots by position (inferred from that error, not tested), so the tool preserves the count, keys and order, and refuses any write that would change them.
- **The endpoint replaces the whole record**, so updates are read-modify-write: GET the record, overlay the changes, POST it all back.
- **`answerMatch` and `contentScore` are computed by the server.** An answer write that carried the old values came back with both recomputed, so they are left out of answer writes.
- **Adding a named answer is done in the RFPIO app, not the API.** Responsive calls the named answers "response headers"; its help center says anyone who can edit a Q&A pair can create, edit or delete them. The API documents no way to add one (the published update type is `question` plus `answers` of `{key, value, mergeTag, options}`), so `update_answer` never adds one. A new header needs some text in the response field to save. Once it exists, `update_answer` with `answer_key` fills it.
- **Keys are not unique within an entry.** Some imported entries repeat a key (e.g. `DESCRIPTION` on every column), so slots are told apart by position. A key that matches several answers is refused.
- **`RICH_TEXT` values are stored as HTML wrapped in `<div>`.** Plain text sent to a `RICH_TEXT` answer is escaped, newlines become `<br>`, and it is wrapped in `<div>`; input that already contains tags or entities is sent unchanged.
- **An empty `dueDate` (`""`) is stored as null and then left out of GET responses.** Nothing is lost. Real due dates are date-only (e.g. `2026-10-16`), unlike `createdDate` and `updateDate`, and come with `reviewStatus`, `lastReviewedOn`, `customReviewCycleInDays` and `reviewWorkflow`.
- **Entries under review are refused by `update_answer` unless the caller opts in** (`allow_under_review`, CLI `--allow-under-review`). The published update type calls the review switch `enableReview` while live records call it `reviewStatus`, and it is not known that a replace-style update preserves review settings. This has not been tested.
- **Responsive's published API types** (developer.responsive.io, a Stoplight project) are incomplete and differ from live records: they omit answer `type` and `answerSourceType`, and list `answers[].options` (dropdown choices), which no read ever returns.

## Project Structure

```
RFPIO-go/
├── build.sh              # Build script
├── go.mod                # Go module
├── internal/core/        # Core API library
│   ├── config.go         # Config loading
│   ├── http.go           # HTTP primitives
│   ├── htmltext.go       # HTML-to-Markdown converter
│   ├── simhash.go        # SimHash64 fingerprinting
│   ├── extraction.go     # Data extraction helpers
│   ├── normalization.go  # Item normalization & tag resolution
│   ├── merge.go          # Answer merge: named-answer replace, value formatting
│   ├── errors.go         # Error types (API error, answer key not found)
│   └── api.go            # API functions (users, search, Q&A, tags)
├── internal/skills/      # SKILL.md loader (applied to MCP tool descriptions)
├── cmd/rfpio/            # CLI tool
│   ├── main.go           # Entry point & argument parsing
│   ├── table.go          # Table renderer
│   └── reports.go        # Report renderers
├── cmd/rfpio-mcp/        # MCP server (full implementation)
│   └── server.go         # MCP stdio server with all tools
└── fixtures/             # Test fixtures (a real record, sanitized)
```

## Differences from Python Version

| Aspect | Python | Go |
|--------|--------|-----|
| HTTP client | `urllib.request` | `net/http` |
| SSL | `ssl` module | `crypto/tls` |
| HTML parsing | `html.parser` | Regex-based |
| JSON | `json` module | `encoding/json` |
| Concurrency | N/A (single-threaded) | Thread-safe core |
| MCP Server | `mcp.server.fastmcp` | `mcp-go` (full implementation) |

## Dependencies

The core library and CLI use only the Go standard library — no external dependencies.

The MCP server uses:
- `github.com/mark3labs/mcp-go` — MCP protocol implementation

## License

Same as the original Python project.
