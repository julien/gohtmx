package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type page struct {
	Todos []todo
}

type todo struct {
	ID    uuid.UUID
	Title string
	Done  bool
}

type service struct {
	addr    string
	handler http.Handler
	tpl     *template.Template
	todos   []todo
}

func Service(addr string) *service {
	s := &service{
		addr: addr,
		tpl:  template.Must(template.ParseFiles("layout.html")),
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/create", s.create)
	handler.HandleFunc("/update", s.update)
	handler.HandleFunc("/", s.read)
	s.handler = handler

	s.todos = make([]todo, 0)

	return s
}

func (s *service) Start(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{
		Addr:         s.addr,
		Handler:      s.handler,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("listen error: %v\n", err)
			return
		}
		// clean up
	}()
	return srv
}

func (s *service) read(w http.ResponseWriter, r *http.Request) {
	if err := s.tpl.Execute(w, page{Todos: s.todos}); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
}

func (s *service) create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	if len(strings.TrimSpace(title)) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.todos = append(s.todos, todo{ID: uuid.New(), Title: title})

	w.WriteHeader(http.StatusCreated)
	s.tpl.ExecuteTemplate(w, "content", s.todos)
}

func (s *service) update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var (
		done = r.FormValue("done")
		id   = r.FormValue("id")
	)

	for i := 0; i < len(s.todos); i++ {
		if s.todos[i].ID.String() == id {
			if done == "on" {
				s.todos[i].Done = true
			} else {
				s.todos[i].Done = false
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	s.tpl.ExecuteTemplate(w, "content", s.todos)
}
