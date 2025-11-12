package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jung-kurt/gofpdf"
	"os"
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
	Batches map[int]*Batch
	NextID  int
}

func NewRepos() *Repository {
	return &Repository{
		Batches: make(map[int]*Batch),
		NextID:  1,
	}
}

func (rep *Repository) CreateBatch(urls []string) {
	rep.mtx.Lock()
	defer rep.mtx.Unlock()

	if _, idExists := rep.Batches[rep.NextID]; idExists {
		fmt.Println("Error: cannot set banch for id: ", rep.NextID)
		return
	}
	var links []Link
	for _, url := range urls {
		link := CreateLink(url)
		links = append(links, *link)
	}

	rep.Batches[rep.NextID] = &Batch{
		rep.NextID,
		links,
		time.Now(),
	}
	rep.NextID++
}

func (rep *Repository) GetBanchByID(id int) (*Batch, error) {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	if batch, exists := rep.Batches[id]; exists {
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

func (rep *Repository) SaveState() error {
	jsonData, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(jsonfile, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (rep *Repository) LoadState() error {
	data, err := os.ReadFile(jsonfile)
	if err != nil {
		return err
	}

	var snap Repository
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}

	rep.mtx.Lock()
	defer rep.mtx.Unlock()

	rep.Batches = snap.Batches
	rep.NextID = snap.NextID
	return nil
}

// Работаем с pdf с помощью gofpdf

func (rep *Repository) GenerateReport(batchIDs []int) error {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Link Status Report")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(15)

	for _, batchID := range batchIDs {
		batch, exists := rep.Batches[batchID]
		if !exists {
			continue
		}

		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(40, 10, fmt.Sprintf("Batch #%d (Created: %s)", batch.ID, batch.CreatedAt.Format("2006-01-02")))
		pdf.Ln(10)

		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(100, 10, "URL")
		pdf.Cell(40, 10, "Status")
		pdf.Cell(40, 10, "Checked At")
		pdf.Ln(10)

		pdf.SetFont("Arial", "", 10)
		for _, link := range batch.Links {
			url := link.URL
			if len(url) > 60 {
				url = url[:57] + "..."
			}

			pdf.Cell(100, 8, url)

			if link.State == "available" {
				pdf.SetTextColor(0, 128, 0)
			} else {
				pdf.SetTextColor(255, 0, 0)
			}
			pdf.Cell(40, 8, link.State)

			pdf.SetTextColor(0, 0, 0)
			pdf.Cell(40, 8, link.Checked.Format("15:04:05"))
			pdf.Ln(8)
		}

		pdf.Ln(10)
	}

	pdfname := "./reports/report_" + time.Now().Format("DD_MM_YYYY__HH_MM_SS") + ".pdf"

	err := pdf.OutputFileAndClose(pdfname)
	if err != nil {
		return err
	}
	return nil
}
