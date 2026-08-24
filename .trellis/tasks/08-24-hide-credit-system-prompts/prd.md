# Hide provider system prompts from credit detail responses

## Goal

阻止客户通过 API key 查询积分消耗记录时看到 NewAPI 为上游 LLM 追加的渠道系统提示词；NewAPI 服务端和 `task-dashboard` 前端各自过滤，形成双重防线，同时保留必要的额度、token 和计费摘要。

## User Value

渠道系统提示词属于服务方核心资产。客户仍可查询自己的使用量和消费记录，但不能从自助查询接口或 dashboard 详情还原服务端注入内容。

## Confirmed Facts

- 客户使用 `GET /api/log/token`，通过 `TokenAuthReadOnly` 按 token 查询日志；该接口调用 `model.GetLogByTokenId`。
- `model.formatUserLogs` 当前只移除 `other.admin_info`、`other.audit_info` 和 `other.stream_status`，`content` 及其余 `other` 字段仍会返回。
- `task-dashboard/src/api/client.ts` 将该接口结果解析为 `UsageLogItem[]`；`CreditRecordsPage` 会直接显示 `log.content`，并展示过滤后的最多四个 `other` 条目。
- NewAPI 的 LLM 计费日志主要由 `service.PostTextConsumeQuota` / `PostAudioConsumeQuota` 写入，当前代码没有在 token 日志读取边界集中处理渠道系统提示词。
- 渠道系统提示词在 relay 层由 `ChannelSetting.SystemPrompt` 注入请求；该内容不应成为普通客户日志的公开字段。
- `graphify-out/GRAPH_REPORT.md` 在当前工作区不存在，无法提供图谱报告；本次数据流已通过路由、控制器、模型和 dashboard 调用链直接核对。
- 已用用户授权的只读 API key 查询线上 `/api/log/token`：受影响的 `xqyc` 消费记录在 `other.po`（param override audit）中携带了完整的 `prepend messages` JSON，其中包含 NewAPI 注入的 system message；该字段可直接暴露核心提示词。

## Requirements

- NewAPI 对 token-scoped 自助日志响应做服务端脱敏，禁止返回渠道系统提示词及其可还原的内部 prompt 字段；管理员日志查询行为不在本次范围内改变。
- 服务端脱敏不得改变额度、计费、token 数量、时间、模型脱敏和退款识别所需的公开字段。
- `task-dashboard` 在 API 客户端或展示层再做同等过滤，即使后端未来误返回敏感字段，也不能在积分详情 UI 展示。
- 过滤逻辑覆盖 LLM 类消费记录和历史已写入记录；不依赖只修改新日志写入，因为历史数据也必须安全。
- 不记录、打印或提交用户提供的 API key；不新增外部依赖或 API key 查询参数。
- 增加最小回归测试：后端验证 token 日志响应不含敏感 prompt；前端验证敏感 `content`/`other` 不渲染且正常计费详情仍保留。
- `other.po` 作为参数覆盖审计信息仅供管理员排障，不得出现在 token 自助日志响应或客户 dashboard 详情中；保留 `model_price`、倍率、token/cache 等非 prompt 计费元数据。

## Confirmed Product Decision

- 服务端在 `formatUserLogs` 的普通用户序列化边界删除 `other.po`，因此 `/api/log/token` 和用户自查日志均不会返回参数覆盖审计内容；管理员 `GetAllLogs` 行为保持不变。
- `task-dashboard` 在解析 `other` 时同步丢弃 `po`，即使后端回滚或 Worker 缓存返回旧数据，积分详情 UI 也不渲染该字段。

## Acceptance Criteria

- [x] `/api/log/token` 的客户响应不包含 `other.po` 或其中的渠道系统提示词。
- [x] 正常额度、token、退款和公开计费元数据仍可被 dashboard 使用。
- [x] `task-dashboard` 二次过滤后，`po` 不会出现在积分详情 DOM 中。
- [x] 后端和前端定向回归测试通过；两个仓库的现有测试/构建检查通过。
- [x] 未修改管理员日志可见性、渠道实际请求注入逻辑或其他非本任务数据流。

## Notes

- 验证记录：`go test -p 1 ./model -run '^TestFormatUserLogs' -count=1`、dashboard `bun run test`（39 项）和 `bun run build` 通过；`make verify-patches` 的 001–013 重放及定向测试通过，但临时 worktree 的双前端构建最终因资源/超时被 SIGTERM；当前环境未安装 `graphify`。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
