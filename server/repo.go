package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// создаем общий путь для файла с сохранением.
// Можно конечно получить имя файла как параметр в функции,
// но у нас нет цели как-то  экспортировать и использовать этот файл иначе чем save/load
const jsonfile string = "./storage/state.json"

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
	return &Repository{
		batches: make(map[int]*Batch),
		NextID:  1,
	}
}

func (rep *Repository) CreateBatch(urls []string) {
	rep.mtx.Lock()
	defer rep.mtx.Unlock()

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
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	if batch, exists := rep.batches[id]; exists {
		return batch, nil
	}

	return &Batch{}, errors.New("This id is not available")
}

func (rep *Repository) CheckBanchByID(id int) error {
	batch, err := rep.GetBanchByID(id)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(batch.Links))
	for i := 0; i < len(batch.Links); i++ {
		go func(index int) {
			defer wg.Done()
			batch.Links[index].CheckLink()
		}(i)
	}

	wg.Wait()
	return nil
}

// Для корректной перезагрузки я буду использовать сохранение текущих данных в json файл
//

func (rep *Repository) SaveState()
