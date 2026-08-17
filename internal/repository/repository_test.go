package repository

import (
	"errors"
	"testing"

	"warehouse/internal/model"
	"warehouse/internal/store"
)

func TestFindOrderWrapsNotFound(t *testing.T) {
	r := New(store.New())
	_, err := r.FindOrder("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("errors.Is(err, ErrNotFound)=false, err=%v", err)
	}
}

func TestReserveStockInsufficient(t *testing.T) {
	r := New(store.New())
	r.store.SetStock("sku1", 2)
	if err := r.ReserveStock("sku1", 5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err=%v want ErrInsufficientStock", err)
	}
}

func TestFindPickerWrapsNotFound(t *testing.T) {
	r := New(store.New())
	_, err := r.FindPicker("missing")
	if !errors.Is(err, ErrPickerNotFound) {
		t.Fatalf("errors.Is(err, ErrPickerNotFound)=false, err=%v", err)
	}
}

func TestCreateTaskDuplicate(t *testing.T) {
	r := New(store.New())
	if _, err := r.CreateTask(&model.PickTask{ID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateTask(&model.PickTask{ID: "t1"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("dup err=%v", err)
	}
}

func TestFindTaskWrapsNotFound(t *testing.T) {
	r := New(store.New())
	_, err := r.FindTask("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("errors.Is(err, ErrNotFound)=false, err=%v", err)
	}
}
