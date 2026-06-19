package main

import (
	"fmt"

	"visrosa/tasklineUI/internal/taskdata"
)

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
		lines = append(lines, sectionHeader.Render(fmt.Sprintf("%s [%d]", g.project, len(g.tasks))))
		for _, t := range g.tasks {
			lines = append(lines, renderTaskLine(t))
		}
	}
	return lines
}

func renderTaskLine(t taskdata.Task) string {
	marker := "□"
	if t.Start != nil {
		marker = highlight.Render("●")
	}

	line := fmt.Sprintf("%2d. %s %s", t.Id, marker, t.Description)

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
