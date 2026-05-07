package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"calendar-service/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateEvent_ValidationErrors(t *testing.T) {
	svc := NewCalendarService(nil)
	now := time.Now()

	tests := []struct {
		name string
		ev   *model.Event
		code codes.Code
	}{
		{
			name: "empty title",
			ev: &model.Event{
				Title:     "   ",
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "title too long",
			ev: &model.Event{
				Title:     strings.Repeat("а", maxTitleLen+1),
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "invalid timezone",
			ev: &model.Event{
				Title:     "Interview",
				Timezone:  strPtr("Mars/Phobos"),
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "start equals end",
			ev: &model.Event{
				Title:     "Interview",
				StartTime: now,
				EndTime:   now,
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateEvent(context.Background(), 1, tt.ev)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok || st.Code() != tt.code {
				t.Fatalf("expected gRPC code %v, got %v", tt.code, err)
			}
		})
	}
}

func TestListEvents_ValidationErrors(t *testing.T) {
	svc := NewCalendarService(nil)
	now := time.Now()

	t.Run("from after to", func(t *testing.T) {
		_, _, err := svc.ListEvents(context.Background(), 1, now.Add(time.Hour), now, 10, "", true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})

	t.Run("range exceeds one year", func(t *testing.T) {
		_, _, err := svc.ListEvents(context.Background(), 1, now, now.Add(366*24*time.Hour), 10, "", true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", err)
		}
	})
}

func strPtr(s string) *string { return &s }
