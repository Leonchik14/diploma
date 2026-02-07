package model

import "time"

type EventPatch struct {
	Title             *string
	Description       *string
	EventType         *EventType
	StartTime         *time.Time
	EndTime           *time.Time
	Timezone          *string
	Location          *string
	RelatedVacancyID  *string
	ReminderEnabled   *bool
	ReminderMinutes   *int32
}
