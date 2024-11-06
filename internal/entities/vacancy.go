package entities

import "time"

type NotifiedVacancy struct {
	ID            int
	UserID        int64
	VacancyID     string
	LastCheckedAt time.Time
	CreatedAt     time.Time
}
