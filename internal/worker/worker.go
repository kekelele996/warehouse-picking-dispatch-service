package worker

import (
	"context"
	"errors"
	"time"

	"warehouse/internal/model"
	"warehouse/internal/repository"
	"warehouse/internal/service"
	"warehouse/internal/util"
)

// Executor 是调度器依赖的任务执行/重试能力，由 service 层实现。
type Executor interface {
	RetryTask(taskID string) (*model.PickTask, error)
	ExecuteTask(taskID string) (*model.PickTask, error)
}

// Scheduler 是后台调度器：周期性捞取失败任务重试、执行重试中的任务。
type Scheduler struct {
	repo         *repository.Repository
	exec         Executor
	pollInterval time.Duration
}

func New(repo *repository.Repository, exec Executor, pollInterval time.Duration) *Scheduler {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	return &Scheduler{repo: repo, exec: exec, pollInterval: pollInterval}
}

// Tick 执行一轮调度：先把失败且未超限的任务置为 retrying，再执行 retrying 任务。
// 若 ctx 已取消，则立即返回，不进行任何重试或执行。
func (sch *Scheduler) Tick(ctx context.Context) (retried, executed int) {
	if err := ctx.Err(); err != nil {
		return 0, 0
	}
	tasks, err := sch.repo.ListTasks()
	if err != nil {
		return 0, 0
	}
	failed := util.FilterTasksByStatus(tasks, model.StatusFailed)
	for _, t := range failed {
		if t.Attempts < t.MaxAttempts {
			_, err := sch.exec.RetryTask(t.ID)
			if err == nil {
				retried++
			} else if errors.Is(err, service.ErrTaskNotFound) {
				continue
			}
		}
	}

	tasks, err = sch.repo.ListTasks()
	if err != nil {
		return retried, executed
	}
	for _, t := range tasks {
		if t.Status == model.StatusRetrying {
			if _, err := sch.exec.ExecuteTask(t.ID); err == nil {
				executed++
			}
		}
	}
	return retried, executed
}

// Run 周期执行 Tick，直到 ctx 被取消。
func (sch *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(sch.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			sch.Tick(ctx)
		}
	}
}
