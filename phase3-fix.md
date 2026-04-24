# Phase 3 Fix - Fallback 真正转发

## 目标
- 将 HTTP/2 fallback 从“Range 回显探针”升级为“真实回源解密转发入口”。
- 保留原有证书精确匹配与 fail-fast 策略。

## 修补项
- 扩展 `fallback.ServerConfig` 注入 `RangeHandler`。
- `/fallback` 路径优先执行真实 handler，回退到 probe 仅用于调试。
- 主程序把 gateway service 的 `StreamRange` 接入 fallback handler。

## 验收标准
- 通过单测验证 handler 注入路径和错误路径。
- 在无 handler 时保持向后兼容行为。
