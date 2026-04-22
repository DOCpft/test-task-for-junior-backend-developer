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

func (s *RRuleStrategy) NextAfter(after time.Time) (time.Time, error) {
    rule, err := rrule.StrToRRule(s.RRule)
    if err != nil {
        return time.Time{}, err
    }

    // Получаем следующее вхождение после after
    next := rule.After(after, false)

    // Если next нулевое (IsZero), значит вхождений больше нет по RRULE
    if next.IsZero() {
        // Проверяем RDates вручную
        var candidate time.Time
        for _, rd := range s.RDates {
            t, err := time.Parse("2006-01-02", rd)
            if err == nil && t.After(after) {
                if candidate.IsZero() || t.Before(candidate) {
                    candidate = t
                }
            }
        }
        if candidate.IsZero() {
            return time.Time{}, nil // Нет больше дат
        }
        return candidate, nil
    }

    // Проверяем ExDates
    exMap := make(map[string]struct{}, len(s.ExDates))
    for _, d := range s.ExDates {
        exMap[d] = struct{}{}
    }
    if _, excluded := exMap[next.Format("2006-01-02")]; excluded {
        // Рекурсивно ищем следующее после исключённой даты
        return s.NextAfter(next)
    }

    // Проверяем, есть ли более раннее RDate
    for _, rd := range s.RDates {
        t, err := time.Parse("2006-01-02", rd)
        if err == nil && t.After(after) && t.Before(next) && !t.Equal(next) {
            // Проверяем, не исключена ли эта RDate
            if _, ex := exMap[t.Format("2006-01-02")]; !ex {
                next = t
            }
        }
    }

    return next, nil
}
