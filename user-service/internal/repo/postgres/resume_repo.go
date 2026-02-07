package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"user-service/internal/database"

	"github.com/jackc/pgx/v5"
)

type ResumeProfileRow struct {
	UserID           uint
	Status           string
	SourceMaterialID *string
	TargetRoles      []string
	ExperienceLevel  *string
	AreaIDs          []string
	SalaryMin        *int
	Currency         *string
	WorkFormat       []string
	SkillsTop        []string
	EducationLevel   *string
	Notes            *string
	Confidence       map[string]float64
	ConfirmedFields  map[string]bool
	Version          int64
}

func (r *ResumeProfileRow) AreaIDsToAreas() []AreaPair {
	// area_ids храним как ["id1","id2"], имена подставляются из справочника или из notes
	out := make([]AreaPair, 0, len(r.AreaIDs))
	for _, id := range r.AreaIDs {
		out = append(out, AreaPair{ID: id, Name: id})
	}
	return out
}

type AreaPair struct {
	ID   string
	Name string
}

func (r *ResumeProfileRepo) Upsert(ctx context.Context, userID uint, status, sourceMaterialID string, targetRoles []string, experienceLevel *string, areaIDs []string, salaryMin *int, currency *string, workFormat []string, skillsTop []string, educationLevel, notes *string, confidence, confirmedFields map[string]interface{}) (int64, error) {
	confJSON, _ := json.Marshal(confidence)
	if confJSON == nil {
		confJSON = []byte("{}")
	}
	cfJSON, _ := json.Marshal(confirmedFields)
	if cfJSON == nil {
		cfJSON = []byte("{}")
	}

	var salaryVal interface{}
	if salaryMin != nil {
		salaryVal = *salaryMin
	}
	if targetRoles == nil {
		targetRoles = []string{}
	}
	if areaIDs == nil {
		areaIDs = []string{}
	}
	if workFormat == nil {
		workFormat = []string{}
	}
	if skillsTop == nil {
		skillsTop = []string{}
	}

	var ver int64
	err := database.DB.QueryRow(ctx,
		`INSERT INTO resume_profiles (
			user_id, status, source_material_id, target_roles, experience_level, area_ids,
			salary_min, currency, work_format, skills_top, education_level, notes,
			confidence, confirmed_fields, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 1, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			source_material_id = EXCLUDED.source_material_id,
			target_roles = EXCLUDED.target_roles,
			experience_level = EXCLUDED.experience_level,
			area_ids = EXCLUDED.area_ids,
			salary_min = EXCLUDED.salary_min,
			currency = EXCLUDED.currency,
			work_format = EXCLUDED.work_format,
			skills_top = EXCLUDED.skills_top,
			education_level = EXCLUDED.education_level,
			notes = EXCLUDED.notes,
			confidence = EXCLUDED.confidence,
			confirmed_fields = EXCLUDED.confirmed_fields,
			version = resume_profiles.version + 1,
			updated_at = NOW()
		RETURNING version`,
		userID, status, nullStr(sourceMaterialID), targetRoles, experienceLevel, areaIDs,
		salaryVal, currency, workFormat, skillsTop, educationLevel, notes,
		confJSON, cfJSON).Scan(&ver)
	if err != nil {
		return 0, err
	}
	return ver, nil
}

func (r *ResumeProfileRepo) Patch(ctx context.Context, userID uint, patch map[string]interface{}, setConfirmedFields, setConfidence map[string]interface{}, status *string) (int64, error) {
	updates := []string{"version = resume_profiles.version + 1", "updated_at = NOW()"}
	args := []interface{}{}
	argNum := 1

	if v, ok := patch["target_roles"]; ok && v != nil {
		updates = append(updates, "target_roles = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["experience_level"]; ok {
		updates = append(updates, "experience_level = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["area_ids"]; ok && v != nil {
		updates = append(updates, "area_ids = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["salary_min"]; ok {
		updates = append(updates, "salary_min = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["currency"]; ok {
		updates = append(updates, "currency = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["work_format"]; ok {
		updates = append(updates, "work_format = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["skills_top"]; ok && v != nil {
		updates = append(updates, "skills_top = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["education_level"]; ok {
		updates = append(updates, "education_level = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if v, ok := patch["notes"]; ok {
		updates = append(updates, "notes = $"+strconv.Itoa(argNum))
		args = append(args, v)
		argNum++
	}
	if setConfirmedFields != nil {
		cfJSON, _ := json.Marshal(setConfirmedFields)
		updates = append(updates, "confirmed_fields = COALESCE(resume_profiles.confirmed_fields, '{}'::jsonb) || $"+strconv.Itoa(argNum)+"::jsonb")
		args = append(args, cfJSON)
		argNum++
	}
	if setConfidence != nil {
		confJSON, _ := json.Marshal(setConfidence)
		updates = append(updates, "confidence = COALESCE(resume_profiles.confidence, '{}'::jsonb) || $"+strconv.Itoa(argNum)+"::jsonb")
		args = append(args, confJSON)
		argNum++
	}
	if status != nil {
		updates = append(updates, "status = $"+strconv.Itoa(argNum))
		args = append(args, *status)
		argNum++
	}

	args = append(args, userID)
	q := "UPDATE resume_profiles SET " + strings.Join(updates, ", ") + " WHERE user_id = $"+strconv.Itoa(argNum) + " RETURNING version"
	var ver int64
	err := database.DB.QueryRow(ctx, q, args...).Scan(&ver)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("resume profile not found")
		}
		return 0, err
	}
	return ver, nil
}

func (r *ResumeProfileRepo) GetByUserID(ctx context.Context, userID uint) (*ResumeProfileRow, error) {
	var status, sourceMaterialID, experienceLevel, currency, educationLevel, notes *string
	var salaryMin *int
	var targetRolesStr, areaIDsStr, workFormatStr, skillsTopStr string
	var confidenceJSON, confirmedFieldsJSON []byte
	var version int64

	err := database.DB.QueryRow(ctx,
		`SELECT status, source_material_id,
		 COALESCE(array_to_string(target_roles, E'\x01'), ''), COALESCE(array_to_string(area_ids, E'\x01'), ''),
		 experience_level, salary_min, currency,
		 COALESCE(array_to_string(work_format, E'\x01'), ''), COALESCE(array_to_string(skills_top, E'\x01'), ''),
		 education_level, notes, confidence, confirmed_fields, version
		 FROM resume_profiles WHERE user_id = $1`, userID).
		Scan(&status, &sourceMaterialID, &targetRolesStr, &areaIDsStr, &experienceLevel, &salaryMin, &currency,
			&workFormatStr, &skillsTopStr, &educationLevel, &notes, &confidenceJSON, &confirmedFieldsJSON, &version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var confidence, confirmedFields map[string]interface{}
	_ = json.Unmarshal(confidenceJSON, &confidence)
	_ = json.Unmarshal(confirmedFieldsJSON, &confirmedFields)

	row := &ResumeProfileRow{
		UserID:           userID,
		Status:           ptrStr(status, "DRAFT"),
		SourceMaterialID: sourceMaterialID,
		TargetRoles:      splitArr(targetRolesStr),
		ExperienceLevel:  experienceLevel,
		AreaIDs:          splitArr(areaIDsStr),
		SalaryMin:        salaryMin,
		Currency:         currency,
		WorkFormat:       splitArr(workFormatStr),
		SkillsTop:        splitArr(skillsTopStr),
		EducationLevel:   educationLevel,
		Notes:            notes,
		Version:          version,
	}
	row.Confidence = mapFromInterfaceFloat64(confidence)
	row.ConfirmedFields = mapFromInterfaceBool(confirmedFields)
	return row, nil
}

func ptrStr(s *string, def string) string {
	if s != nil && *s != "" {
		return *s
	}
	return def
}

func splitArr(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\x01")
}

type ResumeProfileRepo struct{}

func NewResumeProfileRepo() *ResumeProfileRepo {
	return &ResumeProfileRepo{}
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func mapFromInterfaceFloat64(m map[string]interface{}) map[string]float64 {
	out := make(map[string]float64)
	for k, v := range m {
		if f, ok := v.(float64); ok {
			out[k] = f
		}
	}
	return out
}

func mapFromInterfaceBool(m map[string]interface{}) map[string]bool {
	out := make(map[string]bool)
	for k, v := range m {
		if b, ok := v.(bool); ok {
			out[k] = b
		}
	}
	return out
}
