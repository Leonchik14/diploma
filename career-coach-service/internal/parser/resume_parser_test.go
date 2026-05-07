package parser

import (
	"testing"

	"career-coach-service/internal/model"
)

func ptr(s string) *string { return &s }

func TestNormalizeExperienceLevel(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want *string
	}{
		{name: "nil", in: nil, want: nil},
		{name: "canonical", in: ptr("between1And3"), want: ptr("between1And3")},
		{name: "russian phrase", in: ptr("менее года"), want: ptr("noExperience")},
		{name: "invalid value", in: ptr("senior"), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeExperienceLevel(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %q", *got)
				}
				return
			}
			if got == nil || *got != *tt.want {
				if got == nil {
					t.Fatalf("expected %q, got nil", *tt.want)
				}
				t.Fatalf("expected %q, got %q", *tt.want, *got)
			}
		})
	}
}

func TestNormalizeWorkFormatList(t *testing.T) {
	got := normalizeWorkFormatList([]string{"Удаленно", "remote", "Офис", "Гибрид", "unknown"})
	want := []string{"remote", "office", "hybrid"}
	if len(got) != len(want) {
		t.Fatalf("expected len=%d, got len=%d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestApplyAnswers_AreasAllRegions(t *testing.T) {
	p := &Parser{
		areasByID:    map[string]string{"1": "Москва"},
		areaIDByName: map[string]string{"москва": "1"},
	}
	draft := &model.ResumeProfileDraft{
		Confidence: map[string]float64{"areas": 0.1},
		Areas:      []model.Area{{ID: "1", Name: "Москва", Confidence: 1}},
	}
	questions := []model.Question{{ID: "areas", Type: "single_choice"}}
	answers := []model.QuestionAnswer{{QuestionID: "areas", Value: "Все регионы"}}

	out, next, status, err := p.ApplyAnswers(draft, questions, answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "completed" {
		t.Fatalf("expected completed, got %q", status)
	}
	if len(out.Areas) != 0 {
		t.Fatalf("expected empty areas for all regions, got %+v", out.Areas)
	}
	if out.Confidence["areas"] != 1.0 {
		t.Fatalf("expected areas confidence 1.0, got %v", out.Confidence["areas"])
	}
	if len(next) != 0 {
		t.Fatalf("expected no next questions, got %+v", next)
	}
}

func TestApplyAnswers_InvalidSalary(t *testing.T) {
	p := &Parser{}
	draft := &model.ResumeProfileDraft{Confidence: map[string]float64{"salary_min": 0.1}}
	questions := []model.Question{{ID: "salary_min", Type: "numeric_input"}}
	answers := []model.QuestionAnswer{{QuestionID: "salary_min", Value: "abc"}}

	_, _, _, err := p.ApplyAnswers(draft, questions, answers)
	if err == nil {
		t.Fatal("expected error for invalid salary answer")
	}
}

func TestBuildQuestionsForDraft_ContainsExpectedQuestions(t *testing.T) {
	p := &Parser{
		areasByID: map[string]string{
			"1": "Москва",
			"2": "Казань",
		},
		areaIDByName: map[string]string{
			"москва": "1",
			"казань": "2",
		},
	}
	draft := &model.ResumeProfileDraft{
		Confidence: map[string]float64{},
	}

	questions := p.BuildQuestionsForDraft(draft)
	if len(questions) != 4 {
		t.Fatalf("expected 4 questions, got %d", len(questions))
	}

	qByID := map[string]model.Question{}
	for _, q := range questions {
		qByID[q.ID] = q
	}
	if qByID["areas"].Type != "single_choice" {
		t.Fatalf("areas question type mismatch: %+v", qByID["areas"])
	}
	if len(qByID["areas"].Options) == 0 || qByID["areas"].Options[0] != "Все регионы" {
		t.Fatalf("expected areas options to start with 'Все регионы', got %v", qByID["areas"].Options)
	}
	if qByID["salary_min"].Type != "numeric_input" {
		t.Fatalf("salary_min question type mismatch: %+v", qByID["salary_min"])
	}
}
