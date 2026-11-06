// Package job содержит логику трекера откликов на вакансии.
package job

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("application not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Status — стадия отклика.
type Status string

const (
	StatusWishlist  Status = "wishlist"  // хочу откликнуться
	StatusApplied   Status = "applied"   // отклик отправлен
	StatusInterview Status = "interview" // собеседование / тестовое
	StatusOffer     Status = "offer"     // оффер
	StatusRejected  Status = "rejected"  // отказ
)

// AllStatuses — порядок важен: так статусы выводятся в статистике.
var AllStatuses = []Status{
	StatusWishlist, StatusApplied, StatusInterview, StatusOffer, StatusRejected,
}

func (s Status) Valid() bool {
	for _, v := range AllStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// Application — один отклик на вакансию.
type Application struct {
	ID        int        `json:"id"`
	Company   string     `json:"company"`
	Position  string     `json:"position"`
	City      string     `json:"city,omitempty"`
	Link      string     `json:"link,omitempty"`    // ссылка на вакансию
	Contact   string     `json:"contact,omitempty"` // например, telegram рекрутера
	Notes     string     `json:"notes,omitempty"`
	Status    Status     `json:"status"`
	AppliedAt *time.Time `json:"applied_at,omitempty"` // когда отправил отклик
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Input — данные, которые приходят от клиента при создании/обновлении.
type Input struct {
	Company  string `json:"company"`
	Position string `json:"position"`
	City     string `json:"city"`
	Link     string `json:"link"`
	Contact  string `json:"contact"`
	Notes    string `json:"notes"`
	Status   Status `json:"status"`
}

// Validate очищает пробелы, ставит статус по умолчанию и проверяет обязательные поля.
func (in *Input) Validate() error {
	in.Company = strings.TrimSpace(in.Company)
	in.Position = strings.TrimSpace(in.Position)
	in.City = strings.TrimSpace(in.City)
	in.Link = strings.TrimSpace(in.Link)
	in.Contact = strings.TrimSpace(in.Contact)
	in.Notes = strings.TrimSpace(in.Notes)

	if in.Company == "" {
		return fmt.Errorf("%w: company is required", ErrInvalidInput)
	}
	if in.Position == "" {
		return fmt.Errorf("%w: position is required", ErrInvalidInput)
	}
	if in.Status == "" {
		in.Status = StatusWishlist
	}
	if !in.Status.Valid() {
		return fmt.Errorf("%w: unknown status %q", ErrInvalidInput, in.Status)
	}
	return nil
}
