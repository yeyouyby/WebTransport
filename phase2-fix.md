# Phase 2 Fix - 传输协议闭环

## 目标
- 将 WebTransport 服务从“仅可启动”升级为“可处理真实请求并返回业务数据”。
- 实现基于私有 32B Header 的请求解析与响应。

## 修补项
- 新增传输网关处理器：
  - 接收可靠流请求并解析 `protocol.Header`。
  - 调用网关服务读取区间数据。
  - 发送带同 `RequestID` 的响应头与 payload。
- 打通 `main` 中 WebTransport 服务 handler。

## 验收标准
- 新增集成测试覆盖“请求头 -> 数据读取 -> 响应头”流程。
- 出错时返回可识别错误码并关闭会话。
