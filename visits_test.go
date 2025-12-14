package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	w := httptest.NewRecorder()

	healthCheck(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("esperado status %d, obtido %d", http.StatusOK, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("erro ao ler resposta: %v", err)
	}

	expected := "Hello, world!\n"
	if string(body) != expected {
		t.Errorf("esperado %q, obtido %q", expected, string(body))
	}
}

func TestVisitWithMutex(t *testing.T) {
	var visits uint64
	var mu sync.Mutex
	handler := visitWithMutex(&visits, &mu)

	// Sequential test
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/visit-with-mutex", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("esperado status %d, obtido %d", http.StatusOK, res.StatusCode)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("erro ao ler resposta: %v", err)
		}

		expected := strings.Contains(string(body), "Olá! Você teve")
		if !expected {
			t.Errorf("resposta não contém mensagem esperada: %s", string(body))
		}

		if visits != uint64(i) {
			t.Errorf("esperado %d visitas, obtido %d", i, visits)
		}
	}

	// Concurrency test
	var concurrentVisits uint64
	var concurrentMu sync.Mutex
	concurrentHandler := visitWithMutex(&concurrentVisits, &concurrentMu)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-mutex", nil)
			w := httptest.NewRecorder()
			concurrentHandler(w, req)
		}()
	}

	wg.Wait()

	if concurrentVisits != uint64(numGoroutines) {
		t.Errorf("esperado %d visitas concorrentes, obtido %d", numGoroutines, concurrentVisits)
	}
}

func TestVisitWithAtomic(t *testing.T) {
	var visits uint64
	handler := visitWithAtomic(&visits)

	// Sequential test
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/visit-with-atomic", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("esperado status %d, obtido %d", http.StatusOK, res.StatusCode)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("erro ao ler resposta: %v", err)
		}

		expected := strings.Contains(string(body), "Olá! Você teve")
		if !expected {
			t.Errorf("resposta não contém mensagem esperada: %s", string(body))
		}

		if visits != uint64(i) {
			t.Errorf("esperado %d visitas, obtido %d", i, visits)
		}
	}

	// Concurrency test
	var concurrentVisits uint64
	concurrentHandler := visitWithAtomic(&concurrentVisits)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-atomic", nil)
			w := httptest.NewRecorder()
			concurrentHandler(w, req)
		}()
	}

	wg.Wait()

	if concurrentVisits != uint64(numGoroutines) {
		t.Errorf("esperado %d visitas concorrentes, obtido %d", numGoroutines, concurrentVisits)
	}
}

func TestVisitWithChannel(t *testing.T) {
	var visits uint64
	ch := make(chan *uint64, 1)
	ch <- &visits
	defer close(ch)

	handler := visitWithChannel(ch)

	// Sequential test
	for i := 1; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/visit-with-channel", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		res := w.Result()
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Errorf("esperado status %d, obtido %d", http.StatusOK, res.StatusCode)
		}

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("erro ao ler resposta: %v", err)
		}

		expected := strings.Contains(string(body), "Olá! Você teve")
		if !expected {
			t.Errorf("resposta não contém mensagem esperada: %s", string(body))
		}

		if visits != uint64(i) {
			t.Errorf("esperado %d visitas, obtido %d", i, visits)
		}
	}

	// Concurrency test
	var concurrentVisits uint64
	concurrentCh := make(chan *uint64, 1)
	concurrentCh <- &concurrentVisits
	defer close(concurrentCh)

	concurrentHandler := visitWithChannel(concurrentCh)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-channel", nil)
			w := httptest.NewRecorder()
			concurrentHandler(w, req)
		}()
	}

	wg.Wait()

	if concurrentVisits != uint64(numGoroutines) {
		t.Errorf("esperado %d visitas concorrentes, obtido %d", numGoroutines, concurrentVisits)
	}
}

func TestVisitWithMutexRaceCondition(t *testing.T) {
	var visits uint64
	var mu sync.Mutex
	handler := visitWithMutex(&visits, &mu)

	var wg sync.WaitGroup
	numGoroutines := 1000
	expectedVisits := uint64(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-mutex", nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}()
	}

	wg.Wait()

	if visits != expectedVisits {
		t.Errorf("race condition detectada: esperado %d visitas, obtido %d", expectedVisits, visits)
	}
}

func TestVisitWithAtomicRaceCondition(t *testing.T) {
	var visits uint64
	handler := visitWithAtomic(&visits)

	var wg sync.WaitGroup
	numGoroutines := 1000
	expectedVisits := uint64(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-atomic", nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}()
	}

	wg.Wait()

	if visits != expectedVisits {
		t.Errorf("race condition detectada: esperado %d visitas, obtido %d", expectedVisits, visits)
	}
}

func TestVisitWithChannelRaceCondition(t *testing.T) {
	var visits uint64
	ch := make(chan *uint64, 1)
	ch <- &visits
	defer close(ch)

	handler := visitWithChannel(ch)

	var wg sync.WaitGroup
	numGoroutines := 1000
	expectedVisits := uint64(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/visit-with-channel", nil)
			w := httptest.NewRecorder()
			handler(w, req)
		}()
	}

	wg.Wait()

	if visits != expectedVisits {
		t.Errorf("race condition detectada: esperado %d visitas, obtido %d", expectedVisits, visits)
	}
}

func BenchmarkVisitWithMutex(b *testing.B) {
	var visits uint64
	var mu sync.Mutex
	handler := visitWithMutex(&visits, &mu)

	req := httptest.NewRequest(http.MethodGet, "/visit-with-mutex", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkVisitWithAtomic(b *testing.B) {
	var visits uint64
	handler := visitWithAtomic(&visits)

	req := httptest.NewRequest(http.MethodGet, "/visit-with-atomic", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}

func BenchmarkVisitWithChannel(b *testing.B) {
	var visits uint64
	ch := make(chan *uint64, 1)
	ch <- &visits
	defer close(ch)

	handler := visitWithChannel(ch)

	req := httptest.NewRequest(http.MethodGet, "/visit-with-channel", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}
