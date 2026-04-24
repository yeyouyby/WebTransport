# 用户指令记忆

本文件记录了用户的指令、偏好和教导，用于在未来的交互中提供参考。

## 格式

### 用户指令条目
用户指令条目应遵循以下格式：

[用户指令摘要]
- Date: [YYYY-MM-DD]
- Context: [提及的场景或时间]
- Instructions:
  - [用户教导或指示的内容，逐行描述]

### 项目知识条目
Agent 在任务执行过程中发现的条目应遵循以下格式：

[项目知识摘要]
- Date: [YYYY-MM-DD]
- Context: Agent 在执行 [具体任务描述] 时发现
- Category: [代码结构|代码模式|代码生成|构建方法|测试方法|依赖关系|环境配置]
- Instructions:
  - [具体的知识点，逐行描述]

## 去重策略
- 添加新条目前，检查是否存在相似或相同的指令
- 若发现重复，跳过新条目或与已有条目合并
- 合并时，更新上下文或日期信息
- 这有助于避免冗余条目，保持记忆文件整洁

## 条目

[按上述格式记录的记忆条目]

[全矩阵转密分发网关架构基线]
- Date: 2026-04-21
- Context: Agent 在执行“梳理后端开发技术说明书”任务时发现
- Category: 代码结构
- Instructions:
  - 后端核心按四个模块实现：Crypto 与流式解密、GDrive 存储调度、QUIC/WebTransport 传输、HTTP/2 降级与反连接合并。
  - 流转发要求零拷贝与流式解密，避免 `io.ReadAll` 造成大块堆内存驻留。
  - 传输层需区分可靠流与不可靠数据报，并支持在受限网络下回退到 HTTP/2 多连接策略。

[按阶段持续推进实现]
- Date: 2026-04-21
- Context: 用户要求“先写 tasklist 与 phase1~4 文档，再按顺序持续编码到 phase4 完成”
- Instructions:
  - 先产出任务与阶段文档，再按阶段顺序实现代码。
  - 任务目标是完整推进到 Phase 4，而非只做 POC。

[先做外部依赖联调后代码评审]
- Date: 2026-04-21
- Context: 用户要求“收尾外部依赖集成联调，然后开始 review 代码”
- Instructions:
  - 先完成外部依赖真实接入，再进入代码评审阶段。

[要求完整实现并书面汇报进度]
- Date: 2026-04-21
- Context: 用户要求“确保所有功能完整实现，并书面化汇报开发进度”
- Instructions:
  - 继续补齐关键缺口并加强联调验证。
  - 输出结构化的阶段进度报告给用户。

[Brutal 拥塞策略启用方式]
- Date: 2026-04-21
- Context: Agent 在执行“quic-go fork 集成联调”时发现
- Category: 代码生成
- Instructions:
  - 项目通过 `go.mod` 的 `replace github.com/quic-go/quic-go => ./third_party/quic-go-brutal` 接入本地 fork。
  - Brutal 模式由环境变量控制：`QUIC_GO_BRUTAL_ENABLED`、`QUIC_GO_BRUTAL_TARGET_BPS`、`QUIC_GO_BRUTAL_RTT_MS`。

[要求制定并执行 phase1~4fix]
- Date: 2026-04-21
- Context: 用户要求“制定修补计划 phase1~4fix 然后执行”
- Instructions:
  - 先形成分阶段修补计划文档，再按阶段顺序实施。

[要求恢复分片并支持多媒体高性能]
- Date: 2026-04-21
- Context: 用户要求“把图片和视频的服务器分片以及 js 脚本还原搞好，并升级支持图片/视频/音频高性能传输”
- Instructions:
  - 分片管理与前端脚本要覆盖图片、视频、音频。
  - 方案要考虑高性能分段拉取、并发与格式适配。

[有下一步就继续执行]
- Date: 2026-04-24
- Context: 用户要求“Continue if you have next steps, or stop and ask for clarification if you are unsure how to proceed.”
- Instructions:
  - 若我有明确后续步骤，直接继续执行。
  - 仅在确实不明确或被阻塞时再提出澄清问题。
