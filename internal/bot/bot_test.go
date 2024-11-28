package bot

import (
	"context"
	"errors"
	"fmt"
	"github.com/asaskevich/EventBus"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

type mockSearchRepo struct {
	Searches []entities.JobSearch
}

func (m *mockSearchRepo) Update(_ context.Context, search entities.JobSearch) error {
	for i := 0; i < len(m.Searches); i++ {
		if m.Searches[i].ID == search.ID {
			m.Searches[i] = search
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockSearchRepo) Get(_ context.Context, _ int, _ int) ([]entities.JobSearch, error) {
	return m.Searches, nil
}

func (m *mockSearchRepo) GetByUser(_ context.Context, userID int64) ([]entities.JobSearch, error) {
	result := make([]entities.JobSearch, 0)
	for i := 0; i < len(m.Searches); i++ {
		if m.Searches[i].UserID == userID {
			result = append(result, m.Searches[i])
		}
	}
	return result, nil
}

func (m *mockSearchRepo) GetByID(_ context.Context, ID int64) (*entities.JobSearch, error) {
	for i := 0; i < len(m.Searches); i++ {
		if m.Searches[i].ID == int(ID) {
			return &m.Searches[i], nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockSearchRepo) Add(_ context.Context, search entities.JobSearch) error {
	m.Searches = append(m.Searches, search)
	return nil
}

func (m *mockSearchRepo) UpdateLastCheckedVacancy(_ context.Context, _ int, _ hh.VacancyPreview) error {
	panic("implement me")
}

func (m *mockSearchRepo) Remove(_ context.Context, ID int) error {
	for i, search := range m.Searches {
		if search.ID == ID {
			m.Searches = append(m.Searches[:i], m.Searches[i+1:]...)
		}
	}
	return nil
}

type mockApi struct {
	SentMessages []botApi.Chattable
}

func (m mockApi) Send(chattable botApi.Chattable) (botApi.Message, error) {
	m.SentMessages = append(m.SentMessages, chattable)
	return botApi.Message{}, nil
}

type mockRegionRepo struct {
	Regions []entities.Region
}

func (m *mockRegionRepo) GetIdByName(_ context.Context, name string) (string, error) {
	for _, region := range m.Regions {
		if region.NormalizedName == entities.NormalizeRegionName(name) {
			return region.ID, nil
		}
	}
	return "", errors.New("not found")
}

func simulateUserInput(cmd command, inputs []string) {
	for _, input := range inputs {
		cmd.OnUserInput(input)
	}
}

func Test_AddSearchCmd_WhenValidData_ShouldBeSuccessful(t *testing.T) {

	assert := assert.New(t)

	region := entities.NewRegion("0", "Москва")
	mockSearches := &mockSearchRepo{}
	mockRegions := &mockRegionRepo{Regions: []entities.Region{region}}
	finished := false

	keywords := "C#"
	experience := string(noExperience)
	schedule := "0"
	wish := "Хочу пельмени"
	initialSearchPeriod := 1

	cmd := newAddSearchCommand(&mockApi{}, 0, mockSearches, mockRegions)
	cmd.WithFinishCallback(func() { finished = true })

	cmd.Run()
	simulateUserInput(cmd, []string{keywords, experience, region.Name, schedule, wish, strconv.Itoa(initialSearchPeriod)})

	assert.True(finished)
	assert.True(len(mockSearches.Searches) == 1)
	assert.Equal(keywords, mockSearches.Searches[0].SearchText)
	assert.Equal(region.ID, mockSearches.Searches[0].RegionID)
	assert.Equal(entities.NoExperience, mockSearches.Searches[0].Experience)
	assert.Equal(wish, mockSearches.Searches[0].UserWish)
	assert.Equal(initialSearchPeriod, mockSearches.Searches[0].InitialSearchPeriod)
}

func Test_AddSearchCmd_WhenInvalidInput_ShouldWaitForValid(t *testing.T) {

	assert := assert.New(t)

	region := entities.NewRegion("0", "Москва")
	mockSearches := &mockSearchRepo{}
	mockRegions := &mockRegionRepo{Regions: []entities.Region{region}}
	finished := false

	keywords := "C#"
	experience := string(noExperience)
	schedule := "0"
	wish := "Хочу пельмени"
	initialSearchPeriod := 1

	cmd := newAddSearchCommand(&mockApi{}, 0, mockSearches, mockRegions)
	cmd.WithFinishCallback(func() { finished = true })

	cmd.Run()
	cmd.OnUserInput(keywords)
	simulateUserInput(cmd, []string{"justRandomExperience", experience})
	simulateUserInput(cmd, []string{"justRandomRegion", region.Name})
	simulateUserInput(cmd, []string{"-1", schedule})
	cmd.OnUserInput(wish)
	simulateUserInput(cmd, []string{strconv.Itoa(-1), strconv.Itoa(6), strconv.Itoa(initialSearchPeriod)})

	assert.True(finished)
	assert.True(len(mockSearches.Searches) == 1)
	assert.Equal(keywords, mockSearches.Searches[0].SearchText)
	assert.Equal(region.ID, mockSearches.Searches[0].RegionID)
	assert.Equal(entities.NoExperience, mockSearches.Searches[0].Experience)
	assert.Equal(wish, mockSearches.Searches[0].UserWish)
	assert.Equal(initialSearchPeriod, mockSearches.Searches[0].InitialSearchPeriod)
}

func Test_RemoveSearchCmd_WhenValidData_ShouldBeSuccessful(t *testing.T) {

	assert := assert.New(t)

	search := entities.JobSearch{ID: 0, UserID: 0}
	mockSearches := &mockSearchRepo{Searches: []entities.JobSearch{search}}
	eventPublished := false
	mockBus := EventBus.New()
	_ = mockBus.Subscribe(events.SearchDeletedTopic, func(event events.SearchDeleted) { eventPublished = true })
	finished := false

	cmd, err := newRemoveSearchCommand(&mockApi{}, search.UserID, mockBus, mockSearches)
	assert.NoError(err)
	cmd.WithFinishCallback(func() { finished = true })

	cmd.Run()
	cmd.OnUserInput("1") //search num

	assert.True(finished)
	assert.Empty(mockSearches.Searches)
	assert.True(eventPublished)
}

func Test_RemoveSearchCmd_WhenInvalidInput_ShouldWaitForValid(t *testing.T) {

	assert := assert.New(t)

	search := entities.JobSearch{ID: 0, UserID: 0}
	mockSearches := &mockSearchRepo{Searches: []entities.JobSearch{search}}
	finished := false

	cmd, err := newRemoveSearchCommand(&mockApi{}, search.UserID, EventBus.New(), mockSearches)
	assert.NoError(err)
	cmd.WithFinishCallback(func() { finished = true })

	cmd.Run()
	simulateUserInput(cmd, []string{"-1", "2", "1"}) //search num
	cmd.OnUserInput("1")

	assert.True(finished)
	assert.Empty(mockSearches.Searches)
}

func Test_EditSearchCmd_WhenValidData_ShouldBeSuccessful(t *testing.T) {

	assert := assert.New(t)

	search := entities.JobSearch{ID: 0, UserID: 0}
	mockSearches := &mockSearchRepo{Searches: []entities.JobSearch{search}}
	eventPublished := false
	mockBus := EventBus.New()
	_ = mockBus.Subscribe(events.SearchEditedTopic, func(event events.SearchEdited) { eventPublished = true })
	finished := false

	cmd, err := newEditSearchCommand(&mockApi{}, search.UserID, mockBus, mockSearches)
	assert.NoError(err)
	cmd.WithFinishCallback(func() { finished = true })

	newKeywords := "1C"
	newWish := "вкалывать за копейки"

	cmd.Run()
	cmd.OnUserInput("1") //select search num
	cmd.OnUserInput("0") //select changing of keywords
	cmd.OnUserInput(newKeywords)

	assert.True(eventPublished)
	assert.False(finished)
	assert.Equal(newKeywords, mockSearches.Searches[0].SearchText)

	cmd.OnUserInput("1") //select changing of user wish
	cmd.OnUserInput(newWish)

	assert.False(finished)
	assert.Equal(newWish, mockSearches.Searches[0].UserWish)
}

func Test_EditSearchCmd_WhenInvalidInput_ShouldWaitForValid(t *testing.T) {

	assert := assert.New(t)

	search := entities.JobSearch{ID: 0, UserID: 0}
	mockSearches := &mockSearchRepo{Searches: []entities.JobSearch{search}}
	finished := false

	cmd, err := newEditSearchCommand(&mockApi{}, search.UserID, EventBus.New(), mockSearches)
	assert.NoError(err)
	cmd.WithFinishCallback(func() { finished = true })

	newKeywords := "1C"

	cmd.Run()
	simulateUserInput(cmd, []string{"-1", "2", "1"}) //select search num
	simulateUserInput(cmd, []string{"-1", "3", "0"}) //select changing of keywords
	cmd.OnUserInput(newKeywords)

	assert.False(finished)
	assert.Equal(newKeywords, mockSearches.Searches[0].SearchText)
}
