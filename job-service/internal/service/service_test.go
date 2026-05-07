package service

import (
	"context"
	"sort"
	"testing"
)

func TestNewService_ListAreasSorted(t *testing.T) {
	svc := NewService(nil, nil, nil)

	areas, err := svc.ListAreas(context.Background())
	if err != nil {
		t.Fatalf("ListAreas returned error: %v", err)
	}
	if len(areas) != len(top50AreaNames) {
		t.Fatalf("expected %d areas, got %d", len(top50AreaNames), len(areas))
	}

	if !sort.StringsAreSorted(areas) {
		t.Fatalf("expected sorted areas, got %v", areas)
	}
}

func TestListAreas_ReturnsCopy(t *testing.T) {
	svc := NewService(nil, nil, nil)

	first, err := svc.ListAreas(context.Background())
	if err != nil {
		t.Fatalf("first ListAreas error: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("expected non-empty areas list")
	}

	originalFirst := first[0]
	first[0] = "MUTATED"

	second, err := svc.ListAreas(context.Background())
	if err != nil {
		t.Fatalf("second ListAreas error: %v", err)
	}
	if second[0] != originalFirst {
		t.Fatalf("expected defensive copy, got mutated data: %v", second[0])
	}
}
