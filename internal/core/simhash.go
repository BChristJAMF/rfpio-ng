package core

import (
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
)

const mask64 = (1 << 64) - 1

// simhash64 computes a 64-bit SimHash over HTML-stripped, normalized text.
// Returns an unsigned 64-bit integer. Two fingerprints can be compared via
// Hamming distance: fewer differing bits means more similar answers.
func simhash64(text string) uint64 {
	if text == "" {
		text = ""
	}

	// Normalize and tokenize. (text may already be cleaned)
	t := strings.ToLower(text)
	tokenRe := regexp.MustCompile(`[\w]+`)
	tokens := tokenRe.FindAllString(t, -1)

	if len(tokens) == 0 {
		tokens = []string{t}
		if t == "" {
			tokens = []string{""}
		}
	}

	// Count tokens.
	counts := make(map[string]int)
	for _, tok := range tokens {
		counts[tok]++
	}

	// Build 64-bit vector.
	vector := make([]int64, 64)
	for token, weight := range counts {
		h := sha1.Sum([]byte(token))
		hv := uint64(h[0])<<56 | uint64(h[1])<<48 | uint64(h[2])<<40 | uint64(h[3])<<32 |
			uint64(h[4])<<24 | uint64(h[5])<<16 | uint64(h[6])<<8 | uint64(h[7])

		for i := 0; i < 64; i++ {
			if (hv>>i)&1 == 1 {
				vector[i] += int64(weight)
			} else {
				vector[i] -= int64(weight)
			}
		}
	}

	// Build fingerprint.
	fp := uint64(0)
	for i := 0; i < 64; i++ {
		if vector[i] > 0 {
			fp |= (1 << i)
		}
	}

	return fp & mask64
}

// hammingDistance computes the Hamming distance between two 64-bit SimHash values.
func hammingDistance(a, b uint64) int {
	x := a ^ b
	count := 0
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

// crc64 computes a simple 64-bit hash for general use (not SimHash).
func crc64hash(s string) uint64 {
	return uint64(crc32.ChecksumIEEE([]byte(s)))
}

// wordCount returns the number of words in text (assumed already HTML-cleaned).
func wordCount(text string) int {
	fields := strings.Fields(text)
	return len(fields)
}

// fmtUint64 formats a uint64 as a decimal string, matching Python's format.
func fmtUint64(v uint64) string {
	return fmt.Sprintf("%d", v)
}
