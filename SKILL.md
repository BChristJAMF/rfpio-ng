# RFPIO MCP Tools - Usage Guide

This file provides detailed guidance for using the RFPIO MCP tools.
The descriptions here are loaded at startup and used to advertise tool
capabilities to LLM clients. Edit this file to improve tool usage without
modifying the server code.

---

## search_content

Search the RFPIO content library by keyword, IDs, or tags.

### Description

There are three ways to search the content library:

1. **Keyword search** - If you're looking for a word or phrase, use the keyword parameter.
2. **ID lookup** - If you already know one or more IDs of a library item, use the ids parameter to directly fetch those items.
3. **Tag filtering** - Filter results by tag names using the tags parameter. Tags can be used alone to find all entries with those tags, or combined with keyword to narrow results further.

Tag matching is case-insensitive.

The results will return a **Score** property and a **Last Updated** property. The higher the Score, the better the answer. Where more than one answer have close scores, the Last Updated property should be used as the tie breaker.

Take note of the **ID**, **SimHash**, and **word count** returned for each answer as they can be useful for additional operations. By computing the Hamming distance of two SimHash results, you can determine how similar two library answers are; the word count reflects the length of the answer text.

Each result lists its **answers**, each with a **key** (e.g. "Response", or "Self Score" and "Notes/Justification") and a **type** (RICH_TEXT for HTML text, SINGLE for a pick-list value). Use these keys as update_answer's answer_key or answers; most entries have no "Default" answer.

Each result also carries **Project Name** and **Section Name** (which RFP project/section the entry came from) and a **Content URL** (a direct link to the library entry) when the record has them.

**Digest Mode:** When you only need to identify, compare, or triage answers without reading their full text, use digest mode. This returns only metadata (ID, Score, Rating, Last Used, Last Updated, Project Name, Section Name, Content URL, Question, Alternate Questions, Tags, and per-answer SimHash and word count) and omits the answer body.

Digest mode is useful for:
- Listing or scanning large result sets efficiently
- Comparing answers by score, date, or SimHash without needing the full text
- Identifying duplicates or candidates before fetching specific answers in full

### Parameters

**keyword** (string, optional)
- The keyword is the string to search for
- The string can contain a comma separated list of terms to search on
- Both leading and trailing wildcards are supported for each term
- Using wildcards gives the best results
- **IMPORTANT:** For multi-word phrases, replace spaces with %20 (e.g., "audit%20logging" instead of "audit logging")

**ids** (string, optional)
- This is a string which contains one or more IDs of an entry in the content library
- If you want to fetch multiple entries, the string can be a comma separated list of IDs

**tags** (string, optional)
- Comma-separated tag names to filter search results (case-insensitive)
- Can be used alone or combined with keyword to narrow results
- Example: "Cloud,HIPAA" or "Workflow"

**digest** (boolean, default: false)
- When true, return only metadata (ID, Score, Rating, Last Used, Last Updated, Project Name, Section Name, Content URL, Question, Alternate Questions, Tags, and per-answer SimHash and word count) and omit the answer text
- Use this when you need to scan, compare, or triage results without reading the full answer body

**limit** (integer, default: 10)
- Maximum number of results to return

### Constraints

- Provide at least one of: keyword, ids, or tags
- When using ids, do not use keyword (they are mutually exclusive)

---

## add_answer

Add a new question and answer pair to the content library.

### Description

**[WRITE OPERATION]**

This tool will add a question and answer pair to the content library. It requires two strings: question and answer, and optionally accepts alternate questions and tags. It will return the ID of the new content library entry.

**IMPORTANT FORMATTING NOTES:**

1. **Line breaks:** When the answer contains paragraph breaks or bullet lines, use `<br>` for each line break. Newline characters (\n) are stripped by Responsive's API on write.
   - Use `<br><br>` for paragraph breaks
   - Use single `<br>` for line breaks within a list

2. **Special characters:** Parentheses and apostrophes need no encoding through MCP; stored entries use them literally. The entity forms `&#40;` `&#41;` `&#39;` are only a shell-quoting workaround for the CLI.

3. **Answer key:** The answer is stored under the key "Default". To change an existing entry's named answers (e.g. "Response", "Notes/Justification"), use update_answer with answer_key instead of adding a new entry.

### Parameters

**question** (string, required)
- The question associated with the answer

**answer** (string, required)
- The answer to the question being supplied
- HTML; use `<br>` for line breaks (not \n)

**alternate_questions** (string, optional)
- Alternate phrasings of the question that should also match this answer
- Provide multiple as a newline-delimited (\n) or `<br>`-delimited string; each becomes a separate alternate question
- Omit to add none

**tags** (string, optional)
- Comma-separated tag names to attach to the new entry (e.g., "Cloud,HIPAA")
- Omit to add none

### Important Notes

- Always use `<br>` tags for line breaks, never newline characters
- The tool returns the new content ID for reference

---

## update_answer

Update an existing content-library entry in place.

### Description

**[WRITE OPERATION]**

This tool updates an existing entry. It requires the ID of the entry and at least one field to change (question, answer, answers, alternate_questions, or tags).

Partial updates are non-destructive: the tool reads the full record, overlays only the fields you supply, and writes the whole object back. Every other field — other answers, alternate questions, tags, custom fields, owners, rating, review status, project and section — is preserved, and the order of the answers is kept.

**NAMED ANSWERS:** An entry holds one or more named answers (e.g. a single "Response", or "Self Score" + "Notes/Justification"). Most entries have no "Default" answer. Before writing an answer, call `search_content(ids="<content_id>")` and read the `key` and `type` of each item in `answers`.

- `answer` + `answer_key`: replace the answer with that key (exact match first, then case-insensitive)
- `answers`: an object mapping answer key -> new value, to replace several answers in one atomic write
- `answer` alone: replaces the "Default" answer, or the entry's only answer if it has exactly one; fails if the entry has several answers and none is "Default"

The tool never adds a new answer slot. An unknown key fails with an error listing the entry's real answer keys, and nothing is written. To add a new named answer (for example "Default") to an entry, the user first adds it as a response header in the RFPIO app, which needs some text in the response field to save; after that, use `answer_key`. Some imported entries repeat the same key on several answers (e.g. "DESCRIPTION" on every column); a key that matches more than one answer is refused, and those answers must be edited in the RFPIO app.

**ENTRIES UNDER REVIEW:** An entry is under review when review is turned on, or it has a review due date, cycle, workflow, follow-up dates or a last-reviewed date. Such entries are refused by default, because updating them through the API may reset their review settings; that has not been tested. Ask the user first, and only then retry with `allow_under_review=true`.

**FORMATTING NOTES:**

1. **RICH_TEXT answers** are stored as HTML wrapped in `<div>...</div>`. Plain text is accepted: it is HTML-escaped (`&`, `<`, `>`), newlines become `<br>`, and the result is wrapped in `<div>`. To control formatting, pass HTML (sent unchanged):
   - Use `<br><br>` for paragraph breaks
   - Use single `<br>` for line breaks within a list
   - Raw newlines inside HTML are stripped by Responsive's API on write

2. **Special characters:** Parentheses and apostrophes need no encoding through MCP; stored entries use them literally. The entity forms `&#40;` `&#41;` `&#39;` are a shell-quoting workaround for the CLI. Any tag or entity makes the value count as HTML (sent unchanged), so escape a literal `&` as `&amp;` in that case.

3. **Other answer types** (e.g. `SINGLE`, a pick-list such as "Self Score") are written verbatim; keep to the existing option format, e.g. "3 - Exceeds requirements".

### Parameters

**content_id** (string, required)
- The ID of the library entry to update
- Use search_content to find the ID

**question** (string, optional)
- The new question text
- Omit to keep existing question unchanged

**answer** (string, optional)
- New value for one answer: the answer named by answer_key, else "Default" or the entry's only answer
- Cannot be combined with answers
- Omit to keep existing answers unchanged

**answer_key** (string, optional)
- Key of the existing answer to replace with `answer`, e.g. "Notes/Justification"
- Requires answer; get valid keys from search_content(ids=...)

**answers** (object, optional)
- Map of answer key -> new value, e.g. {"Self Score": "3 - Exceeds requirements", "Notes/Justification": "..."}
- Every key must already exist on the entry, or nothing is written
- Cannot be combined with answer/answer_key

**alternate_questions** (string, optional)
- New alternate phrasings of the question, as a newline-delimited (\n) or `<br>`-delimited string
- Replaces the existing set of alternate questions
- Omit to keep existing alternate questions unchanged

**tags** (string, optional)
- New tags as a comma-separated list of names (e.g., "Cloud,HIPAA")
- Replaces the existing tag set
- Omit to keep existing tags unchanged

**allow_under_review** (boolean, optional, default: false)
- Set true only after the user explicitly agrees to update an entry that is under review
- RFPIO may reset the entry's review settings; by default such entries are refused and nothing is written

### Constraints

- At least one of question, answer, answers, alternate_questions, or tags must be provided
- Never adds answer slots; unknown answer keys are rejected with the list of valid keys
- A key that matches more than one answer on the entry is rejected
- Entries under review are rejected unless allow_under_review is true

---

## delete_answers

Delete one or more question and answer pairs from the content library.

### Description

**[WRITE OPERATION]**

This tool will delete one or more question and answer pairs in the content library. It requires a string of one or more IDs referencing entries in the library.

By default, items are moved to trash (soft delete). Use permanent=True for permanent deletion.

### Parameters

**ids** (string, required)
- A list of one or more IDs of library entries to be deleted
- Use comma separation when specifying multiple IDs
- Example: "id1,id2,id3"

**permanent** (boolean, default: false)
- If false (default), items are moved to trash and can be recovered
- If true, items are permanently deleted and cannot be recovered

### Important Notes

- Default behavior is soft delete (move to trash)
- Multiple IDs can be deleted in a single operation
- Use search_content to find the IDs of items to delete

---

## query_users

List users from the RFPIO system.

### Description

Fetches the user list from the RFPIO /users endpoint. Returns user information including name, email, and active status.

### Parameters

**limit** (integer, default: 50)
- Maximum number of users to return
- Controls the size of the result set

---

## list_tags

List tags in the RFPIO content library, optionally filtered by name.

### Description

Returns tags that exist in the content library. Tags are used to organize and categorize content. Use this to discover available tags before filtering searches or managing tags.

The content library can hold thousands of tags — enough that returning them all at once approaches the transport's 1MB result limit. Use **name_filter** to fetch just the tags you care about (e.g., before renaming or deleting one, or to check whether a tag already exists). When no filter is given, the result is bounded by **limit** so the call never fails on size.

### Parameters

**name_filter** (string, optional)
- Case-insensitive substring matched against tag names
- Example: `name_filter="fido"` returns `FIDO`, `FIDO U2F Tokens`, `FIDO security key`
- Omit (or pass empty) to list all tags, bounded by `limit`

**limit** (integer, default: 1000)
- Maximum number of tags to return
- The full tag set is large enough to approach the 1MB result cap, so output is bounded by default
- Pass `0` to return every match (only advisable with a `name_filter` that narrows the set)

### Returns

A dict with:
- **tags** — list of matching tag objects with names and IDs. Tag IDs are needed for update_tag and delete_tag operations.
- **total** — how many tags matched `name_filter` before `limit` was applied
- **truncated** — `true` if `limit` dropped some matches; refine `name_filter` or raise `limit` to see the rest

---

## create_tag

Create a new tag in the content library.

### Description

**[WRITE OPERATION]**

Creates a new tag that can be used to categorize content library entries. After creating a tag, you can assign it to answers when adding or updating them.

### Parameters

**name** (string, required)
- The name for the new tag
- Tag names should be descriptive and consistent with existing taxonomy

### Returns

The new tag ID and a success message. Save the tag ID for future reference if you need to update or delete the tag.

---

## update_tag

Update a tag name in the content library.

### Description

**[WRITE OPERATION]**

Renames an existing tag. The tag ID remains the same, but the displayed name changes. All content tagged with this tag will show the new name.

Use list_tags to find the tag ID before calling this tool.

### Parameters

**tag_id** (string, required)
- The ID of the tag to update
- Use list_tags to find tag IDs

**new_name** (string, required)
- The new name for the tag
- This replaces the existing tag name

### Important Notes

- Always call list_tags first to find the tag ID
- The tag ID does not change, only the name
- All content using this tag will show the updated name

---

## delete_tag

Delete a tag from the content library.

### Description

**[WRITE OPERATION]**

Permanently deletes a tag from the system. This removes the tag from all content that was tagged with it.

Use list_tags to find the tag ID before calling this tool.

### Parameters

**tag_id** (string, required)
- The ID of the tag to delete
- Use list_tags to find tag IDs

### Important Notes

- Always call list_tags first to find the tag ID
- Deletion is permanent
- The tag will be removed from all content that was tagged with it
- Consider the impact before deleting widely-used tags
