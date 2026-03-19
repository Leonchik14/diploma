package service

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	reScoreLine      = regexp.MustCompile(`(?i)^\s*SCORE\s*:\s*([0-9]+(?:[.,][0-9]+)?)\s*$`)
	reScoreAnywhere  = regexp.MustCompile(`(?i)SCORE\s*:\s*([0-9]+(?:[.,][0-9]+)?)`)
	reOcenka       = regexp.MustCompile(`(?i)ОЦЕНКА\s*:\s*([0-9]+(?:[.,][0-9]+)?)`)
	reSlash10      = regexp.MustCompile(`([0-9]+(?:[.,][0-9]+)?)\s*/\s*10`)
	reIz10         = regexp.MustCompile(`([0-9]+(?:[.,][0-9]+)?)\s*из\s*10`)
	rePrepNumbered = regexp.MustCompile(`\n{2,}\d+\.\s*`)
	reMultiNL      = regexp.MustCompile(`\n{3,}`)
)

func parseFloatScore(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 || f > 10 {
		return 0, false
	}
	return f, true
}

// parseReviewResumeScoreAndBody: первая строка SCORE:X (обязательный формат), далее рекомендации.
func parseReviewResumeScoreAndBody(raw string) (score float64, body string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, ""
	}
	lines := strings.Split(raw, "\n")
	if m := reScoreLine.FindStringSubmatch(strings.TrimSpace(lines[0])); len(m) == 2 {
		if sc, ok := parseFloatScore(m[1]); ok {
			score = sc
			rest := lines[1:]
			for len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
				rest = rest[1:]
			}
			body = sanitizeCoachFacingText(strings.Join(rest, "\n"))
			return score, strings.TrimSpace(body)
		}
	}
	// fallback: ищем оценку в тексте
	score = extractScoreFromText(raw)
	body = stripLeadingScoreSection(raw)
	body = sanitizeCoachFacingText(body)
	return score, strings.TrimSpace(body)
}

func extractScoreFromText(text string) float64 {
	for _, re := range []*regexp.Regexp{reScoreAnywhere, reOcenka, reSlash10, reIz10} {
		if m := re.FindStringSubmatch(text); len(m) == 2 {
			if f, ok := parseFloatScore(m[1]); ok {
				return f
			}
		}
	}
	// строка вида "7.5/10" в начале
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if m := reSlash10.FindStringSubmatch(line); len(m) == 2 {
			if f, ok := parseFloatScore(m[1]); ok {
				return f
			}
		}
	}
	return 0
}

func stripLeadingScoreSection(raw string) string {
	lines := strings.Split(raw, "\n")
	var out []string
	skipUntilContent := true
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if skipUntilContent {
			if t == "" {
				continue
			}
			if reScoreLine.MatchString(t) || reOcenka.MatchString(t) ||
				strings.EqualFold(t, "РЕКОМЕНДАЦИИ:") || strings.EqualFold(t, "РЕКОМЕНДАЦИИ") {
				continue
			}
			if reSlash10.MatchString(t) && len(t) < 20 {
				continue
			}
			skipUntilContent = false
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// sanitizeCoachFacingText: убирает иероглифы и мусорные символы, оставляет кириллицу, латиницу, цифры, пунктуацию.
func sanitizeCoachFacingText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r':
			b.WriteByte('\n')
			prevSpace = false
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			prevSpace = false
			continue
		case unicode.Is(unicode.Cyrillic, r) || unicode.Is(unicode.Latin, r):
			b.WriteRune(r)
			prevSpace = false
		case unicode.IsDigit(r):
			b.WriteRune(r)
			prevSpace = false
		case unicode.IsSpace(r):
			if b.Len() > 0 && !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		case strings.ContainsRune(`.,;:!?-—–()«»"'%/+*=#&`, r):
			b.WriteRune(r)
			prevSpace = false
		default:
			if r < 128 && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
				b.WriteRune(r)
			}
			prevSpace = false
		}
	}
	out := strings.TrimSpace(reMultiNL.ReplaceAllString(b.String(), "\n\n"))
	return out
}

// normalizePrepareForVacancyText: убирает «\n\n1. \n\n2.», оставляет простые переводы строк.
func normalizePrepareForVacancyText(s string) string {
	s = strings.TrimSpace(s)
	prev := ""
	for s != prev {
		prev = s
		s = rePrepNumbered.ReplaceAllString(s, "\n")
	}
	s = regexp.MustCompile(`\n{2,}`).ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}
