package services

import (
	"context"
	"fmt"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"strings"
)

type aiClient interface {
	GenerateResponse(ctx context.Context, request string) (string, error)
}

type AIService struct {
	aiClient aiClient
}

func NewAIService(aiClient aiClient) *AIService {
	return &AIService{aiClient: aiClient}
}

func (a *AIService) DoesVacancyMatchSearch(ctx context.Context, search entities.JobSearch, vacancy hh.Vacancy) (bool, error) {
	response, err := a.aiClient.GenerateResponse(ctx, a.vacancyMatchSearchRequest(search, vacancy))
	if err != nil {
		return false, err
	}

	log.Infof("got response \"%v\" for vacancy %v", response, vacancy.Url)
	response = strings.ReplaceAll(strings.ToLower(response), "*", "") //т.к. иногда может ответить **скорее нет**

	if hasPrefixes(response, []string{"скорее да", "да"}) {
		return true, nil
	} else if hasPrefixes(response, []string{"скорее нет", "нет"}) {
		return false, nil
	} else {
		return false, fmt.Errorf("unexpected response \"%v\" for vacancy %v", response, vacancy.Url)
	}
}

func (a *AIService) vacancyMatchSearchRequest(search entities.JobSearch, vacancy hh.Vacancy) (request string) {

	request = "Название вакансии: " + vacancy.Name
	request += " Описание: " + vacancy.Description

	if len(vacancy.KeySkills) != 0 {
		skillNames := lo.Map(vacancy.KeySkills, func(skill hh.KeySkill, _ int) string { return skill.Name })
		request += " Ключевые навыки: " + strings.Join(skillNames, ", ")
	}

	request += " Пожелание к вакансии: " + search.UserWish
	request += " Ты фильтруешь вакансии на основе пожелания пользователя. Соответствует ли вакансия его запросу? " +
		"Тщательно проанализируй. Можешь отвечать в качестве степени уверенности (по нарастающей) только \"нет\",\"скорее нет\",\"скорее да\", \"да\""
	return request
}

func hasPrefixes(str string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}