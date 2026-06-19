package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"visrosa/tasklineUI/internal/taskdata"
)

func TestRenderTaskLineHangingIndent(t *testing.T) {
	task := taskdata.Task{
		Id:          69,
		Description: "Map terminological boundaries: ideogram, pictogram, symbol, glyph, logogram, morphogram and any others with fuzzy boundaries noted (also informs project naming)",
	}

	const width = 60
	line := renderTaskLine(task, width)
	lines := strings.Split(line, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected description to wrap across multiple lines at width %d, got 1 line: %q", width, line)
	}

	prefix := taskIndex.Render("") +
		sectionHeader.Render("\x1b]66;n=1:d=2:v=2:w=2; 69 \a") +
		taskIndex.Render("🭬") +
		" " + logo.Render("󰏄") +
		" "
	prefixWidth := ansi.StringWidth(prefix)

	for i, l := range lines {
		if ansi.StringWidth(l) > width {
			t.Errorf("line %d exceeds width %d (got %d): %q", i, width, ansi.StringWidth(l), l)
		}
		if i == 0 {
			continue
		}
		got := 0
		for _, r := range l {
			if r != ' ' {
				break
			}
			got++
		}
		if got != prefixWidth {
			t.Errorf("continuation line %d: indent = %d spaces, want %d (prefix width): %q", i, got, prefixWidth, l)
		}
	}
}

func TestRenderTaskLineNarrowViewportNoPanic(t *testing.T) {
	task := taskdata.Task{Id: 1, Description: "short"}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderTaskLine panicked on narrow width: %v", r)
		}
	}()
	renderTaskLine(task, 1)
	renderTaskLine(task, 0)
	renderTaskLine(task, -5)
}
