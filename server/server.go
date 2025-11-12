package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewServer(rep *Repository) *Server {
	return &Server{
		rep: rep,
	}
}

func (s *Server) Start(port string) error {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/api", func(r chi.Router) {
		r.Post("/check", s.checkLinksHandler)
		r.Post("/report", s.reportHandler)
		r.Get("/status/{batchID}", s.batchStatusHandler)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		fmt.Printf("Server listening on port %s...\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-sigChan:
		fmt.Println("\nReceived shutdown signal...")
	case err := <-serverErr:
		fmt.Printf("Server error: %v\n", err)
		return err
	}

	// Начинаем graceful shutdown
	fmt.Println("Starting graceful shutdown...")

	// Даем серверу 30 секунд на завершение текущих запросов
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем прием новых запросов
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("HTTP server shutdown error: %v\n", err)
	}

	// Ждем завершения всех воркеров
	s.rep.Shutdown()

	fmt.Println("Server shutdown completed")

	return nil
}
