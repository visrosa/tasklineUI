package main

import (
    "encoding/json"
    "os"
)

type Item struct {
    ID          int      `json:"id"`
    Date        string   `json:"date"`
    Timestamp   int64    `json:"timestamp"`
    IsTask      bool     `json:"isTask"`
    Description string   `json:"description"`
    IsStarred   bool     `json:"isStarred"`
    Boards      []string `json:"boards"`
    Priority    int      `json:"priority"`
    InProgress  bool     `json:"inProgress"`
    IsCanceled  bool     `json:"isCanceled"`
    IsComplete  bool     `json:"isComplete"`
    DueDate     int64    `json:"dueDate"`
    PassedTime  int64    `json:"passedTime"`
    LastStart   int64    `json:"lastStartTime"`
}

func StoragePath() string {
    base := os.Getenv("XDG_DATA_HOME")
    if base == "" {
        base = os.Getenv("HOME") + "/.local/share"
    }
    return base + "/.taskline/storage/storage.json"
}

func LoadItems() ([]Item, error) {
    data, err := os.ReadFile(StoragePath())
    if err != nil {
        return nil, err
    }
    var items []Item
    err = json.Unmarshal(data, &items)
    return items, err
}
