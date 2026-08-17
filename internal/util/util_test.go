package util

import (
	"testing"

	"warehouse/internal/model"
)

func mkOrder(id string, status model.Status, p model.Priority) *model.Order {
	return &model.Order{ID: id, Status: status, Priority: p}
}

func TestFilterByStatusNoAliasing(t *testing.T) {
	orders := []*model.Order{
		mkOrder("a", model.StatusPending, model.PriorityLow),
		mkOrder("b", model.StatusCompleted, model.PriorityLow),
		mkOrder("c", model.StatusPending, model.PriorityLow),
	}
	got := FilterByStatus(orders, model.StatusPending)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if orders[0].ID != "a" || orders[1].ID != "b" || orders[2].ID != "c" {
		t.Fatalf("corrupted input: %v %v %v", orders[0].ID, orders[1].ID, orders[2].ID)
	}
}

func TestFilterTasksByStatus(t *testing.T) {
	tasks := []*model.PickTask{
		{ID: "t1", Status: model.StatusPending},
		{ID: "t2", Status: model.StatusFailed},
		{ID: "t3", Status: model.StatusPending},
	}
	got := FilterTasksByStatus(tasks, model.StatusPending)
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
}

func TestSortByPriority(t *testing.T) {
	orders := []*model.Order{
		mkOrder("a", model.StatusPending, model.PriorityLow),
		mkOrder("b", model.StatusPending, model.PriorityUrgent),
		mkOrder("c", model.StatusPending, model.PriorityHigh),
	}
	got := SortByPriority(orders)
	want := []string{"b", "c", "a"}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("got[%d]=%s want %s", i, got[i].ID, want[i])
		}
	}
	if orders[0].ID != "a" {
		t.Fatal("mutated input")
	}
}

func TestBatchNoAliasing(t *testing.T) {
	ids := []string{"1", "2", "3", "4", "5"}
	chunks := Batch(ids, 2)
	chunks[0][0] = "MUTATED"
	if ids[0] != "1" {
		t.Fatal("Batch corrupted input")
	}
}

func TestSKUCategory(t *testing.T) {
	cases := map[string]string{
		"FRZ-01": "fresh",
		"GLS-02": "fragile",
		"BLK-03": "bulk",
		"HZM-04": "hazmat",
		"OTHER":  "default",
	}
	for in, want := range cases {
		if got := SKUCategory(in); got != want {
			t.Fatalf("SKUCategory(%q)=%q want %q", in, got, want)
		}
	}
}
