package textstats_test

import (
	"reflect"
	"testing"

	"example.com/ci-demo/internal/textstats"
)

func TestTokenize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", []string{}},
		{"lowercases", "Go GO go", []string{"go", "go", "go"}},
		{"strips punctuation", "hello, world! hello.", []string{"hello", "world", "hello"}},
		{"keeps digits", "http2 is fast", []string{"http2", "is", "fast"}},
		{"collapses separators", "a---b\n\nc", []string{"a", "b", "c"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := textstats.Tokenize(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Tokenize(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestAnalyzeCounts(t *testing.T) {
	got := textstats.Analyze("the quick brown fox jumps over the lazy dog the end", 3)

	if got.Words != 11 {
		t.Errorf("Words = %d, want 11", got.Words)
	}
	if got.Unique != 9 {
		t.Errorf("Unique = %d, want 9", got.Unique)
	}
	if len(got.Top) != 3 {
		t.Fatalf("len(Top) = %d, want 3", len(got.Top))
	}
	if got.Top[0] != (textstats.Entry{Word: "the", Count: 3}) {
		t.Errorf("Top[0] = %+v, want {the 3}", got.Top[0])
	}
}

func TestAnalyzeTiesAreAlphabetical(t *testing.T) {
	got := textstats.Analyze("beta alpha", 2)
	if got.Top[0].Word != "alpha" || got.Top[1].Word != "beta" {
		t.Errorf("ties not broken alphabetically: %+v", got.Top)
	}
}

func TestAnalyzeClampsN(t *testing.T) {
	if got := textstats.Analyze("a b", 99); len(got.Top) != 2 {
		t.Errorf("len(Top) = %d, want 2", len(got.Top))
	}
	if got := textstats.Analyze("a b", -1); len(got.Top) != 0 {
		t.Errorf("len(Top) = %d, want 0", len(got.Top))
	}
}

func TestAnalyzeEmpty(t *testing.T) {
	got := textstats.Analyze("   \n\t ", 5)
	if got.Words != 0 || got.Unique != 0 || len(got.Top) != 0 {
		t.Errorf("empty text produced %+v", got)
	}
}
