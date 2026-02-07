package model

import (
	"time"
)

type EventType int32

const (
	EventTypeUnspecified EventType = 0
	EventTypeInterview   EventType = 1 // Собеседование
	EventTypeCall        EventType = 2 // Звонок
	EventTypeMeeting     EventType = 3 // Встреча
	EventTypeTestTask    EventType = 4 // Тестовое задание
	EventTypePrep        EventType = 5
	EventTypeDeadline    EventType = 6
	EventTypeOther       EventType = 7
)

type Event struct {
	ID                string
	UserID            uint
	Title             string
	Description       *string
	EventType         EventType
	StartTime         time.Time
	EndTime           time.Time
	Timezone          *string
	Location          *string
	RelatedVacancyID  *string
	ReminderMinutes   int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PageToken struct {
	StartTime time.Time
	ID        string
}
