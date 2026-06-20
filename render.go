package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"visrosa/tasklineUI/internal/taskdata"
)

// prefixGlyphFudge corrects for Nerd Font/box-drawing glyphs in the task-line
// prefix rendering wider on screen (this font, kitty) than ansi.StringWidth's
// Unicode-table-based measurement reports. Adjust if it drifts visibly out of
// alignment again — there's no formula behind the number, it's eyeballed.
const prefixGlyphFudge = 2

var (
	codeSpanRe    = regexp.MustCompile("(^|[^`])`([^`]+)`")
	wikilinkRe    = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	boldRe        = regexp.MustCompile(`(^|[^*])\*\*([^*]+)\*\*`)
	italicUnderRe = regexp.MustCompile(`(^|[^_])__([^_]+)__`)
	italicStarRe  = regexp.MustCompile(`(^|[^*])\*([^*]+)\*`)
)

// renderMarkdown applies a small set of inline markdown-like substitutions to
// task description text: backtick code spans and [[wikilinks]] dim, **bold**
// bold, *italic*/__italic__ italic. Order matters — bold must consume
// **text** before the single-* pass runs, or it'd see two adjacent *text*
// spans instead. Single-underscore _text_ is deliberately left untouched:
// real task data uses it to quote identifiers (_add_, _v_), not to signal
// emphasis. Each pass captures the character preceding the marker (or start
// of string) so a markdown construct at position 0 still matches.
func renderMarkdown(text string) string {
	text = replaceStyled(text, codeSpanRe, codeStyle)
	text = replaceStyled(text, wikilinkRe, dim)
	text = replaceStyled(text, boldRe, bold)
	text = replaceStyled(text, italicUnderRe, italic)
	text = replaceStyled(text, italicStarRe, italic)
	return text
}

func replaceStyled(text string, re *regexp.Regexp, style lipgloss.Style) string {
	return re.ReplaceAllStringFunc(text, func(m string) string {
		g := re.FindStringSubmatch(m)
		if len(g) == 2 {
			return style.Render(g[1])
		}
		return g[1] + style.Render(g[2])
	})
}

type projectGroup struct {
	project string
	tasks   []taskdata.Task
}

// groupByProject buckets consecutive tasks sharing a project. Tasks must
// already be sorted by Project (taskdata.LoadTasks guarantees this), so this
// is a single pass, not a sort. Project-less tasks land under "(none)".
func groupByProject(tasks []taskdata.Task) []projectGroup {
	var groups []projectGroup
	for _, t := range tasks {
		label := t.Project
		if label == "" {
			label = "(none)"
		}
		if len(groups) == 0 || groups[len(groups)-1].project != label {
			groups = append(groups, projectGroup{project: label})
		}
		groups[len(groups)-1].tasks = append(groups[len(groups)-1].tasks, t)
	}
	return groups
}

func renderTasks(tasks []taskdata.Task, width int) []string {
	if len(tasks) == 0 {
		return []string{greyed.Render("No pending tasks.")}
	}

	var lines []string
	for i, g := range groupByProject(tasks) {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, 
			taskIndex.Render("🭁") +
			sectionHeader.Render(fmt.Sprintf(" %-25s", g.project)) +
			taskIndex.Render("◗"))
		for _, t := range g.tasks {
			lines = append(lines, renderTaskLine(t, width))
		}
	}
	return lines
}

// renderTaskLine wraps the description (plus priority/due suffix) on its own,
// against width minus the prefix's rendered width, so continuation lines can
// be re-indented to align under the description instead of resetting to
// column 0 — wrapping the whole assembled line in one global pass (as
// tasklineUI.go used to do) has no notion of a per-line prefix to preserve.
func renderTaskLine(t taskdata.Task, width int) string {
// 󱓼 U+f14fc Square rounded outline and
// 󱓻 U+f14fb Square rounded instead of the octagons;
// 󱨇 and 󱨈 as Square rounded badge icons, useful for edited tasks maybe
	marker := "󰏄"
	if t.Start != nil {
		marker = highlight.Render("󰏃")
	}
	prefix := taskIndex.Render("") +
		sectionHeader.Render(fmt.Sprintf("\x1b]66;n=1:d=2:v=2:w=2; %2d \a", t.Id)) +
		taskIndex.Render("🭬") +
		" " + logo.Render(marker) +
		" "

	body := renderMarkdown(t.Description)
	if t.Priority != "" {
		body += " " + greyed.Render("("+t.Priority+")")
	}
	if t.Due != nil {
		body += " " + greyed.Render("due:"+t.Due.Format("2006-01-02"))
	}

	// ansi.StringWidth measures the prefix's Nerd Font/box-drawing glyphs
	// (🭬, the in-progress marker) at 1 column each per standard Unicode
	// width tables, but kitty renders them wider on screen with this font —
	// a font-metric reality no width library captures. prefixGlyphFudge is
	// an empirical correction, not a derived value.
	prefixWidth := ansi.StringWidth(prefix) + prefixGlyphFudge
	wrapWidth := width - prefixWidth
	if wrapWidth < 1 {
		wrapWidth = 1
	}

	wrapped := strings.Split(ansi.Wordwrap(body, wrapWidth, ""), "\n")
	indent := strings.Repeat(" ", prefixWidth)

	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(wrapped[0])
	for _, l := range wrapped[1:] {
		b.WriteString("\n")
		b.WriteString(indent)
		b.WriteString(l)
	}
	return b.String()
}

// summaryLines is derived entirely from the already-loaded pending set —
// no completed-task query, since v1 doesn't track done/total per project.
func summaryLines(tasks []taskdata.Task) []string {
	if len(tasks) == 0 {
		return nil
	}
	active := 0
	for _, t := range tasks {
		if t.Start != nil {
			active++
		}
	}
	return []string{fmt.Sprintf("%d pending · %d in-progress", len(tasks), active)}
}
