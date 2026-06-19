// Package taskdata reads taskwarrior's TaskChampion store directly and
// writes through the `task` CLI, so hooks and the operation log stay
// consistent on the write side while reads avoid a subprocess per refresh.
package taskdata

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Task struct {
	Id          int
	UUID        string
	Description string
	Project     string
	Status      string
	Priority    string
	Tags        []string
	Annotations []string
	Entry       *time.Time
	Modified    *time.Time
	Due         *time.Time
	Start       *time.Time
}

// StoragePath resolves the TaskChampion SQLite file the same way `task`
// itself resolves data.location: $TASKDATA, then the taskrc setting, then
// the XDG fallback.
func StoragePath() (string, error) {
	dir := os.Getenv("TASKDATA")
	if dir == "" {
		var err error
		dir, err = dataLocationFromTaskrc()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(expandHome(dir), "taskchampion.sqlite3"), nil
}

func dataLocationFromTaskrc() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	fallback := filepath.Join(home, ".local", "share", "task")

	data, err := os.ReadFile(taskrcPath(home))
	if err != nil {
		// No taskrc found at any resolved location — fall back rather than error,
		// since a missing config file is a valid (if unconfigured) state.
		return fallback, nil
	}

	// Scans only the top-level taskrc file; `include`d files aren't followed.
	// Not a concern for this user's config (only commented-out theme includes).
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "data.location" {
			continue
		}
		return strings.TrimSpace(value), nil
	}
	return fallback, nil
}

func taskrcPath(home string) string {
	if rc := os.Getenv("TASKRC"); rc != "" {
		return expandHome(rc)
	}
	if legacy := filepath.Join(home, ".taskrc"); fileExists(legacy) {
		return legacy
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "task", "taskrc")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

// LoadTasks reads every pending task directly from the TaskChampion store,
// opened read-only and closed before returning.
func LoadTasks() ([]Task, error) {
	path, err := StoragePath()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	workingSet := make(map[string]int)
	wsRows, err := db.Query("SELECT id, uuid FROM working_set")
	if err != nil {
		return nil, err
	}
	for wsRows.Next() {
		var id int
		var uuid string
		if err := wsRows.Scan(&id, &uuid); err != nil {
			wsRows.Close()
			return nil, err
		}
		workingSet[uuid] = id
	}
	wsRows.Close()

	rows, err := db.Query("SELECT uuid, data FROM tasks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var uuid, data string
		if err := rows.Scan(&uuid, &data); err != nil {
			return nil, err
		}
		// The property bag is a dynamic key set (tag_*/annotation_* keys aren't
		// fixed fields), so decode into a map first rather than a typed struct.
		fields := map[string]string{}
		if err := json.Unmarshal([]byte(data), &fields); err != nil {
			return nil, fmt.Errorf("parsing task %s: %w", uuid, err)
		}
		if fields["status"] != "pending" {
			continue
		}
		tasks = append(tasks, taskFromFields(uuid, workingSet[uuid], fields))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Project != tasks[j].Project {
			return tasks[i].Project < tasks[j].Project
		}
		return tasks[i].Id < tasks[j].Id
	})

	return tasks, nil
}

func taskFromFields(uuid string, id int, fields map[string]string) Task {
	t := Task{
		Id:          id,
		UUID:        uuid,
		Description: fields["description"],
		Project:     fields["project"],
		Status:      fields["status"],
		Priority:    fields["priority"],
		Entry:       parseEpoch(fields["entry"]),
		Modified:    parseEpoch(fields["modified"]),
		Due:         parseEpoch(fields["due"]),
		Start:       parseEpoch(fields["start"]),
	}

	if tags := fields["tags"]; tags != "" {
		t.Tags = strings.Split(tags, ",")
	}

	type annotation struct{ key, text string }
	var annotations []annotation
	for key, value := range fields {
		if strings.HasPrefix(key, "annotation_") {
			annotations = append(annotations, annotation{key, value})
		}
	}
	sort.Slice(annotations, func(i, j int) bool { return annotations[i].key < annotations[j].key })
	for _, a := range annotations {
		t.Annotations = append(t.Annotations, a.text)
	}

	return t
}

func parseEpoch(value string) *time.Time {
	if value == "" {
		return nil
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(seconds, 0)
	return &t
}

// AddTask creates a new pending task through the `task` CLI — writes never
// touch the SQLite file directly, so hooks and the operation log stay correct.
func AddTask(description, project string) error {
	args := []string{"add"}
	if project != "" {
		args = append(args, "project:"+project)
	}
	args = append(args, description)
	return exec.Command("task", args...).Run()
}
