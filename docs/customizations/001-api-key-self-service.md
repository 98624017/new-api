# 001-api-key-self-service

## 1. 背景

本地业务存在 API Key 二次分发、外部控制台和自动化工具直接调用的需求。上游默认的兑换码接口依赖用户登录 Session，异步任务列表也只能通过用户登录态查询，不适合只持有 `Bearer sk-xxx` 的下游客户端。

本补丁把原 `001-token-redeem-via-apikey` 和 `005-task-list-via-apikey` 合并为一个“API Key 自助能力”补丁，集中维护 API Key 免登录场景。

## 2. 目标

- 新增 API Key 认证的兑换码接口。
- 兑换成功时同时增加用户钱包和当前 token/key 额度。
- 新增 API Key 认证的异步任务列表接口。
- 任务列表只返回当前 token/key 创建的新任务。
- 保持原有 Session 登录接口行为不变。

不解决：

- 不改变管理员创建兑换码逻辑。
- 不改变原有 `/api/user/topup` 和 `/api/task/self` 行为。
- 不开放跨用户、跨 token 的任务检索能力。
- 不为补丁上线前未写入 `tasks.token_id` 的历史任务做回填。

## 3. 业务规则

兑换接口：

- 路径：`POST /api/token/redeem`
- 认证方式：`TokenAuthReadOnly()`
- 请求体与原 `TopUp` 接口保持一致。
- 兑换成功后，额度进入用户钱包。
- 兑换成功后，当前请求使用的 token/key 额度同步增加相同值。
- 充值使用记录需标明兑换到哪个 token/key 名称。

任务列表接口：

- 路径：`GET /api/task/token/self`
- 认证方式：`TokenAuthReadOnly()`
- 支持参数：`p`、`page_size` / `size`、`task_id`、`status`、`action`、`platform`、`start_timestamp`、`end_timestamp`
- 只返回当前 token 创建的任务，不返回同一用户其他 token 创建的任务。
- 列表与总数查询只基于任务表独立 `token_id` 列。
- 补丁上线前未写入独立 `token_id` 的历史任务默认查不到。

## 4. 影响范围

- `router/api-router.go`
  - 注册 `POST /api/token/redeem`
  - 注册 `GET /api/task/token/self`
- `controller/user.go`
  - 新增 `TokenRedeem`
- `controller/task.go`
  - 新增 `GetUserTokenTask`
- `controller/relay.go`
  - 新建异步任务时同步写入独立 `token_id`
- `model/redemption.go`
  - 新增 token 场景下的钱包 + token quota 联动兑换逻辑
- `model/task.go`
  - 任务表新增独立 `token_id` 字段
  - 新增按 token 查询用户任务的方法
- `controller/user_token_redeem_test.go`
  - 覆盖 API Key 兑换流程
- `controller/task_token_test.go`
  - 覆盖 API Key 任务列表过滤
- `patches/001-api-key-self-service.patch`
  - 保存本二开的可重放差异

## 5. 数据流

兑换流程：

1. 客户端携带 `Bearer sk-xxx` 调用 `POST /api/token/redeem`
2. `TokenAuthReadOnly()` 校验 token 并写入用户、token 上下文
3. `TokenRedeem` 复用原有 `topUpRequest`
4. `RedeemByToken` 在同一事务中增加用户钱包、增加当前 token 的 `remain_quota`、读取 token 名称、核销兑换码
5. 成功返回兑换额度，失败返回原有兑换失败语义

任务列表流程：

1. 客户端携带 `Bearer sk-xxx` 调用 `GET /api/task/token/self`
2. `TokenAuthReadOnly()` 写入 `id` 和 `token_id`
3. 控制器解析分页与筛选参数
4. 模型层按 `user_id + token_id` 查询任务列表和总数
5. 返回与 `/api/task/self` 一致的分页结构

## 6. 风险点

- 上游若调整 `TopUp` 的请求结构或锁逻辑，本地兑换入口需要同步。
- 上游若调整 `TokenAuthReadOnly()` 行为，两个 API Key 自助接口都可能受影响。
- 上游若调整 token quota 字段或更新方式，联动补额逻辑需要同步。
- 异步任务落库路径若漏写 `tasks.token_id`，新列表接口会漏查该任务。
- 历史任务默认不回填 `token_id`，上线前任务不会出现在新接口结果中。

## 7. 测试方案

最小验证命令：

```bash
go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask)' -count=1
```

完整二开校验：

```bash
make verify-patches
```

当前已覆盖：

- `Bearer sk-xxx` 可直接兑换。
- 成功兑换时用户钱包和当前 token 额度同时增加。
- 成功兑换时充值日志显示兑换到的 token/key 名称。
- 无效 token 返回 401。
- 无效兑换码返回业务失败。
- API Key 任务列表只返回当前 token 创建的任务。
- `task_id` 筛选参数在 API Key 任务列表中生效。

## 8. 升级关注点

- `controller/user.go` 中 `TopUp` 附近逻辑是否重构。
- `model/redemption.go` 中兑换码核销逻辑是否重构。
- `middleware/auth.go` 中 `TokenAuthReadOnly()` 是否调整。
- `controller/task.go` 中用户任务列表入参 / 返回格式是否重构。
- `model/task.go` 中任务表字段和查询函数是否重构。
- `controller/relay.go` 中异步任务落库逻辑是否调整。
