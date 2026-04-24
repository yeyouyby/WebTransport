# Phase 1: Crypto POC

## 目标
- 验证流式解密链路可在不引入大块堆内存分配的情况下完成转发。
- 验证基于绝对偏移的 O(1) 对齐逻辑在任意偏移场景下正确。

## 交付物
- `internal/crypto/stream.go`
  - `NewChaCha20AlignedStream`：根据 `offset` 计算块计数器并对齐。
  - `NewChaCha20ReaderAtOffset`：将上游 `io.Reader` 包装成可直接解密读取器。
- `internal/crypto/stream_test.go`
  - 随机偏移和长度的正确性测试。
  - 基准测试用于观察 `alloc/op` 与 `B/op`。

## 验收标准
- 对同一明文采用同 key/nonce 加密后，在任意 offset 下解密结果与明文切片一致。
- 核心链路不出现 `io.ReadAll` 或等价全量读操作。

## 备注
- 当前先以 ChaCha20 为主路径，AES-CTR 可作为后续兼容实现。
