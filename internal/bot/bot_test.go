package bot

import (
	"context"
	"errors"
	"github.com/asaskevich/EventBus"
	botApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/maxaizer/hh-parser/internal/clients/hh"
	"github.com/maxaizer/hh-parser/internal/entities"
	"github.com/maxaizer/hh-parser/internal/events"
	"strconv"
	"testing"
)

type mockSearchRepo struct {
	Searches []entities.JobSearch
}

func (m *mockSearchRepo) Update(ctx context.Context, search entities.JobSearch) error {
	for i := 0; i < len(m.Searches); i++ {
		if m.Searches[i].ID == search.ID {
			m.Searches[i] = search
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockSearchRepo) Get(ctx context.Context, pageSize int, pageNum int) ([]entities.JobSearch, error) {
	return m.Searches, nil
}

func (m *mockSearchRepo) GetByUser(ctx context.Context, userID int64) ([]entities.JobSearch, error) {
	result := make([]entities.JobSearch, 0)
	for i := 0; i < len(m.Searches); i++ {
		if m.Searches[i].UserID == userID {
			result = append(result, m.Searches[i])
		}
	}
	return result, nil
}

func (m *mockSearchRepo) Add(ctx context.Context, search entities.JobSearch) error {
	m.Searches = append(m.Searches, search)
	return nil
}

func (m *mockSearchRepo) UpdateLastCheckedVacancy(ctx context.Context, id int, vacancy hh.VacancyPreview) error {
	panic("implement me")
}

func (m *mockSearchRepo) Remove(ctx context.Context, ID int) error {
	for i, search := range m.Searches {
		if search.ID == ID {
			m.Searches = append(m.Searches[:i], m.Searches[i+1:]...)
		}
	}
	return nil
}

type mockApi struct {
	SendedMessages []botApi.Chattable
}

func (m mockApi) Send(chattable botApi.Chattable) (botApi.Message, error) {
	m.SendedMessages = append(m.SendedMessages, chattable)
	return botApi.Message{}, nil
}

type mockRegionRepo struct {
	Regions []entities.Region
}

func (m *mockRegionRepo) GetIdByName(ctx context.Context, name string) (string, error) {
	for _, region := range m.Regions {
		if region.NormalizedName == entities.NormalizeRegionName(name) {
			return region.ID, nil
		}
	}
	return "", errors.New("not found")
}

func Test_AddSearchCmd_WhenValidData_ShouldBeSuccessful(t *testing.T) {

	region := entities.NewRegion("0", "Москва")
	mockSearches := &mockSearchRepo{}
	mockRegions := &mockRegionRepo{Regions: []entities.Region{region}}
	mockApi := &mockApi{}
	finished := false

	keywords := "C#"
	experience := string(noExperience)
	schedule := "0"
	wish := "Хочу пельмени"
	initialSearchPeriod := 1

	cmd := newAddSearchCommand(mockApi, 0, mockSearches, mockRegions)
	cmd.WithFinishCallback(func() {
		finished = true
	})
	cmd.Run()

	cmd.OnUserInput(keywords)
	cmd.OnUserInput(experience)
	cmd.OnUserInput(region.Name)
	cmd.OnUserInput(schedule)
	cmd.OnUserInput(wish)
	cmd.OnUserInput(strconv.Itoa(initialSearchPeriod))

	if !finished {
		t.Errorf("AddSearchCmd should have finished")
	}
	if len(mockSearches.Searches) == 0 {
		t.Errorf("AddSearchCmd should add search")
	}

	if mockSearches.Searches[0].SearchText != keywords {
		t.Errorf("wrong search text")
	}

	if mockSearches.Searches[0].RegionID != region.ID {
		t.Errorf("wrong region id")
	}

	if mockSearches.Searches[0].Experience != entities.NoExperience {
		t.Errorf("wrong experience")
	}

	if mockSearches.Searches[0].UserWish != wish {
		t.Errorf("wrong user wish")
	}

	if mockSearches.Searches[0].InitialSearchPeriod != initialSearchPeriod {
		t.Errorf("wrong InitialSearchPeriod")
	}
}

func Test_RemoveSearchCmd_WhenValidData_ShouldBeSuccessful(t *testing.T) {

	search := entities.JobSearch{
		ID:     0,
		UserID: 0,
	}
	mockSearches := &mockSearchRepo{Searches: []entities.JobSearch{search}}
	published := false
	mockBus := EventBus.New()
	_ = mockBus.Subscribe(events.SearchDeletedTopic, func(event events.SearchDeleted) { published = true })
	mockApi := &mockApi{}
	finished := false

	cmd := newRemoveSearchCommand(mockApi, search.UserID, mockBus, mockSearches)
	cmd.WithFinishCallback(func() {
		finished = true
	})
	cmd.Run()
	cmd.OnUserInput("1")

	if !finished {
		t.Errorf("RemoveSearchCmd should have finished")
	}
	if len(mockSearches.Searches) == 1 {
		t.Errorf("RemoveSearchCmd should remove search")
	}
	if !published {
		t.Errorf("RemoveSearchCmd should publish event")
	}
}
