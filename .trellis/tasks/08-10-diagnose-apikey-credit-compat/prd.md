# 诊断 API Key 积分与记录兼容性

## Goal

恢复 task-dashboard 在 new-api 1.0+ 上仅凭 API Key 查询积分、查询使用记录和兑换积分的兼容性，并明确故障责任边界位于 task-dashboard 还是 new-api。

## Confirmed Facts

- task-dashboard 通过 `Bearer sk-xxx` 调用 new-api 的积分与记录接口。
- 积分汇总使用 `/v1/dashboard/billing/subscription` 与 `/v1/dashboard/billing/usage`。
- 使用记录使用 `/api/log/token`；该请求失败时页面显示截图中的错误，积分汇总失败则不会覆盖使用记录错误。
- 兑换使用本地定制接口 `POST /api/token/redeem`。
- `/api/log/token` 和 `/api/token/redeem` 当前共用按客户端 IP 计数的 `CriticalRateLimit`。
- task-dashboard 生产环境由 Cloudflare Worker 统一代理这些请求到 `https://api.xinbao-ai.com`。
- 业务不开放 new-api 用户注册；超级管理员创建并直接分发 1000 多个 API Key，这些 key 可能共享同一个 `user_id`，但拥有不同 `token_id`。
- 线上同一 API Key 直连 `/api/log/token` 返回 200，经 task-dashboard 代理返回 429；积分汇总的直连和代理请求均返回 200。
- `/api/log/token` 当前忽略 task-dashboard 已发送的分页和时间范围参数，固定返回当前 token 最近最多 1000 条日志。
- new-api 已确认允许 `https://task.xinbao-ai.com` 携带 `Authorization` 跨域调用日志接口。
- billing 汇总在 `DisplayTokenStatEnabled=true` 时按当前 `token_id` 返回额度和已用量；项目默认值为 `true`，该部署配置必须保持开启。

## Requirements

- 保持用户无需登录、仅凭 API Key 使用上述三类能力。
- 找到线上 HTTP 429 的可复现根因，区分前端请求错误、代理错误、new-api 鉴权或限流回退。
- 修复应位于共享根因处，避免仅在页面隐藏 429。
- 每把 API Key 的自助接口限流必须按 `token_id` 隔离，不能按共享的管理员 `user_id` 聚合。
- `/api/log/token` 与 `/api/token/redeem` 必须在认证前执行独立的宽松 IP 限流，即使全局 API 限流关闭也要保护 token 查询数据库。
- 每把 API Key 的积分汇总和使用记录必须按当前 `token_id` 隔离；管理员钱包仅作为现有兑换事务的一部分，不作为下游展示统计。
- task-dashboard 的积分汇总、日志和兑换请求应优先由浏览器直连 new-api；仅当 fetch 发生网络级失败时回退到 Worker，HTTP 或业务错误不得触发回退。
- Worker 回退请求应转发 Cloudflare 提供的原始客户端 IP，避免 new-api 全局 IP 限流聚合到 Worker 出口。
- `/api/log/token` 应按当前 token、分页和时间范围返回记录，维持现有数组响应合同。
- 保持现有 API 路径和响应合同兼容；不改变积分换算规则。
- 不在诊断阶段消耗用户提供的兑换码；实现验证阶段最多使用一个明确授权的测试兑换码。
- 若修改 new-api 本地定制代码，同步更新 `docs/customizations/001-api-key-self-service.md` 与 `patches/001-api-key-self-service.patch`。

## Acceptance Criteria

- [ ] 使用同一 API Key 通过 task-dashboard 查询积分汇总成功，浏览器请求不再经过 Worker 公共出口。
- [ ] 使用同一 API Key 通过 task-dashboard 查询当日使用记录成功，正常刷新不会消耗其他 API Key 的限流额度。
- [ ] 浏览器无法直连 new-api 时，积分汇总、日志和兑换能够通过 Worker 回退；HTTP 401/429 和业务错误不回退。
- [ ] 有效兑换码可通过 task-dashboard 兑换一次，用户钱包和当前 token 额度按现有规则增加。
- [ ] 同一 `user_id` 下的不同 `token_id` 拥有独立的日志和兑换限流桶。
- [ ] 全局 API 限流关闭时，认证前 IP 限流仍能阻止同一 IP 超额探测 token。
- [ ] `DisplayTokenStatEnabled=true` 时，同一管理员下不同 API Key 展示各自的总额、已用和剩余额度。
- [ ] 无效、禁用或未提供的 API Key 继续返回明确的鉴权错误。
- [ ] 相关回归测试覆盖实际失败链路，并通过最小充分验证。
- [ ] 若 new-api 有改动，`make verify-patches` 通过且运行 `graphify update .`。

## Out of Scope

- 修改积分兑美元比例或 quota 换算比例。
- 回填旧任务、旧日志或历史兑换数据。
- 重构 task-dashboard UI 或用户注册体系。
- 改变全站通用限流策略。

## Open Questions

- new-api 变更尚未提交或部署；线上 429、真实 IP 分桶和兑换流程需在部署后验证。

## Confirmed Decisions

- task-dashboard 的积分汇总、日志和兑换请求优先直接调用 `https://api.xinbao-ai.com`，网络失败时回退同源 Worker。
- API Key 自助接口按 `token_id` 限流，不按共享的管理员 `user_id` 限流。
- 日志和兑换在 API Key 鉴权前额外执行独立 IP 底线限流，默认 120 次/60 秒且不受全局 API 限流开关影响。
