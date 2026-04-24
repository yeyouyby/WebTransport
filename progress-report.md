# 全矩阵多媒体转密分发网关开发进度汇报

日期：2026-04-24  
状态：持续推进中（分片与脚本已升级到图片/视频/音频，外场实测待执行）

## 零、Phase1~4 Fix 执行结果

- 已完成 `phase1-fix.md` 到 `phase4-fix.md` 计划并按顺序执行。
- 修补后的主链路已从“组件可用”提升为“请求可流转”。
- 已完成图片/视频分片与 JS 脚本恢复，并升级到音频场景：
  - 服务端新增分片管理器与配置下发接口。
  - 前端新增 `shard-client.js` 支持按 key 选片、格式识别和并发分段 Range 拉取。

## 一、里程碑完成情况

### Phase 1 - Crypto POC
- 已完成 ChaCha20 基于绝对偏移的 `O(1)` 对齐实现。
- 已完成流式零拷贝链路：`io.Reader -> cipher.StreamReader -> io.Writer`。
- 已完成随机偏移正确性测试与基准测试。

### Phase 2 - Transport
- 已完成传输层统一抽象：可靠流与不可靠数据报分轨。
- 已完成 `stub` 与真实 `webtransport-go` Provider 双实现。
- 已完成 WebTransport 服务端组件与主进程生命周期接入。
- 已完成本地端到端联调测试（可靠流 ping/pong round-trip）。
- 已接入本地 `quic-go` fork：支持通过环境变量启用 Brutal 模式。
- 已新增 WebTransport 可靠流请求处理器，打通私有 Header 请求到区间读取响应。

### Phase 3 - Fallback
- 已完成 HTTP/2 降级服务入口及 Range 请求解析。
- 已完成 `tls.Config.GetCertificate` 精确 SNI 证书匹配。
- 已完成证书匹配与 Range 解析测试。
- 已将 `/fallback` 从探针回显升级为可注入真实流式转发处理器。

### Phase 4 - Storage
- 已完成 SA 加权轮询、QPS 限流、熔断与半开恢复。
- 已完成 GDrive Range 拉取客户端与截断指数退避重试。
- 已完成重试等待对 `context` 取消的响应修复。
- 已完成预读任务构建与 `sync.Pool` 缓冲池基础实现。
- 已新增预读缓存管理器（容量上限 + TTL），并接入读取链路。

## 二、已补齐的关键质量问题

- 修复了 WebTransport 会话生命周期问题：不再提前关闭 CONNECT 响应体，避免会话被本地取消。
- 增加了 WebTransport 协议必需配置：
  - QUIC `EnableStreamResetPartialDelivery`
  - TLS `ALPN h3`
- 增加了启动期 fail-fast 校验：
  - fallback 证书映射不能为空
  - 启用 WebTransport 服务时必须配置证书文件
- 增加了配置化运行能力：支持切换 `TRANSPORT_PROVIDER=stub|webtransport` 及独立 WebTransport 服务监听参数。
- 补齐了 fork 拥塞策略路径：在 `OnCongestionEvent` 中启用 Brutal 模式时不执行丢包降窗。
- 修复了主程序中核心对象仅初始化不接线的问题，`gateway service` 已接入 fallback 与 transport 请求处理路径。

## 三、测试与验证结果

- `go test ./...`：通过
- `go test -race ./internal/transport ./internal/storage ./internal/crypto ./internal/protocol ./internal/fallback`：通过
- `go test -race ./internal/fallback ./internal/sharding ./internal/transport`：通过
- WebTransport 端到端联调测试：通过
  - 用例：`internal/transport/webtransport_integration_test.go`
  - 场景：本地自签证书 + WebTransport 会话建立 + 可靠流双向收发
- fork 模块测试：
  - `go test ./internal/congestion -run TestCubicSender -count=1`（workdir: `third_party/quic-go-brutal`）通过

## 四、外场执行资产

- 弱网压测手册：`ops/external/weak-network-pressure-test.md`
- H2 反连接合并验收清单：`ops/external/h2-anti-coalescing-checklist.md`
- 外场报告模板：`ops/external/report-template.md`
- 丢包配置脚本：`scripts/netem_loss_profile.sh`
- 压测入口脚本：`scripts/run_external_benchmark.sh`
- 运行时快照脚本：`scripts/collect_runtime_snapshot.sh`

## 五、临时替代与未完全实现（重点）

- 临时替代：`scripts/run_external_benchmark.sh` 仍为压测框架脚本，需接入你们真实 `bench-client` 可执行程序。
- 临时替代：预读当前为“内存缓存 + TTL”实现，尚未完成文档中的多级缓存与全量异构调度执行器。
- 临时替代：`shard-client.js` 已支持图片/视频/音频格式识别与并发分段拉取，但业务层二进制 Header 封装仍需接入前端真实播放器调用栈。
- 未完全实现：WebTransport 目前重点打通可靠流请求处理，Datagram 视频轨的高吞吐分片重组路径仍需独立压测服务验证。
- 未完全实现：Brutal 策略已注入“丢包不降窗”，但基于内核 pacer 的最终发包时钟调优参数仍需外场数据回灌。
- 未完成：VPS 30% 丢包吞吐实测与 H2 降级抓包验收尚未执行并回填报告。

## 六、本轮新增升级点（图片/视频/音频）

- 服务端分片配置新增 `audioShards`，并在 `/api/shards` 返回多媒体能力声明（image/video/audio formats）。
- 分片管理器新增 `ResourceAudio`，统一支持 image/video/audio 三类资源选片。
- `shard-client.js` 新增：
  - 格式/MIME 自动识别资源类型。
  - 可配置 `chunkSize` 与 `concurrency` 的并发分段拉取。
  - 统一 `fetchMedia` 入口与 `listCapabilities` 能力查询。
- 本轮回归：`go test ./...` 与关键模块 `-race` 均通过。

## 七、当前遗留项（外场依赖）

- 目标 VPS 的 30% 丢包吞吐压测仍需在真实网络环境执行并回填报告。
- H2 降级防连接合并抓包验证仍需在真实网络环境执行并归档证据。

## 八、结论

当前代码已具备从架构骨架到关键链路验证的完整实现能力，且核心模块均已可编译、可测试、可联调。下一步重点是按外场手册执行压测并回填数据，完成 SLA 实证闭环。
