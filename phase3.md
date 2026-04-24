# Phase 3: Fallback

## 目标
- 在 UDP/QUIC 不可用时提供 HTTP/2 降级路径。
- 通过 SNI 精确证书映射阻断浏览器连接合并（Connection Coalescing）。

## 交付物
- `internal/fallback/cert_manager.go`
  - 精确域名证书管理与查找（不做泛域名回退）。
- `internal/fallback/server.go`
  - HTTP/2 TLS 服务器构建。
  - Range 请求参数提取与下游处理回调。
- `internal/fallback/cert_manager_test.go`
  - SNI 精确匹配行为测试。

## 验收标准
- `GetCertificate` 仅按 `ServerName` 返回同名证书，错域直接报错。
- 降级链路可处理标准 Range 请求并输出字节流响应。

## 备注
- 生产验证需结合 Chrome `net-export` 抓包确认多 TCP 连接已建立。
