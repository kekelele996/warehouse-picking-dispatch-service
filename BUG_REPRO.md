# BUG_REPRO

## Bug 是什么

上下文取消 + 切片 + 错误链 + 查询过滤的复合缺陷：Tick 去掉 ctx.Err() 检查且 Run 传 Background，SortByPriority 原地排序改输入，FindTask 用 %v 断链，ListOrders 返回未过滤切片。症状是取消后调度器仍跑任务、按状态过滤失效、排序改乱、errors.Is 失灵。

## 如何触发

```bash
go test ./...
```

## 错误信息

```
--- FAIL: TestTickHonorsCancellation
    worker_test.go: retried=1 executed=0 want 0/0
--- FAIL: TestSortByPriority
    util_test.go: mutated input
--- FAIL: TestFindTaskWrapsNotFound
    repository_test.go: errors.Is(err, ErrNotFound)=false
```
