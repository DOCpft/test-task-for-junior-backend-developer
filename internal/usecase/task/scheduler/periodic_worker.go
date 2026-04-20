package task

import (
	"context"
	"log"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	"example.com/taskservice/internal/domain/task/periodicity"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

// PeriodicWorker отвечает за автосоздание задач по расписанию
// Можно запускать как отдельную горутину
func PeriodicWorker(ctx context.Context, repo taskusecase.Repository, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			// Получаем все задачи с периодичностью
			tasks, err := repo.List(ctx)
			if err != nil {
				log.Printf("periodic worker: list error: %v", err)
				continue
			}
			for _, t := range tasks {
				if t.Periodicity == nil {
					continue
				}
				strat, err := periodicity.Factory(t.Periodicity)
				if err != nil {
					log.Printf("periodic worker: strategy error: %v", err)
					continue
				}
				// Например, ищем даты на ближайшие сутки
				occurs, err := strat.Occurrences(now, now.Add(24*time.Hour))
				if err != nil {
					log.Printf("periodic worker: occurrences error: %v", err)
					continue
				}
				for _, occ := range occurs {
					dateStr := occ.Format("2006-01-02")
					existing, err := repo.FindByTemplateAndDate(ctx, t.ID, dateStr)
					if err != nil {
						log.Printf("periodic worker: find error: %v", err)
						continue
					}
					if existing != nil {
						log.Printf("periodic worker: task already exists for template %d on date %s", t.ID, dateStr)
						continue // Уже есть задача на эту дату
					}
					// Создать новую задачу-экземпляр
					scheduledFor := occ
					newTask := t // копируем шаблон
					newTask.ID = 0
					newTask.Status = taskdomain.StatusNew
					newTask.Periodicity = nil // экземпляр не должен быть периодичным
					newTask.ScheduledFor = &scheduledFor
					newTask.CreatedAt = time.Now().UTC()
					newTask.UpdatedAt = newTask.CreatedAt
					_, err = repo.Create(ctx, &newTask)

					if err != nil {
						log.Printf("periodic worker: create error: %v", err)
					}
					log.Printf("periodic worker: created task from template %d for date %s", t.ID, dateStr)
				}
			}
		}
	}
}
