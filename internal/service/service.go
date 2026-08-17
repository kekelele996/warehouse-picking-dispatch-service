package service

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"warehouse/internal/config"
	"warehouse/internal/model"
	"warehouse/internal/repository"
	"warehouse/internal/util"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrTaskNotFound      = errors.New("task not found")
	ErrValidation        = errors.New("invalid input")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrNoAvailablePicker = errors.New("no available picker")
	ErrInsufficientStock = errors.New("insufficient stock")
)

var idCounter atomic.Int64

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idCounter.Add(1))
}

// Dispatcher 根据 SKU 品类给出需要的技能标签。
type Dispatcher interface {
	SkillFor(sku string) string
}

// SkillDispatcher 是 Dispatcher 的默认实现。
type SkillDispatcher struct {
	mu       sync.RWMutex
	routes   map[string]string
	fallback string
}

// NewSkillDispatcher 构造一个 SkillDispatcher；routes 为空时自动初始化。
func NewSkillDispatcher(routes map[string]string) *SkillDispatcher {
	if routes == nil {
		routes = map[string]string{}
	}
	return &SkillDispatcher{routes: routes, fallback: "general"}
}

func (d *SkillDispatcher) SkillFor(sku string) string {
	cat := util.SKUCategory(sku)
	d.mu.RLock()
	if skill, ok := d.routes[cat]; ok {
		d.mu.RUnlock()
		return skill
	}
	d.mu.RUnlock()

	// 未命中时回填 fallback，写操作单独加写锁，避免与并发读竞争触发 panic。
	d.mu.Lock()
	defer d.mu.Unlock()
	// double-check：可能在升级锁期间已被其他 goroutine 写入。
	if skill, ok := d.routes[cat]; ok {
		return skill
	}
	d.routes[cat] = d.fallback
	return d.fallback
}

// Service 是业务逻辑层。
type Service struct {
	repo       *repository.Repository
	cfg        *config.Config
	dispatcher Dispatcher
}

func New(repo *repository.Repository, cfg *config.Config) *Service {
	return &Service{
		repo:       repo,
		cfg:        cfg,
		dispatcher: NewSkillDispatcher(cfg.ZoneRoutes),
	}
}

func (s *Service) CreateOrder(skus []string, priority model.Priority) (*model.Order, error) {
	if len(skus) == 0 {
		return nil, ErrValidation
	}
	if priority < model.PriorityLow || priority > model.PriorityUrgent {
		return nil, ErrValidation
	}
	now := time.Now()
	o := &model.Order{
		ID:        generateID("order"),
		SKUs:      append([]string(nil), skus...),
		Priority:  priority,
		Status:    model.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.CreateOrder(o)
}

func (s *Service) FindOrder(id string) (*model.Order, error) {
	o, err := s.repo.FindOrder(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order %s: %w", id, ErrOrderNotFound)
		}
		return nil, fmt.Errorf("lookup order %s: %w", id, err)
	}
	return o, nil
}

// ListOrders 返回全部或按状态过滤后的订单，按优先级排序。
func (s *Service) ListOrders(status *model.Status) ([]*model.Order, error) {
	orders, err := s.repo.ListOrders()
	if err != nil {
		return nil, err
	}
	if status == nil {
		return util.SortByPriority(orders), nil
	}
	filtered := util.FilterByStatus(orders, *status)
	return util.SortByPriority(filtered), nil
}

// Stats 返回各状态订单数量，供仪表盘展示；同一份快照上连续过滤。
func (s *Service) Stats() map[model.Status]int {
	orders, _ := s.repo.ListOrders()
	result := map[model.Status]int{}
	for _, st := range []model.Status{
		model.StatusPending,
		model.StatusAssigned,
		model.StatusPicking,
		model.StatusRetrying,
		model.StatusCompleted,
		model.StatusFailed,
	} {
		result[st] = len(util.FilterByStatus(orders, st))
	}
	return result
}

// AssignPicker 把订单派给拣货员：校验库存可预留、技能匹配后置为 assigned，并创建任务。
func (s *Service) AssignPicker(orderID, pickerID string) (*model.PickTask, error) {
	o, err := s.repo.FindOrder(orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("order %s: %w", orderID, ErrOrderNotFound)
		}
		return nil, err
	}
	if !model.CanTransition(o.Status, model.StatusAssigned) {
		return nil, fmt.Errorf("order %s from %s: %w", orderID, o.Status, ErrInvalidTransition)
	}
	p, err := s.repo.FindPicker(pickerID)
	if err != nil {
		if errors.Is(err, repository.ErrPickerNotFound) {
			return nil, fmt.Errorf("picker %s: %w", pickerID, ErrNoAvailablePicker)
		}
		return nil, err
	}
	if p.Busy {
		return nil, ErrNoAvailablePicker
	}
	required := s.dispatcher.SkillFor(o.SKUs[0])
	if !p.HasSkill(required) {
		return nil, ErrNoAvailablePicker
	}

	// 原子地把订单从 pending 置为 assigned：用条件更新保证同一订单并发派单时只有一个成功，
	// 失败者不会再去预留库存，避免库存被重复扣减（“扣两遍”）。
	now := time.Now()
	claimed, ok := s.repo.UpdateOrderIf(orderID, model.StatusPending, func(oo *model.Order) {
		oo.Status = model.StatusAssigned
		oo.PickerID = pickerID
		oo.UpdatedAt = now
	})
	if !ok {
		// 订单已被并发占用或状态已不可流转，按非法状态迁移报错。
		return nil, fmt.Errorf("order %s from %s: %w", orderID, o.Status, ErrInvalidTransition)
	}

	// 批量原子预留所有 SKU；任一不足则整体回滚，并把订单状态回退到 pending。
	items := make(map[string]int, len(claimed.SKUs))
	for _, sku := range claimed.SKUs {
		items[sku]++
	}
	if err := s.repo.ReserveBatch(items); err != nil {
		s.repo.UpdateOrder(orderID, func(oo *model.Order) {
			oo.Status = model.StatusPending
			oo.PickerID = ""
			oo.UpdatedAt = now
		})
		if errors.Is(err, repository.ErrInsufficientStock) {
			return nil, fmt.Errorf("order %s: %w", orderID, ErrInsufficientStock)
		}
		return nil, err
	}

	t := &model.PickTask{
		ID:          generateID("task"),
		OrderID:     orderID,
		Status:      model.StatusAssigned,
		PickerID:    pickerID,
		Attempts:    0,
		MaxAttempts: s.cfg.RetryLimit,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := s.repo.CreateTask(t); err != nil {
		// 创建任务失败：回滚库存与订单状态。
		s.repo.ReleaseStockBatch(items)
		s.repo.UpdateOrder(orderID, func(oo *model.Order) {
			oo.Status = model.StatusPending
			oo.PickerID = ""
			oo.UpdatedAt = now
		})
		return nil, err
	}
	return t, nil
}

// ExecuteTask 模拟执行任务：assigned -> picking -> completed/failed。
func (s *Service) ExecuteTask(taskID string) (*model.PickTask, error) {
	t, err := s.repo.FindTask(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("task %s: %w", taskID, ErrTaskNotFound)
		}
		return nil, err
	}
	if t.Status == model.StatusPicking {
		return s.completeTask(t)
	}
	if !model.CanTransition(t.Status, model.StatusPicking) {
		return nil, fmt.Errorf("task %s from %s: %w", taskID, t.Status, ErrInvalidTransition)
	}
	// 进入 picking 并累加尝试次数；用更新后的副本继续判定，避免读到陈旧的 Attempts。
	updated, err := s.repo.UpdateTask(taskID, func(tt *model.PickTask) {
		tt.Status = model.StatusPicking
		tt.Attempts++
		tt.UpdatedAt = time.Now()
	})
	if err != nil {
		return nil, err
	}
	return s.completeTask(updated)
}

func (s *Service) completeTask(t *model.PickTask) (*model.PickTask, error) {
	order, err := s.repo.FindOrder(t.OrderID)
	if err != nil {
		return nil, err
	}
	// 紧急订单首次执行需人工复核：Attempts 在进入 picking 时已自增，首次即 ==1。
	fail := order.Priority == model.PriorityUrgent && t.Attempts == 1
	if fail {
		if _, err := s.repo.UpdateTask(t.ID, func(tt *model.PickTask) {
			tt.Status = model.StatusFailed
			tt.LastError = "urgent order requires review"
			tt.UpdatedAt = time.Now()
		}); err != nil {
			return nil, err
		}
		// 释放已预留库存。
		for _, sku := range order.SKUs {
			s.repo.ReleaseStock(sku, 1)
		}
		return s.repo.FindTask(t.ID)
	}
	if _, err := s.repo.UpdateTask(t.ID, func(tt *model.PickTask) {
		tt.Status = model.StatusCompleted
		tt.LastError = ""
		tt.UpdatedAt = time.Now()
	}); err != nil {
		return nil, err
	}
	return s.repo.FindTask(t.ID)
}

// RetryTask 把失败任务置为 retrying，等待下次调度再次执行。
func (s *Service) RetryTask(taskID string) (*model.PickTask, error) {
	t, err := s.repo.FindTask(taskID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("task %s: %w", taskID, ErrTaskNotFound)
		}
		return nil, err
	}
	if !model.CanTransition(t.Status, model.StatusRetrying) {
		return nil, fmt.Errorf("task %s from %s: %w", taskID, t.Status, ErrInvalidTransition)
	}
	return s.repo.UpdateTask(taskID, func(tt *model.PickTask) {
		tt.Status = model.StatusRetrying
		tt.UpdatedAt = time.Now()
	})
}

// ActiveCount 并发统计仍在处理中的订单数。
func (s *Service) ActiveCount() int {
	orders, _ := s.repo.ListOrders()
	if len(orders) == 0 {
		return 0
	}

	var wg sync.WaitGroup
	var count atomic.Int64
	step := (len(orders) + 3) / 4
	if step < 1 {
		step = 1
	}
	for i := 0; i < len(orders); i += step {
		end := i + step
		if end > len(orders) {
			end = len(orders)
		}
		wg.Add(1)
		go func(chunk []*model.Order) {
			defer wg.Done()
			for _, o := range chunk {
				if model.ActiveStatuses[o.Status] {
					count.Add(1)
				}
			}
		}(orders[i:end])
	}
	wg.Wait()
	return int(count.Load())
}

// PickBatch 按优先级取前 n 张待处理订单，供 worker 批量调度。
func (s *Service) PickBatch(n int) []*model.Order {
	orders, _ := s.repo.ListOrders()
	pending := util.FilterByStatus(orders, model.StatusPending)
	return util.TopByPriority(pending, n)
}
