# warehouse-picking-dispatch-service

一个用 Go 写的仓储拣货调度服务，演示 `handler → service → repository → store → model` 分层、库存预留并发控制、拣货任务状态机与后台调度 worker 的常见写法。

## 功能

- 创建出库订单，按优先级与 SKU 品类派单给空闲拣货员
- 库存预留与释放，拣货任务状态流转：pending → assigned → picking → completed / failed → retrying
- 后台调度器定时捞取到期任务执行，失败自动重试
- 内存存储，读写锁保护，支持按状态 / 拣货员 / 订单查询

## 目录结构

```
cmd/warehouse/         程序入口
internal/config/       环境变量配置
internal/model/        模型定义与状态机
internal/store/        内存存储（订单 / 任务 / 库存 / 拣货员）
internal/repository/   数据访问层（错误包装）
internal/service/      业务逻辑（建单 / 派单 / 执行 / 库存预留 / 查询）
internal/worker/       后台调度 worker（重试）
internal/handler/      HTTP 接口
internal/util/         过滤 / 排序 / 分批等工具函数
```

## 运行与测试

```bash
go build ./...              # 编译
go test ./...               # 全量测试
go run ./cmd/warehouse      # 启动 HTTP 服务
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `WAREHOUSE_WORKERS` | 调度 worker 数量 | `4` |
| `WAREHOUSE_RETRY_LIMIT` | 任务失败重试上限 | `3` |
| `WAREHOUSE_RESERVE_TIMEOUT_MS` | 库存预留超时（毫秒） | `2000` |
| `WAREHOUSE_POLL_INTERVAL_MS` | 调度轮询间隔（毫秒） | `500` |

## 技术栈

- Go 1.22
