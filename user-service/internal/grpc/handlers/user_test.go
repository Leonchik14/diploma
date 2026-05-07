package handlers

import (
	"testing"

	pbuser "proto/user"
	"user-service/internal/repo/postgres"
)

func strPtr(s string) *string { return &s }

func TestNormalizeExperienceLevelInput(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want *string
	}{
		{name: "nil stays nil", in: nil, want: nil},
		{name: "russian no experience", in: strPtr("Нет опыта"), want: strPtr("noExperience")},
		{name: "canonical with spaces", in: strPtr("  between3And6 "), want: strPtr("between3And6")},
		{name: "unknown non-empty passes through", in: strPtr("Senior"), want: strPtr("Senior")},
		{name: "empty to nil", in: strPtr("   "), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeExperienceLevelInput(tt.in)
			if tt.want == nil && got != nil {
				t.Fatalf("expected nil, got %q", *got)
			}
			if tt.want != nil && (got == nil || *got != *tt.want) {
				if got == nil {
					t.Fatalf("expected %q, got nil", *tt.want)
				}
				t.Fatalf("expected %q, got %q", *tt.want, *got)
			}
		})
	}
}

func TestNormalizeWorkFormatsInput(t *testing.T) {
	in := []string{"Удаленно", "remote", "Офис", "  ", "Гибрид", "hybrid"}
	got := normalizeWorkFormatsInput(in)

	want := []string{"remote", "office", "hybrid"}
	if len(got) != len(want) {
		t.Fatalf("expected len %d, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestNormalizeAreaIDsInput(t *testing.T) {
	tests := []struct {
		name string
		in   []struct{ id, name string }
		want []string
	}{
		{
			name: "prefer id, fallback to name and dedupe",
			in: []struct{ id, name string }{
				{id: "1", name: "Москва"},
				{id: "", name: "Санкт-Петербург"},
				{id: "1", name: "Москва"},
			},
			want: []string{"1", "Санкт-Петербург"},
		},
		{
			name: "all regions clears areas",
			in: []struct{ id, name string }{
				{id: "", name: "Все регионы"},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			areas := make([]*pbuser.Area, 0, len(tt.in))
			for _, a := range tt.in {
				areas = append(areas, &pbuser.Area{Id: a.id, Name: a.name})
			}

			got := normalizeAreaIDsInput(areas)
			if len(got) != len(tt.want) {
				t.Fatalf("expected len %d, got %d (%v)", len(tt.want), len(got), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("expected %v, got %v", tt.want, got)
				}
			}
		})
	}
}

func TestRowToProtoProfileHumanized(t *testing.T) {
	row := &postgres.ResumeProfileRow{
		TargetRoles:     []string{"Go Developer"},
		ExperienceLevel: strPtr("between1And3"),
		AreaIDs:         []string{},
		WorkFormat:      []string{"remote", "office"},
	}

	got := rowToProtoProfile(row, true)
	if got.ExperienceLevel == nil || *got.ExperienceLevel != "1-3 года" {
		t.Fatalf("expected humanized experience_level, got %+v", got.ExperienceLevel)
	}
	if len(got.WorkFormat) != 2 || got.WorkFormat[0] != "Удаленно" || got.WorkFormat[1] != "Офис" {
		t.Fatalf("expected humanized work_format, got %v", got.WorkFormat)
	}
	if len(got.Areas) != 1 || got.Areas[0].Name != "Все регионы" {
		t.Fatalf("expected all-regions placeholder, got %+v", got.Areas)
	}
}
