package main

import (
	"errors"
	"net/http"
	"time"
)

// В этом файле я описываю логику работы с самими ссылками
// Тип Link Нужен чтобы систематизировать информацию о каждой отдельной ссылке,
// к тому же, мы могли бы добавить новые данные о конкретной ссылке в таком случае
type Link struct {
	URL     string    `json:"url"`
	State   string    `json:"state"` // Имеет всего три состояния: unknown, available и unavailable. Лучше использовать bool но у нас три состояния
	Checked time.Time `json:"checked at"`
}

func CreateLink(url string) *Link {
	return &Link{
		url,
		"unknown",
		time.Now(),
	}
}

func (link *Link) CheckLink() {

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("Error: too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Head(link.URL)
	if err != nil {
		link.State = "unavailable"
		return
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		link.State = "available"
	case resp.StatusCode >= 300 && resp.StatusCode < 400: // пусть редиректы будут считаться available
		link.State = "available"
	default:
		link.State = "unavailable"
	}
	link.Checked = time.Now()
}

// Тип Batch нужен чтобы мы могли обрабатывать конкретные пачки ссылок
// Именно с ним я и буду работать
type Batch struct {
	ID        int       `json:"id"`
	Links     []Link    `json:"links"`
	CreatedAt time.Time `json:"created at"`
}
