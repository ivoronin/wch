package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("os.UserHomeDir not available: %v", err)
	}

	cases := []struct {
		in, want string
	}{
		{"~", home},
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"~user/foo", "~user/foo"},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"$VAR/foo", "$VAR/foo"},
		{"", ""},
	}
	for _, c := range cases {
		got, err := ExpandTilde(c.in)
		if err != nil {
			t.Errorf("ExpandTilde(%q) returned error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ExpandTilde(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
