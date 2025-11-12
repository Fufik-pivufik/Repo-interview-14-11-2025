package main

import (
	"fmt"
	"os"
)

func main() {

	if err := os.MkdirAll("./storage", 0755); err != nil {
		fmt.Println("Error create dir storage")
	}

	rep := NewRepos()

	err := rep.LoadState()
	if err != nil {
		fmt.Println("Cannot rload previous state, starting as first run")
	}
	defer func() {
		err = rep.SaveState()
		if err != nil {
			fmt.Println("failed to write json file: ", err)
		}
	}()

	server := NewServer(rep)
	fmt.Println("Server litening at 8080...")
	if err := server.Start("8080"); err != nil {
		fmt.Println("Server error: ", err)
	}
}
