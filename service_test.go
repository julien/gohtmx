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
)

func TestStartServer(t *testing.T) {
	// repo, teardown := setup(t, "test.service.startserver.db")
	// defer teardown()

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

func TestServiceTodo(t *testing.T) {
	// repo, teardown := setup(t, "test.service.create.db")
	// defer teardown()

	tcs := []struct {
		name       string
		addr       string
		method     string
		headers    http.Header
		data       url.Values
		statusCode int
	}{
		{
			addr:       ":3000",
			data:       url.Values{"title": {"stuff"}},
			method: http.MethodPost,	
			statusCode: http.StatusCreated,
		},
		{
			addr:       ":3000",
			data:       url.Values{"title": {"stuff"}},
			method: http.MethodPut,	
			statusCode: http.StatusMethodNotAllowed,
		},
		{
			addr:       ":3000",
			data:       url.Values{"title": {"stuff"}},
			method: http.MethodPut,
			statusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			service := Service(tc.addr)
			srv := service.Start(&wg)

			ctx, cancel := context.WithTimeout(context.TODO(),time.Second)
			defer cancel()

			req := httptest.NewRequest(tc.method, "/todo", strings.NewReader(tc.data.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			service.todo(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode == http.StatusCreated {
				b, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("couldn't read response body: %v", err)
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
