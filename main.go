package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	var visits uint64
	mux := chi.NewMux()
	ch := make(chan *uint64, 1)
	defer close(ch)
	ch <- &visits
	var mu sync.Mutex

	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	mux.Get("/health-check", healthCheck)
	mux.Get("/visit-with-mutex", visitWithMutex(&visits, &mu))
	mux.Get("/visit-with-atomic", visitWithAtomic(&visits))
	mux.Get("/visit-with-channel", visitWithChannel(ch))

	s := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())	
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello, world!\n")
}

func visitWithMutex(visits *uint64, mu *sync.Mutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*visits++
		current := *visits
		mu.Unlock()

		message := fmt.Sprintf("Olá! Você teve %d visitas.", current)
		
		io.WriteString(w, message)
	}
}

func visitWithAtomic(visits *uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddUint64(visits, 1)
		message := fmt.Sprintf("Olá! Você teve %d visitas.", current)

		io.WriteString(w, message)
	}
}

func visitWithChannel(ch chan *uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		visits := <- ch
		*visits++
		message := fmt.Sprintf("Olá! Você teve %d visitas.", *visits)
		ch <- visits

		io.WriteString(w, message)
	}
}
