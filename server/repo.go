package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// В этом файле будет описана работа структуры для хранения
// всех доступных пачек ссылок
//
// Тип данных Repository я описываю так как нам нужно работать
// со всеми пачками ссылок. RWMutex я использую так как нам нет необходимости блокировать чтение данных
type Repository struct {
	mtx     sync.RWMutex
	batches map[int]*Batch
	NextID  int
}

func NewRepos() *Repository {
	var mx sync.RWMutex
	return &Repository{
		mx,
		make(map[int]*Batch),
		1,
	}
}

func (rep *Repository) CreateBatch(urls []string) {
	if _, idExists := rep.batches[rep.NextID]; idExists {
		fmt.Println("Error: cannot set banch for id: ", rep.NextID)
		return
	}
	var links []Link
	for _, url := range urls {
		link := CreateLink(url)
		links = append(links, *link)
	}

	rep.batches[rep.NextID] = &Batch{
		rep.NextID,
		links,
		time.Now(),
	}
	rep.NextID++
}

func (rep *Repository) GetBanchByID(id int) (*Batch, error) {
	if batch, exists := rep.batches[id]; exists {
		return batch, nil
	}

	return &Batch{}, errors.New("Cannot this id is not available")
}
