package store

import (
	"testing"

	"warehouse/internal/model"
)

func TestOrderCopySemantics(t *testing.T) {
	s := New()
	o := &model.Order{ID: "o1", SKUs: []string{"a", "b"}}
	if err := s.PutOrder(o); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetOrder("o1")
	got.SKUs[0] = "MUTATED"
	again, _ := s.GetOrder("o1")
	if again.SKUs[0] != "a" {
		t.Fatal("GetOrder returned internal reference")
	}
}

func TestListOrdersCopies(t *testing.T) {
	s := New()
	for _, id := range []string{"b", "a", "c"} {
		if err := s.PutOrder(&model.Order{ID: id}); err != nil {
			t.Fatal(err)
		}
	}
	first := s.ListOrders()
	for i := range first {
		first[i].ID = "x"
	}
	again := s.ListOrders()
	if again[0].ID == "x" {
		t.Fatal("ListOrders returned internal reference")
	}
}

func TestTaskCopySemantics(t *testing.T) {
	s := New()
	if err := s.PutTask(&model.PickTask{ID: "t1", OrderID: "o1"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetTask("t1")
	got.OrderID = "MUTATED"
	again, _ := s.GetTask("t1")
	if again.OrderID != "o1" {
		t.Fatal("GetTask returned internal reference")
	}
}

func TestReserveRelease(t *testing.T) {
	s := New()
	s.SetStock("sku1", 10)
	if !s.Reserve("sku1", 3) {
		t.Fatal("reserve should succeed")
	}
	if s.GetStock("sku1") != 7 {
		t.Fatalf("stock=%d want 7", s.GetStock("sku1"))
	}
	if s.Reserve("sku1", 100) {
		t.Fatal("reserve should fail when insufficient")
	}
	s.Release("sku1", 3)
	if s.GetStock("sku1") != 10 {
		t.Fatalf("stock=%d want 10", s.GetStock("sku1"))
	}
}

func TestStockSnapshotCopy(t *testing.T) {
	s := New()
	s.SetStock("sku1", 5)
	snap := s.StockSnapshot()
	snap["sku1"] = 999
	if s.GetStock("sku1") != 5 {
		t.Fatal("StockSnapshot returned internal map")
	}
}
