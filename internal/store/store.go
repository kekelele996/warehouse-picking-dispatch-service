package store

import (
	"errors"
	"sync"

	"warehouse/internal/model"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

// Store 是内存存储，用读写锁保护所有数据。
type Store struct {
	mu        sync.RWMutex
	orders    map[string]*model.Order
	orderIDs  []string
	tasks     map[string]*model.PickTask
	taskIDs   []string
	inventory map[string]int
	pickers   map[string]*model.Picker
	pickerIDs []string
}

func New() *Store {
	return &Store{
		orders:    make(map[string]*model.Order),
		orderIDs:  []string{},
		tasks:     make(map[string]*model.PickTask),
		taskIDs:   []string{},
		inventory: make(map[string]int),
		pickers:   make(map[string]*model.Picker),
		pickerIDs: []string{},
	}
}

func (s *Store) PutOrder(o *model.Order) error {
	if o == nil || o.ID == "" {
		return errors.New("invalid order")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[o.ID]; ok {
		return ErrAlreadyExists
	}
	s.orders[o.ID] = o
	s.orderIDs = append(s.orderIDs, o.ID)
	return nil
}

func (s *Store) GetOrder(id string) (*model.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	return o, nil
}

func (s *Store) ListOrders() []*model.Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Order, 0, len(s.orderIDs))
	for _, id := range s.orderIDs {
		out = append(out, s.orders[id])
	}
	return out
}

func (s *Store) OrderIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.orderIDs
}

func (s *Store) UpdateOrder(id string, fn func(*model.Order)) (*model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	fn(o)
	return o, nil
}

func (s *Store) PutTask(t *model.PickTask) error {
	if t == nil || t.ID == "" {
		return errors.New("invalid task")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; ok {
		return ErrAlreadyExists
	}
	s.tasks[t.ID] = t
	s.taskIDs = append(s.taskIDs, t.ID)
	return nil
}

func (s *Store) GetTask(id string) (*model.PickTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *Store) ListTasks() []*model.PickTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.PickTask, 0, len(s.taskIDs))
	for _, id := range s.taskIDs {
		out = append(out, s.tasks[id])
	}
	return out
}

func (s *Store) TaskIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taskIDs
}

func (s *Store) UpdateTask(id string, fn func(*model.PickTask)) (*model.PickTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, ErrNotFound
	}
	fn(t)
	return t, nil
}

func (s *Store) SetStock(sku string, qty int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventory[sku] = qty
}

func (s *Store) GetStock(sku string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inventory[sku]
}

// Reserve 尝试预留库存；成功返回 true，不足返回 false。
func (s *Store) Reserve(sku string, qty int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inventory[sku] < qty {
		return false
	}
	s.inventory[sku] -= qty
	return true
}

func (s *Store) Release(sku string, qty int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventory[sku] += qty
}

func (s *Store) StockSnapshot() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inventory
}

func (s *Store) PutPicker(p *model.Picker) error {
	if p == nil || p.ID == "" {
		return errors.New("invalid picker")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pickers[p.ID]; ok {
		return ErrAlreadyExists
	}
	s.pickers[p.ID] = p
	s.pickerIDs = append(s.pickerIDs, p.ID)
	return nil
}

func (s *Store) GetPicker(id string) (*model.Picker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pickers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Store) ListPickers() []*model.Picker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Picker, 0, len(s.pickerIDs))
	for _, id := range s.pickerIDs {
		out = append(out, s.pickers[id])
	}
	return out
}
