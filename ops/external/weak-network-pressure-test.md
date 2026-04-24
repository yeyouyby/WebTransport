# 外场弱网压测执行手册

## 目标
- 在真实海外 VPS 环境下验证 30% 丢包时视频分发吞吐是否维持在 50Mbps 到 100Mbps。
- 采集 TTFB、吞吐、错误率、CPU、内存与 GC 指标，形成可复现对比报告。

## 前置条件
- VPS 已放行 `443/tcp` 与 `443/udp`。
- 已配置 `TLS_CERT_MAP` 与 WebTransport 服务证书。
- 服务启动参数中已启用 Brutal fork 开关。

```bash
# 启用 Brutal fork 开关
export QUIC_GO_BRUTAL_ENABLED=true

# 目标带宽 50Mbps
export QUIC_GO_BRUTAL_TARGET_BPS=50000000

# 估算 RTT 120ms
export QUIC_GO_BRUTAL_RTT_MS=120
```

## 步骤 1: 配置 30% 丢包

使用脚本 `scripts/netem_loss_profile.sh` 注入丢包。

```bash
# 设置出口网卡为 eth0，注入 30% 丢包
bash scripts/netem_loss_profile.sh apply eth0 30
```

## 步骤 2: 启动服务

```bash
# 启动网关服务
go run ./cmd/gateway
```

## 步骤 3: 运行压测

```bash
# 运行 60 秒吞吐压测
bash scripts/run_external_benchmark.sh --duration 60 --concurrency 1 --mode datagram

# 运行 60 秒 TTFB 压测
bash scripts/run_external_benchmark.sh --duration 60 --concurrency 8 --mode stream
```

## 步骤 4: 采集运行时指标

```bash
# 导出进程资源采样
bash scripts/collect_runtime_snapshot.sh --pid <gateway_pid> --seconds 60
```

## 步骤 5: 清理弱网规则

```bash
# 清理网卡 qdisc
bash scripts/netem_loss_profile.sh clear eth0
```

## 验收阈值
- Throughput: 单客户端稳定 50Mbps 到 100Mbps。
- TTFB: P99 小于 200ms。
- API 错误率: 小于 0.1%。
- 内存: 不出现持续增长与明显 GC 风暴。

## 结果归档
- 将原始日志与采样文件写入 `ops/external/results/YYYYMMDD/`。
- 使用 `ops/external/report-template.md` 填写结论。
