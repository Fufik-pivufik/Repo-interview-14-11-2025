package main

import (
	"fmt"
)

func main() {
	repo := NewRepos()
	repo.CreateBatch([]string{"https://google.com", "https://ya.ru", "https://invalid.url"})
	fmt.Printf("repo's next batch id: %d\n", repo.NextID)
}
