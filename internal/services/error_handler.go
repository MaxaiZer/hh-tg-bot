package services

import (
	"context"
	log "github.com/sirupsen/logrus"
)

type errorHandler struct {
	Done      chan struct{}
	vacancies vacancyRepository
}

func newErrorHandler(vacancies vacancyRepository) *errorHandler {
	return &errorHandler{make(chan struct{}), vacancies}
}

func (e *errorHandler) Run(errors <-chan analysisError) {
	total := 0
	for err := range errors {
		total++
		dbErr := e.vacancies.AddFailedToAnalyze(context.Background(), err.searchID, err.vacancyID, err.error.Error())
		if dbErr != nil {
			log.Errorf("couldn't add vacancy as failed to analyze: %v", dbErr)
		}
		log.Infof("vacancy saved as failed to analyze, searchID: %v vacancyID: %v, error: %v",
			err.searchID, err.vacancyID, err.error)
	}
	log.Infof("saved %v vacancies as failed to analyze", total)
	e.Done <- struct{}{}
}
