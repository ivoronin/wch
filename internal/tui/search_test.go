package tui

import (
	"reflect"
	"strings"
	"testing"
)

func TestFindMatchesSmartCase(t *testing.T) {
	body := "pod-01 Running\npod-02 Pending\npod-03 PENDING\npod-04 pending\n"

	// All-lowercase query → case-insensitive, picks up Pending/PENDING/pending.
	got := findMatches(body, "pending")
	want := []searchMatch{
		{line: 1, col: 7, length: 7},
		{line: 2, col: 7, length: 7},
		{line: 3, col: 7, length: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("smart-case lowercase: got %v want %v", got, want)
	}

	// Mixed-case query → exact match only.
	got = findMatches(body, "Pending")
	want = []searchMatch{{line: 1, col: 7, length: 7}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("smart-case exact: got %v want %v", got, want)
	}
}

func TestFindMatchesMultiplePerLine(t *testing.T) {
	got := findMatches("foo bar foo bar foo", "foo")
	want := []searchMatch{
		{line: 0, col: 0, length: 3},
		{line: 0, col: 8, length: 3},
		{line: 0, col: 16, length: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestFindMatchesEmptyAndNoMatch(t *testing.T) {
	if findMatches("anything", "") != nil {
		t.Errorf("empty query must yield nil matches")
	}
	if findMatches("alpha\nbeta\n", "gamma") != nil {
		t.Errorf("no-occurrence query must yield nil matches")
	}
	if findMatches("", "foo") != nil {
		t.Errorf("empty body must yield nil matches")
	}
}

func TestFindMatchesWideRunes(t *testing.T) {
	// Each CJK rune is 2 display cells. "緑Running" -> col of "Running" is 2 (after one wide rune).
	body := "緑Running\nRunning"
	got := findMatches(body, "Running")
	want := []searchMatch{
		{line: 0, col: 2, length: 7},
		{line: 1, col: 0, length: 7},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wide rune cols: got %v want %v", got, want)
	}
}

func TestFindMatchesAcrossLines(t *testing.T) {
	body := strings.Repeat("alpha\n", 3) + "alphabeta"
	got := findMatches(body, "alpha")
	want := []searchMatch{
		{line: 0, col: 0, length: 5},
		{line: 1, col: 0, length: 5},
		{line: 2, col: 0, length: 5},
		{line: 3, col: 0, length: 5},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}
