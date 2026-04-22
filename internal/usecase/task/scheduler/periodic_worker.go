package task

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	taskdomain "example.com/taskservice/internal/domain/task"
	"example.com/taskservice/internal/domain/task/periodicity"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

// PeriodicWorker отвечает за автосоздание задач по расписанию
// Можно запускать как отдельную горутину
func PeriodicWorker(ctx context.Context, repo taskusecase.Repository, interval time.Duration) {
	nextRun := time.Now().Add(interval)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			now := time.Now()
			if now.Before(nextRun) {
				sleepDuration := nextRun.Sub(now)
				select {
				case <-time.After(sleepDuration):
				case <-ctx.Done():
					return
				}
			}

			// Обработка
			processPeriodicTasks(ctx, repo)

			// Рассчитываем следующее время запуска
			nextRun = nextRun.Add(interval)
			if nextRun.Before(time.Now()) {
				// Если отстали, сбрасываем на следующий интервал
				nextRun = time.Now().Add(interval)
			}
		}
	}
}

func processPeriodicTasks(ctx context.Context, repo taskusecase.Repository) {
	now := time.Now().UTC()

	// Получаем только периодические задачи
	tasks, err := repo.ListPeriodic(ctx)
	if err != nil {
		log.Printf("periodic worker: list periodic error: %v", err)
		return
	}
	for _, t := range tasks {
		if t.LastRunAt == nil {
			// Инициализируем LastRunAt, если не установлено
			t.LastRunAt = &t.CreatedAt
			_, err := repo.Update(ctx, t)
			if err != nil {
				log.Printf("periodic worker: update last_run_at error: %v", err)
				continue
			}
		}

		params := &periodicity.Params{
			Type:    periodicity.PeriodicityTypeRRule,
			RRule:   t.Periodicity.RRule,
			ExDates: t.Periodicity.ExDates,
			RDates:  t.Periodicity.RDates,
		}
		strat, err := periodicity.Factory(params)
		if err != nil {
			log.Printf("periodic worker: strategy error: %v", err)
			continue
		}

		// Вычисляем следующую дату после последнего запуска
		next, err := strat.NextAfter(*t.LastRunAt)
		if err != nil {
			log.Printf("periodic worker: next after error: %v", err)
			continue
		}
		if next.IsZero() {
			// Нет больше повторений
			continue
		}

		if next.After(now) {
			// Ещё не время
			continue
		}

		// Создаём экземпляр на дату next
		scheduledFor := next
		newTask := *t // копируем шаблон
		newTask.ID = 0
		newTask.Status = taskdomain.StatusNew
		parentID := t.ID
		newTask.ParentID = &parentID
		newTask.Periodicity = nil // экземпляр не должен быть периодичным
		newTask.ScheduledFor = &scheduledFor
		newTask.LastRunAt = nil // экземпляр не имеет last_run_at
		newTask.CreatedAt = time.Now().UTC()
		newTask.UpdatedAt = newTask.CreatedAt

		_, err = repo.Create(ctx, &newTask)
		if err != nil {
			// Игнорируем ошибку дубликата
			if !isDuplicateError(err) {
				log.Printf("periodic worker: create error: %v", err)
			}
			// Даже если дубликат, обновляем LastRunAt
		} else {
			log.Printf("periodic worker: created task from template %d for date %s", t.ID, scheduledFor.Format("2006-01-02"))
		}

		// Обновляем LastRunAt на next
		t.LastRunAt = &next
		_, err = repo.Update(ctx, t)
		if err != nil {
			log.Printf("periodic worker: update last_run_at error: %v", err)
		}
	}
}

func isDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return strings.Contains(err.Error(), "duplicate key value")
}
