package main

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/lipgloss"

	"visrosa/tasklineUI/internal/taskdata"
)

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

func renderTasks(tasks []taskdata.Task) []string {
	if len(tasks) == 0 {
		return []string{greyed.Render("No pending tasks.")}
	}

	var lines []string
	for i, g := range groupByProject(tasks) {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, sectionHeader.Render(fmt.Sprintf("%-28s", g.project)))
		for _, t := range g.tasks {
			lines = append(lines, renderTaskLine(t))
		}
	}
	return lines
}

func renderTaskLine(t taskdata.Task) string {
	marker := "󰏄"
	if t.Start != nil {
		marker = highlight.Render("󰏃")
	}
	line := dim.Render(fmt.Sprintf("\x1b]66;n=1:d=2:v=2:w=1;%2d\a", t.Id)) +
	" " + marker +
	" " + renderMarkdown(t.Description)

	if t.Priority != "" {
		line += " " + greyed.Render("("+t.Priority+")")
	}
	if t.Due != nil {
		line += " " + greyed.Render("due:"+t.Due.Format("2006-01-02"))
	}

	return line
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
