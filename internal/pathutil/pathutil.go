// Package pathutil holds small filesystem-path helpers shared between the CLI and the TUI.
package pathutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandTilde replaces a leading "~" or "~/" with the user's home directory. Other forms
// (including "~user/...") pass through unchanged. The original input is returned alongside
// the error on home-lookup failure so callers can choose between aborting and proceeding
// with the literal path.
func ExpandTilde(p string) (string, error) {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p, err
	}
	if p == "~" {
		return home, nil
	}
	return filepath.Join(home, p[2:]), nil
}
