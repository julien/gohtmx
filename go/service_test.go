package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStartServer(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name    string
		addr    string
		success bool
	}{
		{
			addr:    ":3000",
			success: true,
		},
		{
			addr:    "123::3000",
			success: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup

			if tc.success {
				wg.Add(1)
				service := Service(tc.addr)
				srv := service.Start(&wg)

				ctx, cancel := context.WithTimeout(context.TODO(), 400*time.Millisecond)
				defer cancel()

				if err := srv.Shutdown(ctx); err != nil {
					t.Errorf("expected error to be nil, got %v", err)
				}
				wg.Wait()
			}
		})
	}
}

func TestServiceRead(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name       string
		addr       string
		statusCode int
	}{
		{
			addr:       ":3001",
			statusCode: http.StatusOK,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			service := Service(tc.addr)
			srv := service.Start(&wg)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
			defer cancel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			service.read(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("couldn't rea response body: %v", err)
				}
				if len(b) == 0 {
					t.Error("got an empty response")
				}
			}

			if res.StatusCode != tc.statusCode {
				t.Errorf("expected status code %d, got %d\n", tc.statusCode, res.StatusCode)
			}

			if err := srv.Shutdown(ctx); err != nil {
				t.Errorf("expected error to be nil, got %v", err)
			}

			wg.Wait()
		})
	}
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		name       string
		addr       string
		method     string
		headers    http.Header
		data       url.Values
		statusCode int
	}{
		{
			addr:       ":3002",
			data:       url.Values{"title": {"stuff"}},
			method:     http.MethodPost,
			statusCode: http.StatusCreated,
			headers: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
		{
			addr:       ":3003",
			data:       url.Values{"title": {"stuff"}},
			method:     http.MethodPut,
			statusCode: http.StatusMethodNotAllowed,
			headers: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
		{
			addr:       ":3004",
			data:       url.Values{"title": {"stuff"}},
			method:     http.MethodPost,
			statusCode: http.StatusUnsupportedMediaType,
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			addr:       ":3005",
			data:       url.Values{"title": {}},
			method:     http.MethodPost,
			statusCode: http.StatusBadRequest,
			headers: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			service := Service(tc.addr)
			srv := service.Start(&wg)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
			defer cancel()

			req := httptest.NewRequest(tc.method, "/create", strings.NewReader(tc.data.Encode()))
			w := httptest.NewRecorder()
			for k, v := range tc.headers {
				for _, vv := range v {
					req.Header.Set(k, vv)
				}
			}

			service.create(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode == http.StatusCreated {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("couldn't rea response body: %v", err)
				}
				if len(b) == 0 {
					t.Error("got an empty response")
				}
			}

			if res.StatusCode != tc.statusCode {
				t.Errorf("expected status code %d, got %d\n", tc.statusCode, res.StatusCode)
			}

			if err := srv.Shutdown(ctx); err != nil {
				t.Errorf("expected error to be nil, got %v", err)
			}

			wg.Wait()
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	t.Parallel()

	id := uuid.New()

	tcs := []struct {
		name       string
		addr       string
		method     string
		headers    http.Header
		data       url.Values
		statusCode int
	}{
		{
			addr:       ":3006",
			data:       url.Values{"title": {"more stuff"}, "id": {id.String()}, "done": {"on"}},
			method:     http.MethodPost,
			statusCode: http.StatusOK,
			headers: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
		{
			addr:       ":3007",
			method:     http.MethodPut,
			statusCode: http.StatusMethodNotAllowed,
		},
		{
			addr:       ":3008",
			method:     http.MethodPost,
			statusCode: http.StatusUnsupportedMediaType,
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			addr:       ":3009",
			data:       url.Values{"title": {"more stuff"}, "id": {id.String()}},
			method:     http.MethodPost,
			statusCode: http.StatusOK,
			headers: http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			service := Service(tc.addr)
			service.todos[id.String()] = todo{ID: id, Title: "stuff"}
			srv := service.Start(&wg)

			ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
			defer cancel()

			req := httptest.NewRequest(tc.method, "/update", strings.NewReader(tc.data.Encode()))
			w := httptest.NewRecorder()
			for k, v := range tc.headers {
				for _, vv := range v {
					req.Header.Set(k, vv)
				}
			}

			service.update(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("couldn't rea response body: %v", err)
				}
				if len(b) == 0 {
					t.Error("got an empty response")
				}
			}

			if res.StatusCode != tc.statusCode {
				t.Errorf("expected status code %d, got %d\n", tc.statusCode, res.StatusCode)
			}

			if err := srv.Shutdown(ctx); err != nil {
				t.Errorf("expected error to be nil, got %v", err)
			}

			wg.Wait()
		})
	}
}
