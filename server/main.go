package main

import (
	"fmt"
)

func main() {
	repo := NewRepos()
	repo.CreateBatch([]string{"https://google.com", "https://ya.ru", "https://invalid.url"})
	fmt.Printf("repo's next batch id: %d\n", repo.NextID)
	fmt.Println(repo.batches[1])

	repo.CheckBanchByID(1)
	fmt.Println(repo.batches[1])
}
