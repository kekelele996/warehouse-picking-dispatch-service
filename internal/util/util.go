package util

import (
	"sort"

	"warehouse/internal/model"
)

// FilterByStatus 返回状态等于 status 的工单子集，结果是新切片，不影响入参。
func FilterByStatus(orders []*model.Order, status model.Status) []*model.Order {
	out := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		if o.Status == status {
			out = append(out, o)
		}
	}
	return out
}

// FilterActive 返回仍在处理中的工单子集，结果是新切片。
func FilterActive(orders []*model.Order) []*model.Order {
	out := make([]*model.Order, 0, len(orders))
	for _, o := range orders {
		if model.ActiveStatuses[o.Status] {
			out = append(out, o)
		}
	}
	return out
}

// FilterTasksByStatus 返回状态等于 status 的任务子集，结果是新切片。
func FilterTasksByStatus(tasks []*model.PickTask, status model.Status) []*model.PickTask {
	out := make([]*model.PickTask, 0, len(tasks))
	for _, t := range tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out
}

// SortByPriority 按优先级从高到低排序，返回新切片，不影响入参。
func SortByPriority(orders []*model.Order) []*model.Order {
	out := make([]*model.Order, len(orders))
	copy(out, orders)
	sort.SliceStable(out, func(i, j int) bool {
		return model.PriorityRank(out[i].Priority) > model.PriorityRank(out[j].Priority)
	})
	return out
}

// TopByPriority 返回优先级最高的 n 张订单。
func TopByPriority(orders []*model.Order, n int) []*model.Order {
	if n <= 0 {
		return []*model.Order{}
	}
	sorted := SortByPriority(orders)
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

// Batch 把 ids 分成最多 size 一批，结果是新切片。
func Batch(ids []string, size int) [][]string {
	if size <= 0 {
		size = 1
	}
	chunks := make([][]string, 0, (len(ids)+size-1)/size)
	for i := 0; i < len(ids); i += size {
		end := i + size
		if end > len(ids) {
			end = len(ids)
		}
		chunk := make([]string, end-i)
		copy(chunk, ids[i:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}

// SKUCategory 从 SKU 编号推导品类，供派单路由使用。
func SKUCategory(sku string) string {
	if len(sku) >= 3 {
		switch sku[:3] {
		case "FRZ":
			return "fresh"
		case "GLS":
			return "fragile"
		case "BLK":
			return "bulk"
		case "HZM":
			return "hazmat"
		}
	}
	return "default"
}
