# Phase 1 Fix - 核心链路接线

## 目标
- 将已实现的 `storage + crypto` 能力接入网关业务服务层，消除 `main` 中对象仅初始化未使用的问题。
- 形成统一的字节区间读取接口，为 Fallback 与 WebTransport 共用。

## 修补项
- 新增 `internal/gateway/service.go`：
  - `StreamRange(ctx, offset, length, writer)`：流式解密输出。
  - `ReadRange(ctx, offset, length)`：用于传输层消息回复。
- 服务内部接入 `SAPool`、`GDriveClient`、`crypto.StreamReader`。
- 增加密钥和 nonce 的配置解析与启动校验。

## 验收标准
- 主程序不再出现核心对象“仅赋值不使用”。
- 可通过单测覆盖 `ReadRange` 的缓存命中与回源路径。
