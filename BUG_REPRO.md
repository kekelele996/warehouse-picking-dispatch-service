# BUG_REPRO

## Bug 是什么

nil + 切片的复合缺陷：自定义路由解析为空时返回 nil map，SkillDispatcher 构造未兜底，SkillFor 向 nil map 写入 panic；FilterByStatus 用 s[:0] 原地复用底层数组污染调用方快照。

## 如何触发

```bash
go test ./...
```

## 错误信息

```
--- FAIL: TestZoneRouteFallback
    panic: assignment to entry in nil map
--- FAIL: TestFilterByStatusNoAliasing
    util_test.go: FilterByStatus corrupted input
```
