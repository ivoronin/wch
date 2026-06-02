package recording

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsBadFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	if err := os.WriteFile(path, []byte(`{"format":"other","version":1,"command":"x","interval":"1s"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Errorf("expected Load to reject unknown format")
	}
}

func TestLoadRejectsBadVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.jsonl")
	if err := os.WriteFile(path, []byte(`{"format":"wch-history","version":99,"command":"x","interval":"1s"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Errorf("expected Load to reject unsupported version")
	}
}

func TestLoadRejectsBadInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-interval.jsonl")
	if err := os.WriteFile(path, []byte(`{"format":"wch-history","version":1,"command":"x","interval":"not-a-duration"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Errorf("expected Load to reject bad interval")
	}
}
