package recording

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ivoronin/wch/internal/session"
)

func TestNormalizePathRejectsEmpty(t *testing.T) {
	if _, err := NormalizePath(""); err == nil {
		t.Errorf("empty input should error")
	}
	if _, err := NormalizePath("   "); err == nil {
		t.Errorf("whitespace-only input should error")
	}
}

func TestNormalizePathTrimsAndPassesThrough(t *testing.T) {
	got, err := NormalizePath("  /tmp/foo.jsonl  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "/tmp/foo.jsonl" {
		t.Errorf("got %q want %q", got, "/tmp/foo.jsonl")
	}
}

func TestNormalizePathExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	got, err := NormalizePath("~/recordings/x.jsonl")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := filepath.Join(home, "recordings/x.jsonl")
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestPreflightCheckMissingFileOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.jsonl")
	if err := PreflightCheck(path); err != nil {
		t.Errorf("missing file should pass: %v", err)
	}
}

func TestPreflightCheckExistingFileFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "present.jsonl")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PreflightCheck(path)
	if err == nil {
		t.Fatal("existing file should fail PreflightCheck")
	}
	if !errors.Is(err, ErrPathExists) {
		t.Errorf("err = %v want ErrPathExists", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("err message should include path; got %q", err.Error())
	}
}

func TestSanitizeCommand(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"kubectl get pods -A", "kubectl_get_pods_A"},
		{"", ""},
		{"   ", ""},
		{"---a---", "a"},
		{"cat /tmp/foo | grep bar", "cat_tmp_foo_grep_bar"},
		{"df -h | awk '{print $1}'", "df_h_awk_print_1"},
		{"top", "top"},
		// Runs of non-alphanumerics collapse to a single underscore.
		{"a  b\t\tc", "a_b_c"},
	}
	for _, c := range cases {
		got := sanitizeCommand(c.in)
		if got != c.want {
			t.Errorf("sanitizeCommand(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	long := strings.Repeat("a", 200)
	got := sanitizeCommand(long)
	if len(got) != maxSanitizedCommandLen {
		t.Errorf("sanitizeCommand(200×'a') length = %d, want %d", len(got), maxSanitizedCommandLen)
	}
	if strings.HasSuffix(got, "_") {
		t.Errorf("sanitizeCommand of long alpha left a trailing underscore: %q", got)
	}

	// Truncated tail that lands on a run of separators should also drop trailing underscores.
	mix := strings.Repeat("a", maxSanitizedCommandLen-2) + "--tail"
	got = sanitizeCommand(mix)
	if strings.HasSuffix(got, "_") {
		t.Errorf("sanitizeCommand(%q) left trailing underscore: %q", mix, got)
	}
}

func TestDefaultFilename(t *testing.T) {
	when := time.Date(2026, 5, 30, 15, 30, 45, 0, time.Local)

	got := DefaultFilename("kubectl get pods -A", when)
	want := "kubectl_get_pods_A_20260530-153045.wch.jsonl"
	if got != want {
		t.Errorf("DefaultFilename(kubectl, ...) = %q, want %q", got, want)
	}

	got = DefaultFilename("", when)
	want = "wch_20260530-153045.wch.jsonl"
	if got != want {
		t.Errorf("DefaultFilename(empty, ...) = %q, want %q", got, want)
	}
}

// newFlowWith injects a recorder factory so Flow.Start can be exercised against an
// InMemoryRecorder rather than the on-disk JSONL one.
func newFlowWith(s *session.Session, factory func(path string) (session.Recorder, error)) *Flow {
	f := New(s)
	f.factory = factory
	return f
}

func TestFlowStartSuccessOnInMemory(t *testing.T) {
	s := session.NewSession("cmd", time.Second)
	rec := NewInMemoryRecorder()
	flow := newFlowWith(s, func(path string) (session.Recorder, error) { return rec, nil })

	if err := flow.Start("/tmp/anything.jsonl"); err != nil {
		t.Fatalf("Start err = %v", err)
	}
	if !flow.IsActive() {
		t.Errorf("IsActive() should be true")
	}
	if rec.Header().Command != "cmd" {
		t.Errorf("recorder Header.Command=%q want cmd", rec.Header().Command)
	}
}

func TestFlowStartFileExists(t *testing.T) {
	s := session.NewSession("cmd", time.Second)
	flow := newFlowWith(s, func(path string) (session.Recorder, error) {
		return nil, os.ErrExist
	})
	err := flow.Start("/tmp/existing.jsonl")
	if !errors.Is(err, ErrPathExists) {
		t.Errorf("err=%v want wraps ErrPathExists", err)
	}
	if flow.IsActive() {
		t.Errorf("IsActive() should be false after ErrPathExists")
	}
}

func TestFlowStartIOError(t *testing.T) {
	s := session.NewSession("cmd", time.Second)
	boom := errors.New("disk explode")
	flow := newFlowWith(s, func(path string) (session.Recorder, error) { return nil, boom })
	err := flow.Start("/tmp/err.jsonl")
	if !errors.Is(err, boom) {
		t.Errorf("err=%v want wraps boom", err)
	}
	if errors.Is(err, ErrPathExists) {
		t.Errorf("err should not match ErrPathExists")
	}
}

func TestFlowStopIdempotent(t *testing.T) {
	s := session.NewSession("cmd", time.Second)
	rec := NewInMemoryRecorder()
	flow := newFlowWith(s, func(path string) (session.Recorder, error) { return rec, nil })
	if err := flow.Start("/tmp/x.jsonl"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := flow.Stop(); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := flow.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
	if flow.IsActive() {
		t.Errorf("IsActive() should be false after Stop")
	}
}

func TestFlowStartDefaultFactoryIsJSONL(t *testing.T) {
	s := session.NewSession("cmd", time.Second)
	flow := New(s)
	path := filepath.Join(t.TempDir(), "y.jsonl")
	if err := flow.Start(path); err != nil {
		t.Fatalf("Start err = %v", err)
	}
	if err := flow.Stop(); err != nil {
		t.Fatal(err)
	}
}
