# 实施计划

1. 添加最小回归测试，证明同一代理 IP、同一 `user_id` 下不同 `token_id` 不共享限流额度，并覆盖同一 token 超限。
2. 为 API Key 兑换和 token 日志增加独立认证前 IP 限流，再于鉴权后按 `token_id` 执行对应限流。
3. 为 token 日志查询接入 `p`、`page_size`、`start_timestamp`、`end_timestamp`，保持现有响应结构。
4. 在 task-dashboard 中将积分汇总、日志和兑换改为通过可配置地址优先直连 new-api，仅在网络失败时回退 Worker。
5. Worker 回退请求向 new-api 转发 Cloudflare 原始客户端 IP；不信任浏览器提供的转发头。
6. 补充日志筛选/分页、token 限流、直连回退条件和 Worker IP 转发回归测试。
7. 更新 `docs/customizations/001-api-key-self-service.md` 与 `patches/001-api-key-self-service.patch`。
8. 运行 new-api 相关 Go 测试、task-dashboard Vitest/构建与 `make verify-patches`。
9. 运行 `graphify update .`，复核差异中无调试输出。

## 上线验证

- 先部署 new-api，再部署 task-dashboard；用同一 API Key 确认积分相关请求从浏览器直达 new-api。
- 使用同一管理员下两把 API Key 验证限流桶相互独立。
- 确认生产 `DisplayTokenStatEnabled=true`，并验证两把 API Key 的汇总额度互不混淆。
- 使用一个用户已授权的测试兑换码通过 task-dashboard 兑换，并复查积分汇总与充值日志。

## 回滚点

- 不包含数据库迁移；若验证失败，仅回滚本任务代码、文档和补丁文件。
