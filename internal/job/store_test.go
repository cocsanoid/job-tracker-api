package job

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func newMemStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreate_DefaultsToWishlist(t *testing.T) {
	s := newMemStore(t)
	a, err := s.Create(Input{Company: " AIparking ", Position: "Golang Intern"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != StatusWishlist || a.Company != "AIparking" || a.AppliedAt != nil {
		t.Errorf("unexpected: %+v", a)
	}
}

func TestCreate_Validation(t *testing.T) {
	s := newMemStore(t)
	tests := []struct {
		name string
		in   Input
	}{
		{"no company", Input{Position: "Dev"}},
		{"no position", Input{Company: "X"}},
		{"blank company", Input{Company: "   ", Position: "Dev"}},
		{"bad status", Input{Company: "X", Position: "Dev", Status: "maybe"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := s.Create(tt.in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("got %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestSetStatus_SetsAppliedAtOnce(t *testing.T) {
	s := newMemStore(t)
	a, _ := s.Create(Input{Company: "X", Position: "Dev"})

	a, err := s.SetStatus(a.ID, StatusApplied)
	if err != nil || a.AppliedAt == nil {
		t.Fatalf("applied_at must be set: %+v, %v", a, err)
	}
	first := *a.AppliedAt

	// при дальнейших сменах статуса дата отклика не должна меняться
	a, _ = s.SetStatus(a.ID, StatusInterview)
	if !a.AppliedAt.Equal(first) {
		t.Error("applied_at changed after second status change")
	}
}

func TestSetStatus_Errors(t *testing.T) {
	s := newMemStore(t)
	if _, err := s.SetStatus(1, StatusApplied); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
	a, _ := s.Create(Input{Company: "X", Position: "Dev"})
	if _, err := s.SetStatus(a.ID, "bogus"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("got %v, want ErrInvalidInput", err)
	}
}

func TestListFilterAndStats(t *testing.T) {
	s := newMemStore(t)
	s.Create(Input{Company: "A", Position: "Dev", Status: StatusApplied})
	s.Create(Input{Company: "B", Position: "Dev", Status: StatusApplied})
	s.Create(Input{Company: "C", Position: "Dev", Status: StatusRejected})

	applied := StatusApplied
	if n := len(s.List(&applied)); n != 2 {
		t.Errorf("filter: got %d, want 2", n)
	}

	st := s.Stats()
	if st.Total != 3 || st.ByStatus[StatusApplied] != 2 ||
		st.ByStatus[StatusRejected] != 1 || st.ByStatus[StatusOffer] != 0 {
		t.Errorf("unexpected stats: %+v", st)
	}
}

func TestDelete(t *testing.T) {
	s := newMemStore(t)
	a, _ := s.Create(Input{Company: "X", Position: "Dev"})
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")

	s1, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Create(Input{Company: "AIparking", Position: "Golang Intern", Status: StatusApplied})
	s1.Create(Input{Company: "Kaspi", Position: "Junior Go"})

	// "перезапуск": открываем тот же файл новым хранилищем
	s2, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	list := s2.List(nil)
	if len(list) != 2 || list[0].Company != "AIparking" {
		t.Fatalf("data not restored: %+v", list)
	}

	// id продолжаются с правильного места
	a, _ := s2.Create(Input{Company: "Third", Position: "Dev"})
	if a.ID != 3 {
		t.Errorf("next id = %d, want 3", a.ID)
	}
}

// Запускать с: go test -race ./...
func TestConcurrentCreate(t *testing.T) {
	s := newMemStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Create(Input{Company: "X", Position: "Dev"})
		}()
	}
	wg.Wait()
	if n := len(s.List(nil)); n != 50 {
		t.Errorf("got %d, want 50", n)
	}
}
