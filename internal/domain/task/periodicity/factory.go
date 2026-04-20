package periodicity

import (
	"fmt"

	"example.com/taskservice/internal/domain/task"
)

// Factory создает нужную стратегию по типу периодичности
func Factory(p *task.Periodicity) (Strategy, error) {
	if p == nil || p.RRule != "" {
		return &RRuleStrategy{
			RRule:   p.RRule,
			ExDates: p.ExDates,
			RDates:  p.RDates,
		}, nil
	}
	// Здесь можно добавить другие типы стратегий (например, cron, custom и т.д.)
	return nil, fmt.Errorf("unsupported periodicity type")
}
