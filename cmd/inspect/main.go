package main

import (
    "fmt"
    "log"
)

func main() {
    items, err := LoadItems()
    if err != nil {
        log.Fatal(err)
    }
    for _, item := range items {
        fmt.Printf("%d\t%s\n", item.ID, item.Description)
    }
}
