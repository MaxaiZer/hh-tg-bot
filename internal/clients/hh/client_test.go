package hh

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, fmt.Errorf("DoFunc not implemented")
}

func Test_HHClient_GetVacancies_ShouldBeSuccessful(t *testing.T) {

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {

			if req.URL.String() != "https://api.hh.ru/vacancies?experience=noExperience&page=1&perPage=10&period=1&schedule=fullDay&text=golang" {
				t.Errorf("Unexpected request URL: %s", req.URL.String())
			}

			file, err := os.ReadFile("testdata/get_vacancies.json")
			if err != nil {
				t.Errorf("Error reading file: %s", err)
			}

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBuffer(file)),
			}, nil
		},
	}

	client := NewClient()
	client.SetHTTPClient(mockClient)

	params := SearchParameters{
		Text:       "golang",
		Experience: NoExperience,
		Schedules:  []Schedule{FullDay},
		Page:       1,
		PerPage:    10,
		Period:     1,
	}
	vacancies, err := client.GetVacancies(params)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(vacancies) != 2 {
		t.Errorf("Expected 2 vacancies, got %d", len(vacancies))
	}

	if vacancies[0].ID != "107958774" || vacancies[0].Name != "Разработчик веб-приложений / фронтенд / верстальщик HTML (Junior)" {
		t.Errorf("Unexpected first vacancy: %+v", vacancies[0])
	}

	if vacancies[1].ID != "108122273" || vacancies[1].Name != "Junior/Junior+ Golang developer" {
		t.Errorf("Unexpected second vacancy: %+v", vacancies[1])
	}
}

func Test_HHClient_GetVacancy_ShouldBeSuccessful(t *testing.T) {

	vacancyID := "108444291"

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {

			if req.URL.String() != "https://api.hh.ru/vacancies/"+vacancyID {
				t.Errorf("Unexpected request URL: %s", req.URL.String())
			}

			file, err := os.ReadFile("testdata/get_vacancy.json")
			if err != nil {
				t.Errorf("Error reading file: %s", err)
			}

			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBuffer(file)),
			}, nil
		},
	}

	client := NewClient()
	client.SetHTTPClient(mockClient)

	vacancy, err := client.GetVacancy(vacancyID)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if vacancy.ID != vacancyID || vacancy.Name != "Младший Back-end разработчик" {
		t.Errorf("Unexpected vacancy: %+v", vacancy)
	}
}
