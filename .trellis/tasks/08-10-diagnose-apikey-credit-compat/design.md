# 技术设计

## 结论

故障由 1.0+ 的 new-api IP 限流与 task-dashboard 公共代理共同触发。所有分发 key 可能共享超级管理员 `user_id`，因此不能改成用户级限流。修复分三层：new-api 自助接口按 `token_id` 限流；task-dashboard 的积分相关请求优先由浏览器直连 new-api；网络失败时回退 Worker，并转发 Cloudflare 提供的客户端 IP。

## 请求链路

1. 浏览器向 task-dashboard 发起相对路径请求并携带 API Key。
2. Cloudflare Worker 将积分、日志和兑换请求转发到 new-api。
3. new-api 先执行全局 API 限流，再执行路由中间件和控制器。
4. 当前日志、兑换请求在认证前按 Worker 出口 IP 共用 20 次/20 分钟的高敏限流桶，导致稳定 429。
5. 即使仅修复高敏限流，new-api 的全局 IP 限流仍会聚合 Worker 流量；因此正常请求直接访问 new-api，只有网络失败才回退 Worker。

## 修改边界

- `middleware/rate-limit.go`
  - 增加独立于全局限流开关的认证前 IP 限流，默认 120 次/60 秒。
  - 增加鉴权后按 `token_id` 计数的限流入口，不使用 `user_id`。
- `router/api-router.go`
  - 兑换：先执行认证前 IP 限流和 `TokenAuthReadOnly`，再执行 token 级高敏限流。
  - 日志：先执行认证前 IP 限流和 `TokenAuthReadOnly`，再执行 token 级查询限流。
- `controller/log.go`、`model/log.go`
  - 解析现有分页和时间范围参数。
  - 查询限定为当前 `token_id`，响应继续返回 `{ success, message, data: [] }`。
- 本地定制文档与 `001-api-key-self-service.patch`
  - 记录代理场景下的限流合同、筛选合同和验证方式。
- task-dashboard `src/api/client.ts`
  - 为积分汇总、日志和兑换请求使用可配置的 new-api 浏览器端地址。
  - 仅捕获 fetch 的网络 `TypeError` 并重试同源 Worker 路径；`ApiError` 不重试。
  - 异步图片、视频和已有 Worker 缓存链路保持不变。
- task-dashboard `worker/api-proxy.ts`
  - 从入站 `CF-Connecting-IP` 设置上游 `X-Real-IP` 和 `X-Forwarded-For`。

## 兼容性

- API 路径、认证头、响应 envelope、日志数组结构和积分换算不变。
- billing 汇总继续依赖 `DisplayTokenStatEnabled=true` 按 `token_id` 读取额度；部署检查必须确认该选项未关闭。
- SQLite、MySQL、PostgreSQL 均使用 GORM `Where/Order/Limit/Offset`。
- 浏览器直连已通过线上 CORS 预检验证，允许 `Authorization`。
- 需要分别部署 new-api 和 task-dashboard；两端可独立回滚。

## 风险与回滚

- 风险：旧版 task-dashboard 仍经过 Worker，但 new-api 的 token 级限流会先解决当前 20 次高敏限流冲突；升级前端后再消除全局 IP 桶共享。
- 风险：浏览器直接看到 new-api 地址；该地址本就是公开 API 入口，API Key 仍只发往 new-api。
- 风险：兑换直连可能已被服务端处理但响应丢失；回退重试仍由兑换码单次核销事务防止重复到账，客户端可能需要刷新确认不确定结果。
- 风险：日志筛选后每页数量会从固定最近记录变为真实分页结果，正是 task-dashboard 当前合同。
- 回滚：new-api 可恢复原路由中间件顺序和日志查询签名；task-dashboard 可恢复相对路径，不涉及迁移或数据变更。
