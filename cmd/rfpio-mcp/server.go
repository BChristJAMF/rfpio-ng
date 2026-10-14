package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/bryanchrist/rfpio-go/internal/core"
	"github.com/bryanchrist/rfpio-go/internal/skills"
)

// ---------------------------------------------------------------------------
// Credential management
// ---------------------------------------------------------------------------

func getConfig() *core.Config {
	configPath := os.Getenv("RFPIO_CONFIG")
	cfg, err := core.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading RFPIO config: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

var tools = []mcp.Tool{
	mcp.NewTool("query_users",
		mcp.WithDescription("Query RFPIO users. Fetches the user list from the RFPIO /users endpoint."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of users to return (default 50)."),
			mcp.DefaultNumber(50),
		),
	),
	mcp.NewTool("search_content",
		mcp.WithDescription("Search RFPIO content library by keyword, IDs, or tags. ID lookup takes precedence. Each item lists its answers with key and type (e.g. RICH_TEXT, SINGLE); use those keys as update_answer's answer_key/answers."),
		mcp.WithString("keyword",
			mcp.Description("Search term(s). Use %20 for spaces. Supports wildcards. Comma-separated for multiple terms."),
		),
		mcp.WithString("ids",
			mcp.Description("Comma-separated content IDs to fetch exactly. Non-existent IDs are omitted."),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tag names (case-insensitive). Can be used alone or with keyword."),
		),
		mcp.WithBoolean("digest",
			mcp.Description("When true, omit answer body text. SimHash and word_count remain."),
			mcp.DefaultBool(false),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results (default 10, clamped to API's 0..100 range)."),
			mcp.DefaultNumber(10),
		),
		mcp.WithString("cursor",
			mcp.Description("Paging cursor. Pass previous response's next_cursor to page forward."),
		),
	),
	mcp.NewTool("list_tags",
		mcp.WithDescription("List tags in the RFPIO content library, optionally filtered by name. Client-side filtering."),
		mcp.WithString("name_filter",
			mcp.Description("Case-insensitive substring. E.g. 'fido' matches 'FIDO', 'FIDO U2F Tokens'."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max tags to return (default 1000). Pass 0 for all matches."),
			mcp.DefaultNumber(1000),
		),
	),
	mcp.NewTool("tag_usage_count",
		mcp.WithDescription("Count how many content-library entries carry a given tag."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The tag name to count (any case)."),
		),
	),
	mcp.NewTool("list_answers_by_tag",
		mcp.WithDescription("List the content IDs of every entry carrying a given tag. Cursor-walks full membership."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The tag name to enumerate (any case)."),
		),
	),
	mcp.NewTool("add_answer",
		mcp.WithDescription("[WRITE] Add a new Q/A pair. The answer is stored under the key \"Default\". Use <br> for line breaks; parentheses and apostrophes need no encoding over MCP (&#40; &#41; &#39; are only a shell-quoting workaround)."),
		mcp.WithString("question",
			mcp.Required(),
			mcp.Description("The question text."),
		),
		mcp.WithString("answer",
			mcp.Required(),
			mcp.Description("The answer text, as HTML. Use <br> for line breaks (raw newlines are dropped by the API)."),
		),
		mcp.WithString("alternate_questions",
			mcp.Description("Alternate phrasings as newline-delimited (\\n) or <br>-delimited string."),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tag names to attach."),
		),
	),
	mcp.NewTool("update_answer",
		mcp.WithDescription(`[WRITE] Update an existing content-library entry in place. Non-destructive: the tool reads the full record, overlays only what you supply, and writes it back, so every other field (other answers, alternate questions, tags, custom fields, owners, rating, review status, project/section) and the order of the answers are preserved.

Entries hold one or more NAMED answers (e.g. "Response", or "Self Score" + "Notes/Justification"); most have no "Default" answer. Before writing an answer, call search_content(ids="<content_id>") and read the key and type of each item in its answers list.

Choosing the answer to write:
- answer + answer_key: replace the answer with that key (exact match first, then case-insensitive).
- answers: a JSON object mapping answer key -> new value, to replace several answers in one atomic write.
- answer alone: replaces the "Default" answer, or the entry's only answer if it has exactly one. Fails if the entry has several answers and none is "Default".
This tool never adds a new answer slot. An unknown key fails with an error listing the entry's real answer keys, and nothing is written.

Entries under review (review turned on, or a review due date, cycle or workflow set) are refused by default, because updating them through the API may reset their review settings; that has not been tested. Ask the user first, and only then retry with allow_under_review=true.

Answer value format:
- RICH_TEXT answers are stored as HTML wrapped in <div>...</div>. Plain text is accepted: it is HTML-escaped (& < >), newlines become <br>, and the result is wrapped in <div>. To control formatting, pass HTML instead (it is sent unchanged), using <br> for line breaks and <br><br> between paragraphs; raw newlines in HTML are dropped by the API.
- Parentheses and apostrophes need no encoding here (stored entries use them literally). Entity forms such as &#40; &#41; &#39; are still accepted, but any entity or tag makes the value count as HTML, so escape a literal & as &amp; in that case.
- Other answer types (e.g. SINGLE, a pick-list such as "Self Score") are written verbatim; keep to the existing option format, e.g. "3 - Exceeds requirements".`),
		mcp.WithString("content_id",
			mcp.Required(),
			mcp.Description("ID of the entry to update. Find it with search_content."),
		),
		mcp.WithString("question",
			mcp.Description("New question text (keeps existing if empty)."),
		),
		mcp.WithString("answer",
			mcp.Description("New value for one answer: the answer named by answer_key, else \"Default\" or the entry's only answer. Cannot be combined with answers."),
		),
		mcp.WithString("answer_key",
			mcp.Description("Key of the existing answer to replace with answer, e.g. 'Notes/Justification'. Requires answer. Get valid keys from search_content(ids=...)."),
		),
		mcp.WithObject("answers",
			mcp.Description("Replace several existing answers in one write: object mapping answer key -> new value, e.g. {\"Self Score\": \"3 - Exceeds requirements\", \"Notes/Justification\": \"...\"}. Keys match like answer_key; every key must already exist or nothing is written. Cannot be combined with answer/answer_key."),
			mcp.AdditionalProperties(map[string]any{"type": "string"}),
		),
		mcp.WithString("alternate_questions",
			mcp.Description("New alternate questions as a newline-delimited (\\n) or <br>-delimited string. Replaces the existing set."),
		),
		mcp.WithString("tags",
			mcp.Description("New tags as comma-separated names. Replaces the existing tag set."),
		),
		mcp.WithBoolean("allow_under_review",
			mcp.Description("Set true only after the user explicitly agrees to update an entry that is under review. RFPIO may reset its review settings. Default false: such entries are refused and nothing is written."),
			mcp.DefaultBool(false),
		),
	),
	mcp.NewTool("delete_answers",
		mcp.WithDescription("[WRITE] Delete Q/A pairs. Default: move to trash."),
		mcp.WithString("ids",
			mcp.Required(),
			mcp.Description("Comma-separated content IDs to delete."),
		),
		mcp.WithBoolean("permanent",
			mcp.Description("If true, permanently delete; if false, move to trash (default)."),
			mcp.DefaultBool(false),
		),
	),
	mcp.NewTool("create_tag",
		mcp.WithDescription("[WRITE] Create a new tag in RFPIO content library."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The tag name."),
		),
	),
	mcp.NewTool("delete_tag",
		mcp.WithDescription("[WRITE] Delete a tag from RFPIO content library."),
		mcp.WithString("tag_id",
			mcp.Required(),
			mcp.Description("The tag ID to delete."),
		),
	),
	mcp.NewTool("update_tag",
		mcp.WithDescription("[WRITE] Update a tag name in RFPIO content library."),
		mcp.WithString("tag_id",
			mcp.Required(),
			mcp.Description("The tag ID to update."),
		),
		mcp.WithString("new_name",
			mcp.Required(),
			mcp.Description("The new tag name."),
		),
	),
}

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------

func handleQueryUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()
	limit := req.GetInt("limit", 50)

	result, err := core.APIQueryUsers(cfg, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(result.Users)
}

func handleSearchContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	keyword := req.GetString("keyword", "")
	idStr := req.GetString("ids", "")
	tagStr := req.GetString("tags", "")
	digest := req.GetBool("digest", false)
	limit := req.GetInt("limit", 10)
	cursor := req.GetString("cursor", "")

	var ids []string
	if idStr != "" {
		for _, id := range strings.Split(idStr, ",") {
			if strings.TrimSpace(id) != "" {
				ids = append(ids, strings.TrimSpace(id))
			}
		}
	}

	var tags []string
	if tagStr != "" {
		for _, t := range strings.Split(tagStr, ",") {
			if strings.TrimSpace(t) != "" {
				tags = append(tags, strings.TrimSpace(t))
			}
		}
	}

	var keywordPtr *string
	if keyword != "" {
		keywordPtr = &keyword
	}

	result, err := core.APISearch(cfg, keywordPtr, ids, tags, digest, limit, cursor)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out := map[string]any{
		"items": result.Items,
	}
	if result.NextCursor != "" {
		out["next_cursor"] = result.NextCursor
	}
	if len(result.Omitted) > 0 {
		out["omitted"] = result.Omitted
	}
	return mcp.NewToolResultText(toJSON(out)), nil
}

func handleListTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	nameFilter := req.GetString("name_filter", "")
	limit := req.GetInt("limit", 1000)

	result, err := core.APITagList(cfg, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Client-side filter.
	if nameFilter != "" {
		filtered := make([]any, 0)
		for _, t := range result.Tags {
			if m, ok := t.(map[string]any); ok {
				if tn, ok := m["tagName"]; ok {
					if strings.Contains(strings.ToLower(fmt.Sprintf("%v", tn)), strings.ToLower(nameFilter)) {
						filtered = append(filtered, t)
					}
				}
			}
		}
		result.Tags = filtered
	}

	return toJsonResult(map[string]any{
		"tags":      result.Tags,
		"total":     result.Total,
		"truncated": result.Truncated,
	})
}

func handleTagUsageCount(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := core.APITagUsageCount(cfg, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"tag_name": result.TagName,
		"count":    result.Count,
	})
}

func handleListAnswersByTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := core.APIListAnswersByTag(cfg, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"tag_name":    result.TagName,
		"content_ids": result.ContentIDs,
		"count":       result.Count,
	})
}

func handleAddAnswer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	question := req.GetString("question", "")
	answer := req.GetString("answer", "")
	altQ := req.GetString("alternate_questions", "")
	tags := req.GetString("tags", "")

	if question == "" || answer == "" {
		return mcp.NewToolResultError("both question and answer are required"), nil
	}

	var altPtr *string
	if altQ != "" {
		altPtr = &altQ
	}

	var tagsPtr *[]string
	if tags != "" {
		tagList := strings.Split(tags, ",")
		for i, t := range tagList {
			tagList[i] = strings.TrimSpace(t)
		}
		tagsPtr = &tagList
	}

	result, err := core.APIAdd(cfg, question, answer, altPtr, tagsPtr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success": result.Success,
		"id":      result.ID,
		"message": result.Message,
	})
}

func handleUpdateAnswer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	contentID := req.GetString("content_id", "")
	question := req.GetString("question", "")
	answer := req.GetString("answer", "")
	answerKey := req.GetString("answer_key", "")
	altQ := req.GetString("alternate_questions", "")
	tags := req.GetString("tags", "")
	allowUnderReview := req.GetBool("allow_under_review", false)

	answers, err := parseAnswersArg(req.GetArguments()["answers"])
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if contentID == "" {
		return mcp.NewToolResultError("content_id is required"), nil
	}
	if answerKey != "" && answer == "" {
		return mcp.NewToolResultError("answer_key was given but no answer text; supply answer to set the named answer"), nil
	}
	if len(answers) > 0 && (answer != "" || answerKey != "") {
		return mcp.NewToolResultError("use either answers or answer/answer_key, not both"), nil
	}
	if answer != "" {
		answers = []core.AnswerUpdate{{Key: answerKey, Value: answer}}
	}
	if question == "" && len(answers) == 0 && altQ == "" && tags == "" {
		return mcp.NewToolResultError("at least one of question, answer, answers, alternate_questions, or tags must be provided"), nil
	}

	var qPtr, aqPtr, tPtr *string
	if question != "" {
		qPtr = &question
	}
	if altQ != "" {
		aqPtr = &altQ
	}
	if tags != "" {
		tPtr = &tags
	}

	result, err := core.APIUpdate(cfg, contentID, qPtr, answers, aqPtr, tPtr, allowUnderReview)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success": result.Success,
		"message": result.Message,
	})
}

// parseAnswersArg converts the update_answer "answers" argument (an object of
// answer key -> string value, or that object JSON-encoded as a string) into
// answer updates sorted by key. A nil or empty argument yields no updates.
func parseAnswersArg(raw any) ([]core.AnswerUpdate, error) {
	if s, ok := raw.(string); ok {
		if strings.TrimSpace(s) == "" {
			return nil, nil
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(s), &decoded); err != nil {
			return nil, fmt.Errorf("answers must be an object mapping answer key to value: %v", err)
		}
		raw = decoded
	}
	if raw == nil {
		return nil, nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("answers must be an object mapping answer key to value")
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	updates := make([]core.AnswerUpdate, 0, len(keys))
	for _, k := range keys {
		if strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("answers contains an empty answer key")
		}
		v, ok := m[k].(string)
		if !ok {
			return nil, fmt.Errorf("answers[%q] must be a string", k)
		}
		if v == "" {
			return nil, fmt.Errorf("answers[%q] is empty; to keep an answer unchanged, leave it out", k)
		}
		updates = append(updates, core.AnswerUpdate{Key: k, Value: v})
	}
	return updates, nil
}

func handleDeleteAnswers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	idStr := req.GetString("ids", "")
	permanent := req.GetBool("permanent", false)

	if idStr == "" {
		return mcp.NewToolResultError("ids parameter is required"), nil
	}

	var ids []string
	for _, id := range strings.Split(idStr, ",") {
		if strings.TrimSpace(id) != "" {
			ids = append(ids, strings.TrimSpace(id))
		}
	}

	result, err := core.APIDelete(cfg, ids, !permanent)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success":     result.Success,
		"deleted":     result.Deleted,
		"not_deleted": result.NotDeleted,
		"message":     result.Message,
	})
}

func handleCreateTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := core.APITagCreate(cfg, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success": result.Success,
		"tag_id":  result.ID,
		"message": result.Message,
	})
}

func handleDeleteTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	tagID, err := req.RequireString("tag_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := core.APITagDelete(cfg, tagID)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success": result.Success,
		"message": result.Message,
	})
}

func handleUpdateTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cfg := getConfig()

	tagID, err := req.RequireString("tag_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	newName, err := req.RequireString("new_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result, err := core.APITagUpdate(cfg, tagID, newName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return toJsonResult(map[string]any{
		"success": result.Success,
		"message": result.Message,
	})
}

// ---------------------------------------------------------------------------
// MCP Server setup
// ---------------------------------------------------------------------------

func main() {
	s := server.NewMCPServer("rfpio", "0.1.0",
		server.WithLogging(),
	)

	// Map tool names to handlers.
	handlers := map[string]server.ToolHandlerFunc{
		"query_users":         handleQueryUsers,
		"search_content":      handleSearchContent,
		"list_tags":           handleListTags,
		"tag_usage_count":     handleTagUsageCount,
		"list_answers_by_tag": handleListAnswersByTag,
		"add_answer":          handleAddAnswer,
		"update_answer":       handleUpdateAnswer,
		"delete_answers":      handleDeleteAnswers,
		"create_tag":          handleCreateTag,
		"delete_tag":          handleDeleteTag,
		"update_tag":          handleUpdateTag,
	}

	// Apply SKILL.md descriptions before registering, so clients see them.
	cfg := getConfig()
	skillPath := cfg.SkillFile
	if skillPath == "" {
		exe, _ := os.Executable()
		skillPath = filepath.Join(filepath.Dir(exe), "..", "SKILL.md")
	}
	if docs := skills.ParseSkillMarkdown(skillPath); docs != nil {
		n := skills.ApplyToTools(tools, docs)
		fmt.Fprintf(os.Stderr, "Applied %d tool descriptions from %s\n", n, skillPath)
	} else {
		fmt.Fprintf(os.Stderr, "No skill file at %s, using built-in tool descriptions\n", skillPath)
	}

	// Register tools with handlers.
	for i, tool := range tools {
		handler := handlers[tool.Name]
		if handler == nil {
			fmt.Fprintf(os.Stderr, "Warning: no handler for tool %s\n", tool.Name)
			continue
		}
		s.AddTool(tools[i], handler)
	}

	fmt.Fprintln(os.Stderr, "Starting RFPIO MCP server on stdio...")
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toJsonResult(v any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultJSON(v)
}

func toJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(data)
}
