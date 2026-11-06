package job

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"
)

// Store хранит отклики в памяти и сохраняет их в JSON-файл.
// Если path пустой — работает только в памяти (удобно для тестов).
type Store struct {
	mu     sync.RWMutex
	path   string
	items  map[int]*Application
	nextID int
}

// NewStore создаёт хранилище и загружает данные из файла, если он уже есть.
func NewStore(path string) (*Store, error) {
	s := &Store{path: path, items: make(map[int]*Application), nextID: 1}
	if path == "" {
		return s, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil // файла ещё нет — начинаем с пустого списка
	}
	if err != nil {
		return nil, err
	}

	var list []*Application
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for _, a := range list {
		s.items[a.ID] = a
		if a.ID >= s.nextID {
			s.nextID = a.ID + 1
		}
	}
	return s, nil
}

// save записывает все данные в файл. Вызывать только под s.mu.Lock().
// Пишем во временный файл и переименовываем — так файл не испортится при сбое.
func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	list := make([]*Application, 0, len(s.items))
	for _, a := range s.items {
		list = append(list, a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Create(in Input) (Application, error) {
	if err := in.Validate(); err != nil {
		return Application{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	a := &Application{
		ID: s.nextID, Company: in.Company, Position: in.Position, City: in.City,
		Link: in.Link, Contact: in.Contact, Notes: in.Notes, Status: in.Status,
		CreatedAt: now, UpdatedAt: now,
	}
	markApplied(a, now)

	s.items[a.ID] = a
	s.nextID++
	return *a, s.save()
}

// List возвращает отклики по возрастанию id. status != nil — фильтр по статусу.
func (s *Store) List(status *Status) []Application {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Application, 0, len(s.items))
	for _, a := range s.items {
		if status == nil || a.Status == *status {
			result = append(result, *a)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *Store) Get(id int) (Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.items[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	return *a, nil
}

// Update полностью заменяет редактируемые поля отклика.
func (s *Store) Update(id int, in Input) (Application, error) {
	if err := in.Validate(); err != nil {
		return Application{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.items[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	now := time.Now()
	a.Company, a.Position, a.City = in.Company, in.Position, in.City
	a.Link, a.Contact, a.Notes, a.Status = in.Link, in.Contact, in.Notes, in.Status
	a.UpdatedAt = now
	markApplied(a, now)
	return *a, s.save()
}

// SetStatus меняет только статус (например, "applied" -> "interview").
func (s *Store) SetStatus(id int, status Status) (Application, error) {
	if !status.Valid() {
		return Application{}, ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.items[id]
	if !ok {
		return Application{}, ErrNotFound
	}
	now := time.Now()
	a.Status = status
	a.UpdatedAt = now
	markApplied(a, now)
	return *a, s.save()
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return s.save()
}

// Stats — сколько откликов в каждом статусе.
type Stats struct {
	Total    int            `json:"total"`
	ByStatus map[Status]int `json:"by_status"`
}

func (s *Store) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := Stats{Total: len(s.items), ByStatus: make(map[Status]int)}
	for _, status := range AllStatuses {
		st.ByStatus[status] = 0 // чтобы нулевые статусы тоже были в ответе
	}
	for _, a := range s.items {
		st.ByStatus[a.Status]++
	}
	return st
}

// markApplied запоминает дату первого отклика: как только статус
// становится не "wishlist", а отклик реально ушёл.
func markApplied(a *Application, now time.Time) {
	if a.Status != StatusWishlist && a.AppliedAt == nil {
		a.AppliedAt = &now
	}
}
