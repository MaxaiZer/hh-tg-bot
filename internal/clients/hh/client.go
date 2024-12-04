package hh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"net/http"
)

type getVacanciesResponse struct {
	Vacancies []VacancyPreview `json:"items"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient  HTTPClient
	rateLimiter *rate.Limiter
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{}}
}

func (c *Client) SetHTTPClient(client HTTPClient) {
	c.httpClient = client
}

func (c *Client) SetRateLimit(maxRequestsPerSecond float32) {
	c.rateLimiter = rate.NewLimiter(rate.Limit(maxRequestsPerSecond), 1)
}

func (c *Client) GetVacancies(parameters SearchParameters) ([]VacancyPreview, error) {

	if err := parameters.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	apiURL := "https://api.hh.ru/vacancies"
	params := parameters.ToUrlParams()

	body, err := c.sendRequest("GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var vacanciesResponse getVacanciesResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&vacanciesResponse); err != nil {
		return nil, fmt.Errorf("error decoding JSON response:: %v", err)
	}

	return vacanciesResponse.Vacancies, nil
}

func (c *Client) GetVacancy(id string) (Vacancy, error) {

	apiURL := "https://api.hh.ru/vacancies/" + id

	body, err := c.sendRequest("GET", apiURL, nil)
	if err != nil {
		return Vacancy{}, err
	}

	var vacancyResponse Vacancy
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&vacancyResponse); err != nil {
		return Vacancy{}, fmt.Errorf("error decoding JSON response:: %v", err)
	}

	return vacancyResponse, nil
}

func (c *Client) GetAreas() ([]Area, error) {

	apiUrl := "https://api.hh.ru/areas"

	body, err := c.sendRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	var areas []area
	if err = json.NewDecoder(bytes.NewReader(body)).Decode(&areas); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	var allAreas []Area

	var collectAreas func(areas []area)
	collectAreas = func(areas []area) {
		for _, area := range areas {
			allAreas = append(allAreas, Area{ID: area.ID, Name: area.Name})
			collectAreas(area.Areas)
		}
	}
	collectAreas(areas)
	return allAreas, nil
}

func (c *Client) sendRequest(method string, url string, body io.Reader) ([]byte, error) {

	if c.rateLimiter != nil {
		err := c.rateLimiter.Wait(context.Background())
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp)
}

func (c *Client) handleResponse(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %v, body: %v", resp.StatusCode, string(body))
	}

	return body, nil
}
