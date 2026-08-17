# BUG_REPRO

## Bug 是什么

任务状态机 + 配置 nil + 错误链的复合缺陷：retrying 到 picking 的转换边缺失，自定义路由为空时返回 nil map 且构造函数未兜底，FindOrder/FindPicker 用 %v 丢失错误链。症状是失败任务切不到 retrying、自定义路由 panic、查不存在的订单返回 500。

## 如何触发

```bash
go test ./...
```

## 错误信息

```
--- FAIL: TestCanTransition
    model_test.go:21: CanTransition(retrying,picking)=false want true
--- FAIL: TestZoneRouteFallback
    panic: assignment to entry in nil map
--- FAIL: TestGetMissingOrderReturns404
    handler_test.go:24: status=500 want 404
```
