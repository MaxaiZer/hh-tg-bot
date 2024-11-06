package hh

import (
	"encoding/json"
	"fmt"
	"time"
)

type Vacancy struct {
	VacancyPreview
	Description string
	KeySkills   []KeySkill `json:"key_skills"`
}

type VacancyPreview struct {
	ID          string
	Name        string
	Url         string     `json:"alternate_url"`
	PublishedAt CustomTime `json:"published_at"`
}

type KeySkill struct {
	Name string
}

type CustomTime struct {
	time.Time
}

func (dt *CustomTime) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	t, err := time.Parse("2006-01-02T15:04:05-0700", str)
	if err != nil {
		return fmt.Errorf("parsing time %s: %v", str, err)
	}
	dt.Time = t
	return nil
}
