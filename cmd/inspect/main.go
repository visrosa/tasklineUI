package main

import (
	"fmt"
	"log"

	"visrosa/tasklineUI/internal/taskdata"
)

func main() {
	tasks, err := taskdata.LoadTasks()
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range tasks {
		fmt.Printf("%d\t%s\t%s\n", t.Id, t.Project, t.Description)
	}
}
