# HTTP/2 反连接合并验收清单

## 验收目标
- 在 UDP 被限制的环境下，浏览器访问分片域名时应建立多个独立 TCP 连接。
- 证明 SNI 精确证书策略成功阻断了 HTTP/2 Connection Coalescing。

## 环境准备
- 预置域名示例：`s1.api.com`、`s2.api.com`、`s3.api.com`。
- 每个域名使用独立证书文件，不含互相 SAN。
- 网关已启用 `GetCertificate` 精确 SNI 路由。

## 抓包工具
- Chrome `chrome://net-export`
- 或 Wireshark / tcpdump

## 执行步骤

### 1) 强制走 H2 降级链路
- 请在云防火墙或安全组中临时关闭 UDP 443 入站与出站。
- 保持 TCP 443 正常放行。

### 2) 启动 Chrome 网络日志
- 打开 `chrome://net-export`。
- 开启日志记录并访问测试页面。
- 页面中并发请求 `https://s1.api.com/...` 与 `https://s2.api.com/...`。

### 3) 观察连接行为
- 在 net-export 中筛选 `HTTP2_SESSION` 与 `SOCKET` 事件。
- 检查不同 SNI 是否映射到不同 socket id。

### 4) Wireshark 辅助确认

```bash
# 抓取 TCP 443 流量
tcpdump -i any tcp port 443 -w h2-coalescing-check.pcap
```

## 判定标准
- 通过：
  - 不同域名请求对应不同底层 TCP 四元组。
  - 不出现跨域复用同一 H2 连接。
- 失败：
  - `s1.api.com` 与 `s2.api.com` 复用了同一个 H2 连接。
  - 证书 SAN 覆盖多个分片域导致浏览器合并连接。

## 清理步骤
- 在云防火墙或安全组中恢复 UDP 443 入站与出站规则。
