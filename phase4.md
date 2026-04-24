# Phase 4: Storage

## 目标
- 构建高并发、可容错的 GDrive 拉取平面。
- 用异构预读提升视频拖拽与漫画爆发式读取性能。

## 交付物
- `internal/storage/sa_pool.go`
  - SA 加权轮询、QPS 限流、失败熔断与半开恢复。
- `internal/storage/gdrive_client.go`
  - Range 拉取、认证注入、截断指数退避。
- `internal/prefetch/pool.go`
  - `sync.Pool` 缓冲管理。
  - 视频深预读与漫画分块并发预读任务编排。

## 验收标准
- 在并发请求下，SA 调度可持续分配且单账号不会超过配置 QPS。
- 拉取失败可在限定次数内重试，且不出现长时间阻塞。
- 预读缓冲复用可观测，避免大块临时内存持续增长。

## 备注
- 真实上线前需接入实际 GDrive API、监控与压测链路进行参数校准。
