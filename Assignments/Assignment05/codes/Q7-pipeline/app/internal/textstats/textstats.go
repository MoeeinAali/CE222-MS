// Package textstats computes simple word statistics for a piece of text.
package textstats

import (
	"sort"
	"strings"
	"unicode"
)

// Entry is a single word and how many times it occurred.
type Entry struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// Summary is the full result of analysing a text.
type Summary struct {
	Words  int     `json:"words"`
	Unique int     `json:"unique"`
	Top    []Entry `json:"top"`
}

// Tokenize splits text into lower-cased words. Anything that is not a letter
// or a digit acts as a separator, so punctuation never becomes part of a word.
func Tokenize(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, strings.ToLower(f))
	}
	return out
}

// Analyze returns the word count, the number of distinct words and the n most
// frequent ones. Ties are broken alphabetically so the output is deterministic.
func Analyze(text string, n int) Summary {
	words := Tokenize(text)

	counts := make(map[string]int, len(words))
	for _, w := range words {
		counts[w]++
	}

	entries := make([]Entry, 0, len(counts))
	for w, c := range counts {
		entries = append(entries, Entry{Word: w, Count: c})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count != entries[j].Count {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].Word < entries[j].Word
	})

	if n < 0 {
		n = 0
	}
	if n > len(entries) {
		n = len(entries)
	}

	return Summary{
		Words:  len(words),
		Unique: len(counts),
		Top:    entries[:n],
	}
}
