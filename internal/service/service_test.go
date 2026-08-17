package service

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"warehouse/internal/config"
	"warehouse/internal/model"
	"warehouse/internal/repository"
	"warehouse/internal/store"
)

func newServiceWithStore() (*Service, *store.Store) {
	st := store.New()
	return New(repository.New(st), config.Load()), st
}

func seedPicker(s *Service, id string, skills []string, busy bool) {
	s.repo.CreatePicker(&model.Picker{ID: id, Name: id, Skills: skills, Busy: busy})
}

func TestCreateAndFind(t *testing.T) {
	s, _ := newServiceWithStore()
	o, err := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != model.StatusPending {
		t.Fatalf("status=%q want pending", o.Status)
	}
	got, err := s.FindOrder(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.SKUs) != 1 || got.SKUs[0] != "FRZ-01" {
		t.Fatalf("skus=%v", got.SKUs)
	}
}

func TestFindMissingWraps(t *testing.T) {
	s, _ := newServiceWithStore()
	if _, err := s.FindOrder("missing"); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("err=%v want ErrOrderNotFound", err)
	}
}

func TestAssignPickerReservesStock(t *testing.T) {
	s, st := newServiceWithStore()
	st.SetStock("FRZ-01", 3)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	o, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	tk, err := s.AssignPicker(o.ID, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != model.StatusAssigned || tk.PickerID != "p1" {
		t.Fatalf("task=%+v", tk)
	}
	if got := st.GetStock("FRZ-01"); got != 2 {
		t.Fatalf("stock=%d want 2", got)
	}
}

func TestAssignRejectsWrongSkill(t *testing.T) {
	s, st := newServiceWithStore()
	st.SetStock("FRZ-01", 3)
	seedPicker(s, "p1", []string{"forklift"}, false)
	o, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	if _, err := s.AssignPicker(o.ID, "p1"); !errors.Is(err, ErrNoAvailablePicker) {
		t.Fatalf("err=%v want ErrNoAvailablePicker", err)
	}
}

func TestExecuteLifecycle(t *testing.T) {
	s, st := newServiceWithStore()
	st.SetStock("GLS-01", 5)
	seedPicker(s, "p1", []string{"fragile-cert"}, false)
	o, _ := s.CreateOrder([]string{"GLS-01"}, model.PriorityMedium)
	tk, _ := s.AssignPicker(o.ID, "p1")
	got, err := s.ExecuteTask(tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusCompleted {
		t.Fatalf("status=%q want completed", got.Status)
	}
}

func TestRetryLifecycle(t *testing.T) {
	s, st := newServiceWithStore()
	st.SetStock("FRZ-01", 5)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	o, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityUrgent)
	tk, _ := s.AssignPicker(o.ID, "p1")
	if _, err := s.ExecuteTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	failed, _ := s.repo.FindTask(tk.ID)
	if failed.Status != model.StatusFailed {
		t.Fatalf("status=%q want failed", failed.Status)
	}
	if _, err := s.RetryTask(tk.ID); err != nil {
		t.Fatal(err)
	}
	retried, _ := s.repo.FindTask(tk.ID)
	if retried.Status != model.StatusRetrying {
		t.Fatalf("status=%q want retrying", retried.Status)
	}
}

func TestActiveCount(t *testing.T) {
	s, st := newServiceWithStore()
	o1, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	s.CreateOrder([]string{"FRZ-02"}, model.PriorityHigh)
	s.CreateOrder([]string{"FRZ-03"}, model.PriorityHigh)
	st.SetStock("FRZ-01", 5)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	s.AssignPicker(o1.ID, "p1")
	if got := s.ActiveCount(); got != 3 {
		t.Fatalf("ActiveCount=%d want 3", got)
	}
}

func TestStats(t *testing.T) {
	s, st := newServiceWithStore()
	o1, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	s.CreateOrder([]string{"FRZ-02"}, model.PriorityHigh)
	s.CreateOrder([]string{"FRZ-03"}, model.PriorityHigh)
	st.SetStock("FRZ-01", 5)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	s.AssignPicker(o1.ID, "p1")
	stats := s.Stats()
	if stats[model.StatusPending] != 2 {
		t.Fatalf("pending=%d want 2", stats[model.StatusPending])
	}
	if stats[model.StatusAssigned] != 1 {
		t.Fatalf("assigned=%d want 1", stats[model.StatusAssigned])
	}
}

func TestConcurrentAssignAndRead(t *testing.T) {
	s, st := newServiceWithStore()
	for i := 0; i < 120; i++ {
		sku := fmt.Sprintf("FRZ-%03d", i)
		st.SetStock(sku, 3)
		if _, err := s.CreateOrder([]string{sku}, model.PriorityHigh); err != nil {
			t.Fatal(err)
		}
	}
	seedPicker(s, "p1", []string{"coldchain"}, false)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		orders, _ := s.ListOrders(nil)
		for _, o := range orders {
			if _, err := s.AssignPicker(o.ID, "p1"); err == nil {
				tasks, _ := s.repo.ListTasks()
				for _, tk := range tasks {
					if tk.OrderID == o.ID {
						_, _ = s.ExecuteTask(tk.ID)
					}
				}
			}
		}
	}()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 60; j++ {
				_ = s.ActiveCount()
			}
		}()
	}
	wg.Wait()
}

func TestPickBatch(t *testing.T) {
	s, _ := newServiceWithStore()
	s.CreateOrder([]string{"FRZ-01"}, model.PriorityLow)
	o2, _ := s.CreateOrder([]string{"FRZ-02"}, model.PriorityUrgent)
	o3, _ := s.CreateOrder([]string{"FRZ-03"}, model.PriorityMedium)
	batch := s.PickBatch(2)
	if len(batch) != 2 || batch[0].ID != o2.ID || batch[1].ID != o3.ID {
		t.Fatalf("batch=%v", batch)
	}
}

func TestZoneRouteFallback(t *testing.T) {
	t.Setenv("WAREHOUSE_ZONE_ROUTES", "invalid")
	s, st := newServiceWithStore()
	st.SetStock("FRZ-01", 3)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	o, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityHigh)
	if _, err := s.AssignPicker(o.ID, "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestListOrdersByStatus(t *testing.T) {
	s, st := newServiceWithStore()
	o1, _ := s.CreateOrder([]string{"FRZ-01"}, model.PriorityLow)
	o2, _ := s.CreateOrder([]string{"FRZ-02"}, model.PriorityUrgent)
	st.SetStock("FRZ-02", 3)
	seedPicker(s, "p1", []string{"coldchain"}, false)
	s.AssignPicker(o2.ID, "p1")

	status := model.StatusPending
	got, err := s.ListOrders(&status)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d want 1", len(got))
	}
	if got[0].ID != o1.ID {
		t.Fatalf("got[0]=%s want %s", got[0].ID, o1.ID)
	}
}
