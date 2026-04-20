package periodicity

import (
	"time"

	"github.com/teambition/rrule-go"
)

// RRuleStrategy реализует Strategy для rrule-правил
// Использует стандарт RFC 5545

type RRuleStrategy struct {
	RRule   string
	ExDates []string
	RDates  []string
}

func (s *RRuleStrategy) Occurrences(from, to time.Time) ([]time.Time, error) {
	rule, err := rrule.StrToRRule(s.RRule)
	if err != nil {
		return nil, err
	}
	dates := rule.Between(from, to, true)

	ex := make(map[string]struct{})
	for _, d := range s.ExDates {
		ex[d] = struct{}{}
	}
	var result []time.Time
	for _, d := range dates {
		if _, skip := ex[d.Format("2006-01-02")]; !skip {
			result = append(result, d)
		}
	}
	for _, rd := range s.RDates {
		t, err := time.Parse("2006-01-02", rd)
		if err == nil && t.After(from) && t.Before(to) {
			result = append(result, t)
		}
	}
	return result, nil
}

func (s *RRuleStrategy) Type() string {
	return "rrule"
}
