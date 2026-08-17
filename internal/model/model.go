package model

import "time"

// Priority 表示订单优先级，数值越大越紧急。
type Priority int

const (
	PriorityLow Priority = iota + 1
	PriorityMedium
	PriorityHigh
	PriorityUrgent
)

// Status 表示拣货任务状态。
type Status string

const (
	StatusPending   Status = "pending"
	StatusAssigned  Status = "assigned"
	StatusPicking   Status = "picking"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusRetrying  Status = "retrying"
)

// ActiveStatuses 是「还在处理中」的状态集合。
var ActiveStatuses = map[Status]bool{
	StatusPending:  true,
	StatusAssigned: true,
	StatusPicking:  true,
	StatusRetrying: true,
}

// Order 是一张出库订单。
type Order struct {
	ID        string
	SKUs      []string
	Priority  Priority
	Status    Status
	PickerID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PickTask 是一个拣货任务。
type PickTask struct {
	ID          string
	OrderID     string
	Status      Status
	PickerID    string
	Attempts    int
	MaxAttempts int
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Picker 是一名拣货员。
type Picker struct {
	ID     string
	Name   string
	Skills []string
	Busy   bool
}

// transitions 是合法状态迁移表。
var transitions = map[Status][]Status{
	StatusPending:   {StatusAssigned, StatusFailed},
	StatusAssigned:  {StatusPicking, StatusFailed},
	StatusPicking:   {StatusCompleted, StatusFailed},
	StatusFailed:    {StatusRetrying},
	StatusRetrying:  {StatusPicking},
	StatusCompleted: {},
}

// CanTransition 判断 from 能否直接迁移到 to。
func CanTransition(from, to Status) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// PriorityRank 返回优先级对应的整数权重。
func PriorityRank(p Priority) int {
	return int(p)
}

// HasSkill 判断拣货员是否具备指定技能标签。
func (p *Picker) HasSkill(skill string) bool {
	for _, s := range p.Skills {
		if s == skill {
			return true
		}
	}
	return false
}
