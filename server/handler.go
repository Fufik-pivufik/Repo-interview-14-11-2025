package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"time"
)

// Структуры для HTTP запросов/ответов
type CheckLinksRequest struct {
	URLs []string `json:"urls"`
}

type CheckLinksResponse struct {
	BatchID int    `json:"batch_id"`
	Status  string `json:"status"`          // "processing" или "completed"
	Links   []Link `json:"links,omitempty"` // опционально - если быстро проверили
}

type ReportRequest struct {
	BatchIDs []int `json:"batch_ids"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Server struct {
	rep *Repository
}

func (s *Server) checkLinksHandler(wt http.ResponseWriter, r *http.Request) {
	var req CheckLinksRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(wt, "invalid json", http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(wt, "URL list is empty", http.StatusBadRequest)
		return
	}

	batchID := s.rep.CreateBatch(req.URLs)
	if err := s.rep.CheckBanchByID(batchID); err != nil {
		http.Error(wt, err.Error(), http.StatusBadRequest)
		return
	}

	batch, err := s.rep.GetBanchByID(batchID)
	if err != nil {
		http.Error(wt, err.Error(), http.StatusBadRequest)
		return
	}

	wt.Header().Set("Content-Type", "application/json")
	json.NewEncoder(wt).Encode(CheckLinksResponse{
		BatchID: batchID,
		Status:  "in process",
		Links:   batch.Links,
	})
}

func (s *Server) reportHandler(wt http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(wt, "invalid json", http.StatusBadRequest)
		return
	}

	if len(req.BatchIDs) == 0 {
		http.Error(wt, "Batch IDs list cannot be empty", http.StatusBadRequest)
		return
	}

	pdfData, err := s.rep.GenerateReport(req.BatchIDs)
	if err != nil {
		http.Error(wt, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("link_report_%s.pdf", time.Now().Format("2006-01-02_15-04-05"))

	wt.Header().Set("Content-Type", "application/pdf")
	wt.Header().Set("Content-Disposition", "attachment; filename="+filename)
	wt.Header().Set("Content-Length", strconv.Itoa(len(pdfData)))

	wt.Write(pdfData)
}

// Новый хендлер для проверки статуса пачки
func (s *Server) batchStatusHandler(wt http.ResponseWriter, r *http.Request) {
	batchIDStr := chi.URLParam(r, "batchID")
	batchID, err := strconv.Atoi(batchIDStr)
	if err != nil {
		http.Error(wt, "Invalid batch ID", http.StatusBadRequest)
		return
	}

	batch, err := s.rep.GetBanchByID(batchID)
	if err != nil {
		http.Error(wt, err.Error(), http.StatusNotFound)
		return
	}

	status := "completed"
	if !s.rep.IsBatchCompleted(batchID) {
		status = "processing"
	}

	wt.Header().Set("Content-Type", "application/json")
	json.NewEncoder(wt).Encode(CheckLinksResponse{
		BatchID: batchID,
		Status:  status,
		Links:   batch.Links,
	})
}
