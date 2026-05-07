package handlers

import (
	"testing"
	"time"

	"calendar-service/internal/model"
	pbcalendar "proto/calendar"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestPbToModel_ValidationAndMapping(t *testing.T) {
	h := &CalendarHandler{}
	now := time.Now()

	t.Run("missing start time", func(t *testing.T) {
		_, err := h.pbToModel(&pbcalendar.Event{
			Title:   "Interview",
			EndTime: timestamppb.New(now.Add(time.Hour)),
		})
		if err == nil {
			t.Fatal("expected error for missing start_time")
		}
	})

	t.Run("maps reminder correctly", func(t *testing.T) {
		ev, err := h.pbToModel(&pbcalendar.Event{
			Title:           "Interview",
			StartTime:       timestamppb.New(now),
			EndTime:         timestamppb.New(now.Add(time.Hour)),
			ReminderEnabled: true,
			ReminderMinutes: 30,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.ReminderMinutes != 30 {
			t.Fatalf("expected reminder 30, got %d", ev.ReminderMinutes)
		}
	})
}

func TestPbPatchToModel_MapsOptionalFields(t *testing.T) {
	h := &CalendarHandler{}
	now := time.Now()
	title := "Updated title"
	enabled := true
	minutes := int32(15)

	patch, err := h.pbPatchToModel(&pbcalendar.EventPatch{
		Title:           &title,
		StartTime:       timestamppb.New(now),
		ReminderEnabled: &enabled,
		ReminderMinutes: &minutes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patch.Title == nil || *patch.Title != title {
		t.Fatalf("expected title %q, got %+v", title, patch.Title)
	}
	if patch.StartTime == nil || !patch.StartTime.Equal(now) {
		t.Fatalf("expected start_time %v, got %+v", now, patch.StartTime)
	}
	if patch.ReminderEnabled == nil || *patch.ReminderEnabled != enabled {
		t.Fatalf("expected reminder_enabled=true, got %+v", patch.ReminderEnabled)
	}
	if patch.ReminderMinutes == nil || *patch.ReminderMinutes != minutes {
		t.Fatalf("expected reminder_minutes=%d, got %+v", minutes, patch.ReminderMinutes)
	}
}

func TestModelToPB_CompletedFlag(t *testing.T) {
	h := &CalendarHandler{}
	desc := "desc"

	past := &model.Event{
		ID:          "1",
		Title:       "Past",
		EventType:   model.EventTypeInterview,
		StartTime:   time.Now().Add(-2 * time.Hour),
		EndTime:     time.Now().Add(-time.Hour),
		Description: &desc,
		CreatedAt:   time.Now().Add(-3 * time.Hour),
		UpdatedAt:   time.Now().Add(-2 * time.Hour),
	}
	future := &model.Event{
		ID:        "2",
		Title:     "Future",
		EventType: model.EventTypeInterview,
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if !h.modelToPB(past).Completed {
		t.Fatal("expected past event to be completed")
	}
	if h.modelToPB(future).Completed {
		t.Fatal("expected future event to be not completed")
	}
}
