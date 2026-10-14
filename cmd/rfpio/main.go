package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bryanchrist/rfpio-go/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "query", "q":
		handleQuery(os.Args[2:])
	case "action", "a":
		handleAction(os.Args[2:])
	case "config", "c":
		handleConfig(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println(`RFPIO CLI - Interactive content library manager

Usage:
  rfpio <subcommand> [options]

Subcommands:
  query (q)    Query operations
  action (a)   Modify operations
  config (c)   Configuration management

Query subcommands:
  rfpio q users [--limit N]              List users
  rfpio q search --keyword "term" [--limit N] [--tags tag1,tag2] [--digest]
  rfpio q search --id id1,id2,id3        Fetch by IDs
  rfpio q tags [--limit N] [--keyword "term"]

Action subcommands:
  rfpio a add --question "Q" --answer "A" [--tags tag1,tag2]
  rfpio a update --id id [--question "Q"] [--answer "A" [--answer-key "Key"]] [--allow-under-review]
  rfpio a delete --id id1,id2 [--permanent]

Config subcommands:
  rfpio c init                           Create sample config file

Options:
  --print-url       Print endpoint URL to stderr
  --debug PATH      Write raw JSON response to file
  --limit N         Maximum results
  --digest          Return metadata only (no answer body)
  --permanent       Permanently delete (default: move to trash)
  -u, --urlencoded  URL-encode the keyword`)
}

// ---------------------------------------------------------------------------
// Query handling
// ---------------------------------------------------------------------------

func handleQuery(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: query requires a subcommand (users, search, tags)")
		os.Exit(2)
	}

	cfg, err := core.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "users":
		handleQueryUsers(args[1:], cfg)
	case "search":
		handleQuerySearch(args[1:], cfg)
	case "tags":
		handleQueryTags(args[1:], cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown query subcommand: %s\n", subcommand)
		os.Exit(2)
	}
}

func handleQueryUsers(args []string, cfg *core.Config) {
	limit := 50
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				l, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid limit: %s\n", args[i+1])
					os.Exit(2)
				}
				limit = l
				i++
			}
		case "--print-url":
			fmt.Fprintln(os.Stderr, "[URL output disabled]")
		}
	}

	result, err := core.APIQueryUsers(cfg, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(drawUserTable(result.Users))
}

func handleQuerySearch(args []string, cfg *core.Config) {
	var keyword, idStr, tagStr string
	limit := 10
	digest := false
	urlencoded := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--keyword":
			if i+1 < len(args) {
				keyword = args[i+1]
				i++
			}
		case "-k":
			if i+1 < len(args) {
				keyword = args[i+1]
				i++
			}
		case "--id":
			if i+1 < len(args) {
				idStr = args[i+1]
				i++
			}
		case "--tags":
			if i+1 < len(args) {
				tagStr = args[i+1]
				i++
			}
		case "--limit":
			if i+1 < len(args) {
				l, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid limit: %s\n", args[i+1])
					os.Exit(2)
				}
				limit = l
				i++
			}
		case "--digest":
			digest = true
		case "--print-url":
			fmt.Fprintln(os.Stderr, "[URL output disabled]")
		case "-u", "--urlencoded":
			urlencoded = true
		case "--debug":
			if i+1 < len(args) {
				i++
			}
		}
	}

	// Parse IDs.
	var ids []string
	if idStr != "" {
		for _, id := range strings.Split(idStr, ",") {
			if strings.TrimSpace(id) != "" {
				ids = append(ids, strings.TrimSpace(id))
			}
		}
	}

	// Parse tags.
	var tags []string
	if tagStr != "" {
		for _, t := range strings.Split(tagStr, ",") {
			if strings.TrimSpace(t) != "" {
				tags = append(tags, strings.TrimSpace(t))
			}
		}
	}

	// URL-decode keyword if needed.
	if urlencoded && keyword != "" {
		// Simple URL decode - just replace + and %20.
		keyword = strings.ReplaceAll(keyword, "+", " ")
		keyword = strings.ReplaceAll(keyword, "%20", " ")
	}

	result, err := core.APISearch(cfg, &keyword, ids, tags, digest, limit, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(drawSearchReport(result.Items, digest))

	if result.NextCursor != "" {
		fmt.Fprintf(os.Stderr, "\n[next_cursor: %s]\n", result.NextCursor)
	}
	if len(result.Omitted) > 0 {
		fmt.Fprintf(os.Stderr, "Omitted (not found): %s\n", strings.Join(result.Omitted, ", "))
	}
}

func handleQueryTags(args []string, cfg *core.Config) {
	limit := 0 // 0 = no limit

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				l, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid limit: %s\n", args[i+1])
					os.Exit(2)
				}
				limit = l
				i++
			}
		}
	}

	result, err := core.APITagList(cfg, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(drawTagTable(result.Tags))
	if result.Total > 0 {
		fmt.Fprintf(os.Stderr, "\nShowing %d of %d total tags.\n", len(result.Tags), result.Total)
	}
}

// ---------------------------------------------------------------------------
// Action handling
// ---------------------------------------------------------------------------

func handleAction(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: action requires a subcommand (add, update, delete)")
		os.Exit(2)
	}

	cfg, err := core.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "add":
		handleActionAdd(args[1:], cfg)
	case "update":
		handleActionUpdate(args[1:], cfg)
	case "delete":
		handleActionDelete(args[1:], cfg)
	default:
		fmt.Fprintf(os.Stderr, "Unknown action subcommand: %s\n", subcommand)
		os.Exit(2)
	}
}

func handleActionAdd(args []string, cfg *core.Config) {
	var question, answer, altQuestions, tags string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--question":
			if i+1 < len(args) {
				question = args[i+1]
				i++
			}
		case "--answer":
			if i+1 < len(args) {
				answer = args[i+1]
				i++
			}
		case "--altquestions":
			if i+1 < len(args) {
				altQuestions = args[i+1]
				i++
			}
		case "--tags":
			if i+1 < len(args) {
				tags = args[i+1]
				i++
			}
		case "--print-url":
			fmt.Fprintln(os.Stderr, "[URL output disabled]")
		}
	}

	if question == "" || answer == "" {
		fmt.Fprintln(os.Stderr, "Error: --question and --answer are required for add")
		os.Exit(2)
	}

	var altPtr *string
	if altQuestions != "" {
		altPtr = &altQuestions
	}

	var tagsPtr *[]string
	if tags != "" {
		tagsList := strings.Split(tags, ",")
		for i, t := range tagsList {
			tagsList[i] = strings.TrimSpace(t)
		}
		tagsPtr = &tagsList
	}

	result, err := core.APIAdd(cfg, question, answer, altPtr, tagsPtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !result.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Message)
		os.Exit(1)
	}

	fmt.Printf("%s\n", result.Message)
}

func handleActionUpdate(args []string, cfg *core.Config) {
	var id, question, answer, altQuestions, tags, answerKey string
	var allowUnderReview bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				id = args[i+1]
				i++
			}
		case "--question":
			if i+1 < len(args) {
				question = args[i+1]
				i++
			}
		case "--answer":
			if i+1 < len(args) {
				answer = args[i+1]
				i++
			}
		case "--altquestions":
			if i+1 < len(args) {
				altQuestions = args[i+1]
				i++
			}
		case "--tags":
			if i+1 < len(args) {
				tags = args[i+1]
				i++
			}
		case "--answer-key":
			if i+1 < len(args) {
				answerKey = args[i+1]
				i++
			}
		case "--allow-under-review":
			allowUnderReview = true
		}
	}

	if id == "" {
		fmt.Fprintln(os.Stderr, "Error: --id is required for update")
		os.Exit(2)
	}

	if answerKey != "" && answer == "" {
		fmt.Fprintln(os.Stderr, "Error: --answer-key requires --answer")
		os.Exit(2)
	}

	if question == "" && answer == "" && altQuestions == "" && tags == "" {
		fmt.Fprintln(os.Stderr, "Error: at least one of --question, --answer, --altquestions, or --tags must be provided")
		os.Exit(2)
	}

	var qPtr, aqPtr, tPtr *string
	var answers []core.AnswerUpdate
	if question != "" {
		qPtr = &question
	}
	if answer != "" {
		answers = []core.AnswerUpdate{{Key: answerKey, Value: answer}}
	}
	if altQuestions != "" {
		aqPtr = &altQuestions
	}
	if tags != "" {
		tPtr = &tags
	}

	result, err := core.APIUpdate(cfg, id, qPtr, answers, aqPtr, tPtr, allowUnderReview)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !result.Success {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Message)
		os.Exit(1)
	}

	fmt.Printf("%s\n", result.Message)
}

func handleActionDelete(args []string, cfg *core.Config) {
	var idStr string
	permanent := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			if i+1 < len(args) {
				idStr = args[i+1]
				i++
			}
		case "--permanent":
			permanent = true
		}
	}

	if idStr == "" {
		fmt.Fprintln(os.Stderr, "Error: --id is required for delete")
		os.Exit(2)
	}

	var ids []string
	for _, id := range strings.Split(idStr, ",") {
		if strings.TrimSpace(id) != "" {
			ids = append(ids, strings.TrimSpace(id))
		}
	}

	result, err := core.APIDelete(cfg, ids, !permanent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(drawDeleteReport(ids, result.Deleted, result.NotDeleted))
}

// ---------------------------------------------------------------------------
// Config handling
// ---------------------------------------------------------------------------

func handleConfig(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: rfpio config init")
		os.Exit(2)
	}

	switch args[0] {
	case "init":
		path, err := core.CreateSampleConfig("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if path == "" {
			fmt.Println("Config file already exists.")
		} else {
			fmt.Printf("Created config file: %s\n", path)
			fmt.Println("Please edit it with your API token and site URL.")
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown config subcommand: %s\n", args[0])
		os.Exit(2)
	}
}
