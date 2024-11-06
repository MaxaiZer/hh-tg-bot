package events

var SearchDeletedTopic = "SearchDeletedEvent"

type SearchDeleted struct {
	SearchID int
}
