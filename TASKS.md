# RFPIO-Go Port Tasks

## Status Legend
- [ ] Not started
- [x] Completed
- [🔧] In progress

---

## Phase 1: Project Setup
- [x] Initialize Go module and project structure
- [x] Create `build.sh` build script
- [ ] Add `go.mod` dependencies (HTTP client is stdlib only, no external deps needed for core)

## Phase 2: Core Library (`internal/core`)
- [x] Config loading (env vars + multi-location JSON file search)
- [x] HTTP primitives (GET, POST with Bearer auth, SSL context)
- [x] Exception/error types (`RFPIOAPIError`, `AnswerKeyNotFound`)
- [x] HTML-to-Markdown converter (`_HTMLToText` → `HTMLToText`)
- [x] SimHash64 computation
- [x] Data extraction helpers (`_extractAnswerItems`, `_extractFirstList`, `_getFirst`, `_getFirstDeep`)
- [x] Item normalization (`_normalizeItem`)
- [x] Tag name resolution (`_resolveTagNames`)
- [x] Answer merge helpers (`_mergeDefaultAnswer`, `_mergeAnswer`)
- [x] Write-payload helpers (`_normalizeAlternateQuestions`, `_normalizeTags`)
- [x] API: `api_query_users`
- [x] API: `api_search` (keyword + tag path with cursor paging, ID lookup path, back-fill, digest mode)
- [x] API: `api_add`
- [x] API: `api_update` (read-modify-write, answer_key support)
- [x] API: `api_delete`
- [x] API: `api_list_tags`
- [x] API: `api_create_tag`
- [x] API: `api_delete_tag`
- [x] API: `api_update_tag`
- [x] API: `api_tag_usage_count`
- [x] API: `api_list_answers_by_tag`
- [x] `create_sample_config` function

## Phase 3: CLI Tool (`cmd/rfpio`)
- [x] CLI argument parsing (`argparse` equivalent)
- [x] Query subcommands: `users`, `search` (keyword/ids/tags), `tags`
- [x] Action subcommands: `add`, `update`, `delete`
- [x] Unicode box-drawing table renderer
- [x] Search report renderer
- [x] Action report renderers (add, update, delete)
- [x] Tag report renderers (create, delete)
- [x] Main entry point wiring

## Phase 4: MCP Server (`cmd/rfpio-mcp`)
- [x] Basic MCP stdio server (stub with JSON-RPC)
- [x] Full MCP tools implementation with `mcp-go` dependency
- [x] Tool wrappers: `query_users`, `search_content`, `list_tags`, `tag_usage_count`, `list_answers_by_tag`
- [x] Write tools: `add_answer`, `update_answer`, `delete_answers`
- [x] Tag tools: `create_tag`, `delete_tag`, `update_tag`
- [x] Async concurrency limiter
- [x] Skill loader integration

## Phase 5: Skill Loader & Docs
- [x] Parse SKILL.md and apply to MCP tools
- [x] Copy SKILL.md into project
- [x] README.md

## Phase 6: Tests
- [ ] Unit tests for core functions (config, HTML-to-text, SimHash, normalization)
- [🔧] Unit tests for API functions (mock HTTP): `APIUpdate` covered; the other API functions are not
- [ ] Unit tests for CLI formatting
- [🔧] Test fixtures migration: one sanitized record (`fixtures/answer_get_named_answers.json`)

---

## Completed Worklog

### 2026-10-07
- Project setup: Go module, directory structure, build script created
- Core library: config loading, HTTP primitives, error types
- Core library: HTML-to-Markdown converter
- Core library: SimHash64, data extraction helpers
- Core library: Item normalization, tag name resolution, answer merge helpers
- Core library: All API functions (`query_users`, `search`, `add`, `update`, `delete`, `list_tags`, `create_tag`, `delete_tag`, `update_tag`, `tag_usage_count`, `list_answers_by_tag`)
- CLI: Argument parsing, all query subcommands (`users`, `search`, `tags`), all action subcommands (`add`, `update`, `delete`), Unicode box-drawing renderers
- MCP server (stub): Basic JSON-RPC stub server
- MCP server (full): Added `mcp-go` dependency, full MCP tool implementation with `ServerTool` pattern
- Skills: Parsed SKILL.md, skill loader package created
- Documentation: README.md, SKILL.md copied from Python project

### 2026-10-13
- `update_answer`: replace a named answer (`answer_key`) or several at once (`answers`), never adding an answer slot; unknown, ambiguous or missing-Default keys fail with the entry's real keys before anything is written
- `update_answer`: RICH_TEXT values are stored as `<div>` HTML; plain text is escaped, wrapped and given `<br>` line breaks
- `update_answer`: tags are sent as a list; stale `answerMatch` and `contentScore` are no longer echoed back on answer writes
- MCP server: `SKILL.md` descriptions are now applied to the tools (the parser existed but its result was discarded); the loader matches the Python one
- Tests: merge rules, the read-overlay-write payload (fake server), the MCP argument parsing and tool schema, the SKILL.md loader
- Documented RFPIO update-endpoint behavior in the README
- `update_answer`: refuses entries under review (review on, due date, cycle, workflow, follow-up or last-reviewed date) unless `allow_under_review` / `--allow-under-review` is set; review preservation on update is untested
- `update_answer`: an `answer_key` that matches several answers is rejected with the positions, instead of silently updating the first
- Normalized answer text no longer starts with a blank line (a leading `<div>` produced one); SimHash and word counts are unchanged
- Unknown-key and missing-Default errors now say a new named answer is added in the RFPIO app first
- Added sanitized fixture `fixtures/answer_get_under_review.json`
