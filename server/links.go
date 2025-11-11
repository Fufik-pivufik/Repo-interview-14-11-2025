package main

import (
	"time"
)

// В этом файле я описываю логику работы с самими ссылками
// Тип Link Нужен чтобы систематизировать информацию о каждой отдельной ссылке,
// к тому же, мы могли бы добавить новые данные о конкретной ссылке в таком случае
type Link struct {
	URL     string
	State   string // Имеет всего три состояния: unknown, available и unavailable
	Checked time.Time
}

func CreateLink(url string) *Link {
	return &Link{
		url,
		"unknown",
		time.Now(),
	}
}

// Тип Batch нужен чтобы мы могли обрабатывать конкретные пачки ссылок
// Именно с ним я и буду работать
type Batch struct {
	ID        int
	Links     []Link
	CreatedAt time.Time
}
