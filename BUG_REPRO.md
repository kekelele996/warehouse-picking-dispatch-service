# BUG_REPRO

## Bug 是什么

并发拣货相关的复合缺陷：存储层返回内部引用且去掉深拷贝，过滤用 s[:0] 原地复用底层数组，库存不足错误用 %v 丢掉 %w 错误链，并发统计无快照。症状是库存重复扣减、批量任务串数据、库存不足被当成系统错误、并发 map 读写 panic。

## 如何触发

```bash
go test ./...
# 并发类加 race
go test -race -count=20 ./...
```

## 错误信息

```
--- FAIL: TestOrderCopySemantics
    store_test.go:29: GetOrder returned internal reference
--- FAIL: TestReserveStockInsufficient
    repository_test.go:29: errors.Is(err, ErrInsufficientStock)=false
--- FAIL: TestConcurrentAssignAndRead
    testing.go:1617: race detected during execution of test
```
