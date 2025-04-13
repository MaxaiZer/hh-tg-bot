package events

import (
	"github.com/maxaizer/hh-parser/internal/domain/models"
)

var VacancyFoundTopic = "VacancyFoundEvent"

type VacancyFound struct {
	Search models.JobSearch
	Name   string
	Url    string
}
