package entities

import "time"

type Vacancy struct {
	ID          string
	Url         string
	Name        string
	Description string
	KeySkills   []string
	PublishedAt time.Time
}

type NotifiedVacancy struct {
	ID              int
	UserID          int64
	VacancyID       string
	DescriptionHash []byte
	LastCheckedAt   time.Time
	CreatedAt       time.Time
}

type NotifiedVacancyID struct {
	UserID          int64
	VacancyID       string
	DescriptionHash []byte
}

type FailedVacancy struct {
	SearchID  int    `gorm:"primaryKey"`
	VacancyID string `gorm:"primaryKey"`
	Error     string
	Attempts  int `gorm:"default:1"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
