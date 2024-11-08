package hh

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"os"
	"testing"
)

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func getVacancyMock() (*http.Response, error) {
	file, err := os.ReadFile("testdata/get_vacancy.json")

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBuffer(file)),
	}, err
}

func getVacanciesMock() (*http.Response, error) {
	file, err := os.ReadFile("testdata/get_vacancies.json")

	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBuffer(file)),
	}, err
}

func Test_HHClient_GetVacancies_ShouldBeSuccessful(t *testing.T) {

	assert := assert.New(t)

	mockClient := &mockHTTPClient{}
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == "https://api.hh.ru/vacancies?experience=noExperience&page=1&"+
			"perPage=10&period=1&schedule=fullDay&text=golang"
	})).Return(getVacanciesMock())

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
	assert.NoError(err)

	assert.True(len(vacancies) == 2)
	assert.Equal(vacancies[0].ID, "107958774")
	assert.Equal(vacancies[0].Name, "Разработчик веб-приложений / фронтенд / верстальщик HTML (Junior)")
	assert.Equal(vacancies[1].ID, "108122273")
	assert.Equal(vacancies[1].Name, "Junior/Junior+ Golang developer")
}

func Test_HHClient_GetVacancy_ShouldBeSuccessful(t *testing.T) {

	assert := assert.New(t)
	vacancyID := "108444291"

	mockClient := &mockHTTPClient{}
	mockClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == "https://api.hh.ru/vacancies/"+vacancyID
	})).Return(getVacancyMock())

	client := NewClient()
	client.SetHTTPClient(mockClient)

	vacancy, err := client.GetVacancy(vacancyID)
	assert.NoError(err)
	assert.Equal(vacancy.ID, vacancyID)
	assert.Equal(vacancy.Name, "Младший Back-end разработчик")
}
