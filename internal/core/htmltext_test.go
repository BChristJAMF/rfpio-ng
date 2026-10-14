package core

import "testing"

func TestHTMLToTextTrimsBlankLinesAtBothEnds(t *testing.T) {
	cases := map[string]string{
		"<div>Hello <b>world</b></div>":           "Hello **world**",
		"<div>a</div><div>b</div>":                "a\n\nb",
		"<p>one</p>":                              "one",
		"<div><div><div>nested</div></div></div>": "nested",
		"plain": "plain",
		"":      "",
	}
	for in, want := range cases {
		if got := htmlToText(in); got != want {
			t.Errorf("htmlToText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTrimmedTextKeepsFingerprintAndWordCount(t *testing.T) {
	// simhash64 and wordCount look only at words, so the leading blank line a
	// leading <div> used to produce never affected them.
	padded := "\nSample answer text with several words."
	trimmed := "Sample answer text with several words."
	if simhash64(padded) != simhash64(trimmed) {
		t.Error("simhash changed when the leading newline was removed")
	}
	if wordCount(padded) != wordCount(trimmed) {
		t.Error("word count changed when the leading newline was removed")
	}
}
