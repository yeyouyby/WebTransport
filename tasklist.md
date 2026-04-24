# 全矩阵多媒体转密分发网关任务清单

## Phase 1 - Crypto POC（流式去盐与零拷贝）
- [x] 建立 Go 项目骨架与模块分层。
- [x] 实现 ChaCha20 基于绝对偏移的 O(1) 对齐逻辑。
- [x] 实现 `io.Reader -> cipher.StreamReader -> io.Writer` 流式转发链路。
- [x] 增加单测与基准测试，验证偏移正确性和内存分配表现。

## Phase 2 - Transport（QUIC/WebTransport 抽象与可替换拥塞控制接口）
- [x] 设计传输层接口：可靠流与不可靠数据报分离。
- [x] 提供 QUIC/WebTransport 适配层占位实现（可替换为 fork 版本）。
- [x] 为 Brutal 拥塞控制预留策略注入点与配置项。
- [x] 提供丢包场景的压测入口与吞吐统计框架。

## Phase 3 - Fallback（HTTP/2 降级与反连接合并）
- [x] 实现 `tls.Config.GetCertificate` 精确 SNI 证书返回。
- [x] 提供 HTTP/2 降级服务入口与范围请求解析。
- [x] 增加 SNI 精确匹配单测，防止证书误路由。

## Phase 4 - Storage（GDrive SA 轮询池与异构预读池）
- [x] 实现 SA 加权轮询 + QPS 限流 + 熔断半开状态机。
- [x] 实现 GDrive Range 拉取客户端（截断指数退避 + 抖动）。
- [x] 实现视频深预读与漫画分块并发预读。
- [x] 用 `sync.Pool` 管理预读缓冲区，减少 GC 压力。

## 收尾
- [x] 提供二进制信令头编解码与 HMAC 校验模块。
- [x] 提供统一配置、启动入口和可观测指标桩代码。
- [x] 补齐真实 WebTransport 服务接入（fork quic-go 的 Brutal 算法改造仍待联调）。
- [x] 增加 WebTransport 本地端到端联调测试（可靠流 round-trip）。
- [x] 接入本地 quic-go fork（Brutal 模式基于环境变量启用）。
- [x] 补齐外场压测脚本、执行手册与报告模板。
- [x] 补齐 H2 反连接合并验证清单。
- [ ] 在目标 VPS 环境执行 30% 丢包吞吐压测与 H2 降级抓包验证并回填报告。

## Phase 1~4 Fix
- [x] Phase1-fix：新增 gateway service，打通 storage+crypto 读取主链路。
- [x] Phase2-fix：新增 WebTransport 请求处理器，接入 32B Header 编解码。
- [x] Phase3-fix：Fallback `/fallback` 接入真实 RangeHandler 转发。
- [x] Phase4-fix：新增预读缓存管理器与 TTL/容量控制，接入读取链路。

## 分片与脚本恢复
- [x] 新增图片/视频分片管理器，支持按资源类型与 key 稳定选片。
- [x] 新增 `/api/shards` 分片配置下发接口。
- [x] 新增 `/shard-client.js` 前端脚本下发与基础 Range 拉取能力。
- [x] 扩展音频分片：新增 `audioShards` 配置、服务端下发与测试覆盖。
- [x] 升级 `shard-client.js`：支持图片/视频/音频格式识别与并发分段拉取。
- [x] 回归验证：`go test ./...` 与 `go test -race ./internal/fallback ./internal/sharding ./internal/transport`。
