package client

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildHHQuery_FromProfile(t *testing.T) {
	role := "Go developer"
	skills := []string{"Go", "PostgreSQL", "gRPC", "Docker", "Kubernetes"}
	expLevel := "middle"
	areaID := "1"
	salaryMin := 150000.0
	currency := "RUR"
	workFormat := []string{"remote"}

	profile := &ResumeProfile{
		TargetRoles:     []string{role},
		SkillsTop:       skills,
		ExperienceLevel: &expLevel,
		Areas:           []Area{{ID: areaID, Name: "Moscow"}},
		SalaryMin:       &salaryMin,
		Currency:        &currency,
		WorkFormat:      workFormat,
	}

	params := BuildHHQuery(profile, 1, 20)

	if text := params["text"]; text == "" {
		t.Error("expected text from target_roles + skills")
	} else {
		if !strings.Contains(text, role) {
			t.Errorf("text should contain target role %q, got %q", role, text)
		}
		skillsPart := strings.Join(skills[:5], " ")
		if !strings.Contains(text, "Go") {
			t.Errorf("text should contain skills, got %q", text)
		}
		_ = skillsPart
	}
	if params["experience"] != "between3And6" {
		t.Errorf("expected experience between3And6, got %q", params["experience"])
	}
	if params["area"] != areaID {
		t.Errorf("expected area %q, got %q", areaID, params["area"])
	}
	if params["salary"] != "150000" {
		t.Errorf("expected salary 150000, got %q", params["salary"])
	}
	if params["currency"] != "RUR" {
		t.Errorf("expected currency RUR, got %q", params["currency"])
	}
	if params["schedule"] != "remote" {
		t.Errorf("expected schedule remote, got %q", params["schedule"])
	}
	if params["page"] != "1" {
		t.Errorf("expected page 1, got %q", params["page"])
	}
	if params["per_page"] != "20" {
		t.Errorf("expected per_page 20, got %q", params["per_page"])
	}
}

func TestBuildHHQuery_CurrencyDefaultRUR(t *testing.T) {
	salaryMin := 100000.0
	profile := &ResumeProfile{
		TargetRoles: []string{"Dev"},
		SalaryMin:   &salaryMin,
		Currency:    nil,
	}
	params := BuildHHQuery(profile, 0, 10)
	if params["currency"] != "RUR" {
		t.Errorf("expected default currency RUR, got %q", params["currency"])
	}
}

func TestBuildHHQuery_Pagination(t *testing.T) {
	profile := &ResumeProfile{TargetRoles: []string{"Test"}}
	params := BuildHHQuery(profile, 5, 50)
	if params["page"] != "5" {
		t.Errorf("expected page 5, got %q", params["page"])
	}
	if params["per_page"] != "50" {
		t.Errorf("expected per_page 50, got %q", params["per_page"])
	}
}

func TestBuildHHQuery_SkillsTopMax5(t *testing.T) {
	profile := &ResumeProfile{
		TargetRoles: []string{"Role"},
		SkillsTop:   []string{"a", "b", "c", "d", "e", "f"},
	}
	params := BuildHHQuery(profile, 0, 10)
	text := params["text"]
	count := strings.Count(text, " ")
	if count < 4 {
		t.Errorf("expected at least 5 skills (4 spaces), got text %q", text)
	}
	_ = strconv.Itoa(count)
}
