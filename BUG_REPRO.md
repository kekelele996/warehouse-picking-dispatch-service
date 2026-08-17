# BUG_REPRO

## Bug 是什么

错误链 + 上下文取消的复合缺陷：FindTask 用 %v 丢失 %w 错误链导致 errors.Is 失效；Tick 去掉 ctx.Err() 检查导致取消不生效。

## 如何触发

```bash
go test ./...
```

## 错误信息

```
--- FAIL: TestFindTaskWrapsNotFound
    repository_test.go: errors.Is(err, ErrNotFound)=false
--- FAIL: TestTickHonorsCancellation
    worker_test.go: retried=1 want 0
```
