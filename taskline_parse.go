package main

import (
	"os/exec"
	"strings"
)

// LoadLines runs the taskline command and parses its output into mainLines and taskSummary.
func LoadLines() (mainLines []string, taskSummary []string) {
	// unbuffer preserves taskline's colors
	cmd := exec.Command("unbuffer", "taskline")
	out, err := cmd.Output()
	if err != nil {
		return []string{"(error running taskline)"}, nil
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")

	// Always skip the first line and last two lines as they are empty or control codes
	if len(lines) > 3 {
		lines = lines[1 : len(lines)-2]
		mainLines = lines[:len(lines)-2]
		taskSummary = lines[len(lines)-2:]
	} else {
		mainLines = lines
		taskSummary = nil
	}
	return
}
