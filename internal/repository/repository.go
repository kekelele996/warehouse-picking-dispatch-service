package repository

import (
	"errors"
	"fmt"

	"warehouse/internal/model"
	"warehouse/internal/store"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrPickerNotFound    = errors.New("picker not found")
)

// Repository 在内存 Store 之上提供带错误语义的数据访问层。
type Repository struct {
	store *store.Store
}

func New(s *store.Store) *Repository {
	return &Repository{store: s}
}

func (r *Repository) CreateOrder(o *model.Order) (*model.Order, error) {
	if err := r.store.PutOrder(o); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return nil, fmt.Errorf("create order %s: %w", o.ID, ErrAlreadyExists)
		}
		return nil, fmt.Errorf("create order %s: %w", o.ID, err)
	}
	return o, nil
}

func (r *Repository) FindOrder(id string) (*model.Order, error) {
	o, err := r.store.GetOrder(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("order %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("order %s: %w", id, err)
	}
	return o, nil
}

func (r *Repository) ListOrders() ([]*model.Order, error) {
	return r.store.ListOrders(), nil
}

func (r *Repository) UpdateOrder(id string, fn func(*model.Order)) (*model.Order, error) {
	o, err := r.store.UpdateOrder(id, fn)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("order %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("order %s: %w", id, err)
	}
	return o, nil
}

func (r *Repository) CreateTask(t *model.PickTask) (*model.PickTask, error) {
	if err := r.store.PutTask(t); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return nil, fmt.Errorf("create task %s: %w", t.ID, ErrAlreadyExists)
		}
		return nil, fmt.Errorf("create task %s: %w", t.ID, err)
	}
	return t, nil
}

func (r *Repository) FindTask(id string) (*model.PickTask, error) {
	t, err := r.store.GetTask(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("task %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("task %s: %w", id, err)
	}
	return t, nil
}

func (r *Repository) ListTasks() ([]*model.PickTask, error) {
	return r.store.ListTasks(), nil
}

func (r *Repository) UpdateTask(id string, fn func(*model.PickTask)) (*model.PickTask, error) {
	t, err := r.store.UpdateTask(id, fn)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("task %s: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("task %s: %w", id, err)
	}
	return t, nil
}

func (r *Repository) ReserveStock(sku string, qty int) error {
	if !r.store.Reserve(sku, qty) {
		return fmt.Errorf("sku %s: %w", sku, ErrInsufficientStock)
	}
	return nil
}

func (r *Repository) ReleaseStock(sku string, qty int) {
	r.store.Release(sku, qty)
}

func (r *Repository) StockSnapshot() map[string]int {
	return r.store.StockSnapshot()
}

func (r *Repository) CreatePicker(p *model.Picker) (*model.Picker, error) {
	if err := r.store.PutPicker(p); err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return nil, fmt.Errorf("picker %s: %w", p.ID, ErrAlreadyExists)
		}
		return nil, fmt.Errorf("picker %s: %w", p.ID, err)
	}
	return p, nil
}

func (r *Repository) FindPicker(id string) (*model.Picker, error) {
	p, err := r.store.GetPicker(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("picker %s: %w", id, ErrPickerNotFound)
		}
		return nil, fmt.Errorf("picker %s: %w", id, err)
	}
	return p, nil
}

func (r *Repository) ListPickers() ([]*model.Picker, error) {
	return r.store.ListPickers(), nil
}
