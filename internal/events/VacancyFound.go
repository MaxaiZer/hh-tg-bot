package events

import "github.com/maxaizer/hh-parser/internal/entities"

var VacancyFoundTopic = "VacancyFoundEvent"

type VacancyFound struct {
	Search entities.JobSearch
	Name   string
	Url    string
}
