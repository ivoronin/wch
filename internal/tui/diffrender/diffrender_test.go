package diffrender

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/ivoronin/wch/internal/diff"
)

var testFg = ansi.RGBColor{R: 0x2E, G: 0x7D, B: 0x32}

const greenFg = "38;2;46;125;50" // truecolor SGR params for testFg (foreground)

// render runs the real path: diff on stripped text, overlay onto the styled new output.
func render(old, neu string) string {
	a := diff.Align(ansi.Strip(old), ansi.Strip(neu))
	return Render(a.Lines(), neu, testFg)
}

func TestRenderPlainChange(t *testing.T) {
	body := render("a b", "a c")
	if got := ansi.Strip(body); got != "a c" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "a c", body)
	}
	if !strings.Contains(body, greenFg) {
		t.Errorf("expected highlight, raw=%q", body)
	}
}

func TestRenderPreservesUnchangedFg(t *testing.T) {
	// whole line yellow; only "6" changes -> unchanged "load " keeps yellow fg, "6" turns green.
	body := render("\x1b[33mload 5\x1b[0m", "\x1b[33mload 6\x1b[0m")
	if got := ansi.Strip(body); got != "load 6" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "load 6", body)
	}
	if !strings.Contains(body, "33") {
		t.Errorf("yellow fg lost on unchanged text, raw=%q", body)
	}
	if !strings.Contains(body, greenFg) {
		t.Errorf("no highlight, raw=%q", body)
	}
}

func TestRenderPreservesBackground(t *testing.T) {
	// red background across the line; middle token changes -> bg untouched, "X" gets green fg.
	body := render("\x1b[41mA B C\x1b[0m", "\x1b[41mA X C\x1b[0m")
	if got := ansi.Strip(body); got != "A X C" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "A X C", body)
	}
	if !strings.Contains(body, "41") {
		t.Errorf("red background not preserved, raw=%q", body)
	}
	if !strings.Contains(body, greenFg) {
		t.Errorf("no highlight, raw=%q", body)
	}
}

func TestRenderCarriesColorAcrossLines(t *testing.T) {
	// red opened on line 1, carried to line 2 (no reset until the end); only line 2 changes.
	body := render("\x1b[31mfoo\nbar 5\x1b[0m", "\x1b[31mfoo\nbar 6\x1b[0m")
	if got := ansi.Strip(body); got != "foo\nbar 6" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "foo\nbar 6", body)
	}
	rows := strings.Split(body, "\n")
	if len(rows) != 2 || !strings.Contains(rows[1], "31") {
		t.Errorf("red not carried onto line 2, raw=%q", body)
	}
	if !strings.Contains(body, greenFg) {
		t.Errorf("no highlight, raw=%q", body)
	}
}

func TestRenderAddedLineFullyHighlighted(t *testing.T) {
	body := render("a", "a\nNEW")
	if got := ansi.Strip(body); got != "a\nNEW" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "a\nNEW", body)
	}
	rows := strings.Split(body, "\n")
	if len(rows) != 2 || !strings.Contains(rows[1], greenFg) {
		t.Errorf("added line not highlighted, raw=%q", body)
	}
}

func TestRenderColorOnlyChangeIsNotADiff(t *testing.T) {
	// same visible text, different color -> not a change; shown with the new color, no highlight.
	body := render("\x1b[31mERR\x1b[0m", "\x1b[32mERR\x1b[0m")
	if got := ansi.Strip(body); got != "ERR" {
		t.Fatalf("visible=%q want %q\nraw=%q", got, "ERR", body)
	}
	if strings.Contains(body, greenFg) {
		t.Errorf("color-only change must not be highlighted, raw=%q", body)
	}
	if !strings.Contains(body, "32") {
		t.Errorf("new color not shown, raw=%q", body)
	}
}
