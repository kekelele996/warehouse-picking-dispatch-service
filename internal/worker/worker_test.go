package worker

import (
	"context"
	"testing"
	"time"

	"warehouse/internal/config"
	"warehouse/internal/model"
	"warehouse/internal/repository"
	"warehouse/internal/service"
	"warehouse/internal/store"
)

func setup() (*Scheduler, *service.Service, *repository.Repository, *store.Store) {
	st := store.New()
	repo := repository.New(st)
	svc := service.New(repo, config.Load())
	sch := New(repo, svc, 500*time.Millisecond)
	return sch, svc, repo, st
}

func TestTickRetriesAndExecutes(t *testing.T) {
	sch, svc, repo, st := setup()
	st.SetStock("FRZ-01", 10)
	repo.CreatePicker(&model.Picker{ID: "p1", Name: "p1", Skills: []string{"coldchain"}, Busy: false})
	o, _ := svc.CreateOrder([]string{"FRZ-01"}, model.PriorityUrgent)
	tk, _ := svc.AssignPicker(o.ID, "p1")
	svc.ExecuteTask(tk.ID) // urgent first attempt -> failed

	retried, executed := sch.Tick(context.Background())
	if retried != 1 {
		t.Fatalf("retried=%d want 1", retried)
	}
	if executed != 1 {
		t.Fatalf("executed=%d want 1", executed)
	}

	after, _ := repo.FindTask(tk.ID)
	if after.Status != model.StatusCompleted {
		t.Fatalf("status=%q want completed (urgent fails first attempt, succeeds on retry)", after.Status)
	}
	if after.Attempts < 2 {
		t.Fatalf("attempts=%d want >=2", after.Attempts)
	}
}

func TestTickStopsWhenCancelled(t *testing.T) {
	st := store.New()
	repo := repository.New(st)
	svc := service.New(repo, config.Load())
	sch := New(repo, svc, 500*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	retried, executed := sch.Tick(ctx)
	if retried != 0 || executed != 0 {
		t.Fatalf("retried=%d executed=%d want 0/0", retried, executed)
	}
}

func TestTickHonorsCancellation(t *testing.T) {
	st := store.New()
	repo := repository.New(st)
	svc := service.New(repo, config.Load())
	sch := New(repo, svc, time.Second)

	st.SetStock("FRZ-01", 5)
	repo.CreatePicker(&model.Picker{ID: "p1", Name: "p1", Skills: []string{"coldchain"}, Busy: false})
	o, _ := svc.CreateOrder([]string{"FRZ-01"}, model.PriorityUrgent)
	tk, _ := svc.AssignPicker(o.ID, "p1")
	svc.ExecuteTask(tk.ID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	retried, executed := sch.Tick(ctx)
	if retried != 0 || executed != 0 {
		t.Fatalf("retried=%d executed=%d want 0/0 for cancelled ctx", retried, executed)
	}
}
