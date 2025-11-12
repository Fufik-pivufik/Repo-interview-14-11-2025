package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// создаем общий путь для файла с сохранением.
// Можно конечно получить имя файла как параметр в функции,
// но у нас нет цели как-то  экспортировать и использовать этот файл иначе чем save/load
const (
	jsonfile     string = "./storage/state.json"
	workersCount int    = 5
)

// В этом файле будет описана работа структуры для хранения
// всех доступных пачек ссылок
//
// Тип данных Repository я описываю так как нам нужно работать
// со всеми пачками ссылок. RWMutex я использую так как нам нет необходимости блокировать чтение данных
type Repository struct {
	mtx     sync.RWMutex
	Batches map[int]*Batch
	NextID  int

	tchan    chan *Task
	WGroup   sync.WaitGroup
	shutdown chan struct{}
}

type Task struct {
	BatchID   int
	LinkIndex int
	URL       string
}

func NewRepos() *Repository {
	rep := &Repository{
		Batches:  make(map[int]*Batch),
		NextID:   1,
		tchan:    make(chan *Task, 100),
		shutdown: make(chan struct{}),
	}

	rep.StartWorkers()
	return rep
}

func (rep *Repository) StartWorkers() {
	for i := 0; i < workersCount; i++ {
		rep.WGroup.Add(1)
		go rep.worker()
	}
}

func (rep *Repository) worker() {
	defer rep.WGroup.Done()
	for {
		select {
		case task := <-rep.tchan:
			rep.procTask(task)
		case <-rep.shutdown:
			fmt.Println("Working shutting down...")
			return
		}
	}
}

func (rep *Repository) Shutdown() {
	fmt.Println("Shutting down repository...")

	close(rep.shutdown)

	done := make(chan struct{})
	go func() {
		rep.WGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("All workers completed their tasks")
	case <-time.After(10 * time.Second):
		fmt.Println("Timeout waiting for workers to complete")
	}

	rep.drainTaskChannel()

	if err := rep.SaveState(); err != nil {
		fmt.Printf("Error saving final state: %v\n", err)
	} else {
		fmt.Println("Final state saved successfully")
	}
}

func (rep *Repository) drainTaskChannel() {
	close(rep.tchan)

	for task := range rep.tchan {
		rep.procTask(task)
		fmt.Printf("Processed remaining task for batch %d\n", task.BatchID)
	}
}

func (rep *Repository) procTask(task *Task) {
	tmplink := CreateLink(task.URL)
	tmplink.CheckLink()

	rep.updateLinkStat(task.BatchID, task.LinkIndex, tmplink.State, tmplink.Checked)
}

func (rep *Repository) updateLinkStat(batchID int, linkIndex int, state string, checked time.Time) {
	rep.mtx.Lock()
	defer rep.mtx.Unlock()

	if batch, exists := rep.Batches[batchID]; exists && linkIndex < len(batch.Links) {
		batch.Links[linkIndex].State = state
		batch.Links[linkIndex].Checked = checked
	}
}

func (rep *Repository) CreateBatch(urls []string) int {
	rep.mtx.Lock()
	defer rep.mtx.Unlock()

	if _, idExists := rep.Batches[rep.NextID]; idExists {
		fmt.Println("Error: cannot set banch for id: ", rep.NextID)
		return 0
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
	return rep.NextID - 1
}

func (rep *Repository) GetBanchByID(id int) (*Batch, error) {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	if batch, exists := rep.Batches[id]; exists {
		return batch, nil
	}

	return &Batch{}, errors.New("This id is not available")
}

func (rep *Repository) DeleteBanchByID(id int) error {
	if _, exists := rep.Batches[id]; exists {
		delete(rep.Batches, id)
		return nil
	}

	return errors.New("Cannot delete: element with this id does not exist")
}

func (rep *Repository) CheckBanchByID(id int) error {

	select {
	case <-rep.shutdown:
		return errors.New("service is shutting down")

	default:
	}

	batch, err := rep.GetBanchByID(id)
	if err != nil {
		return err
	}

	for i, link := range batch.Links {
		task := &Task{
			BatchID:   id,
			LinkIndex: i,
			URL:       link.URL,
		}

		select {
		case rep.tchan <- task:

		case <-rep.shutdown:
			return errors.New("service is shutting down")
		}
	}
	return nil
}

func (rep *Repository) IsBatchCompleted(batchID int) bool {
	rep.mtx.RLock()
	defer rep.mtx.RUnlock()

	batch, exists := rep.Batches[batchID]
	if !exists {
		return false
	}

	for _, link := range batch.Links {
		if link.State == "unknown" {
			return false
		}
	}
	return true
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

func (rep *Repository) GenerateReport(batchIDs []int) ([]byte, error) {
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
	var pdfBytes bytes.Buffer
	buffer := pdf.Output(&pdfBytes)
	if buffer != nil {
		return nil, fmt.Errorf("error generating PDF: %v", buffer)
	}

	return pdfBytes.Bytes(), nil

}

//
// Функция завершения работы
