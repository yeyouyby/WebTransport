# Phase 2: Transport

## 目标
- 为视频与漫画建立差异化传输轨道：
  - 漫画走可靠流（Stream）。
  - 视频走不可靠数据报（Datagram）。
- 为未来 fork quic-go 注入 Brutal 拥塞控制预留稳定接口。

## 交付物
- `internal/transport/transport.go`
  - `Session` 与 `Provider` 抽象。
  - `CCPolicy` 与 `BrutalConfig` 配置模型。
- `internal/transport/webtransport_stub.go`
  - 可运行的占位实现，便于上层业务先行联调。

## 验收标准
- 上层业务不依赖具体 quic-go 实现细节，可通过接口切换 provider。
- 可通过配置切换拥塞控制策略（default/brutal）。

## 备注
- 真正“无丢包降窗”的 Brutal 算法需在 fork 的 quic-go `internal/congestion` 中实现。
- 本阶段先完成工程接口与参数流，避免后续大范围重构。
