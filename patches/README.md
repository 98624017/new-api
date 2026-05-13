# 二开补丁（Custom Patches）

本目录存放在上游 new-api 基础上的二开功能补丁。每次同步上游代码后，按序号重新应用即可。

## 与文档的关系

- 每个 patch 都必须与 `docs/customizations/NNN-*.md` 一一对应
- patch 只保存代码差异
- 业务背景、规则、风险、测试方式统一记录在 `docs/customizations/`
- 新增 patch 后，必须同步更新：
  - `docs/customizations/README.md`
  - 对应的 `docs/customizations/NNN-*.md`

## 使用方法

### 校验补丁（每次合并前必须执行）

```bash
make verify-patches
```

该命令会检查：

- `patches/NNN-*.patch` 与 `docs/customizations/NNN-*.md` 是否一一对应
- `patches/README.md` 与 `docs/customizations/README.md` 是否登记了对应二开
- 所有 patch 是否能按编号顺序应用到 `upstream/main`
- 当前工作区如果有源码改动，是否同步修改了至少一个 `patches/*.patch`

### 自动应用（推荐先尝试）

```bash
# 同步上游代码后，逐个应用补丁
git apply patches/001-token-redeem-via-apikey.patch

# 如有空白差异，加 --ignore-whitespace
git apply --ignore-whitespace patches/001-token-redeem-via-apikey.patch

# 如有轻微上下文偏移，用 3way merge
git apply --3way patches/001-token-redeem-via-apikey.patch
```

### 冲突时手动恢复

如果 `git apply` 失败，根据下面每个补丁的说明手动修改即可。

---

## 001-token-redeem-via-apikey.patch

**功能**：兑换码免登录兑换 — 通过 API Key (sk-xxx) 认证兑换，供 neko-api-key-tool 等外部工具使用；兑换成功时同时补到用户钱包和当前 token/key 额度，并在充值使用记录中显示兑换到的 token/key 名称。

**背景**：上游的兑换接口 `POST /api/user/topup` 需要用户登录 Session。此补丁新增 `POST /api/token/redeem`，使用 `TokenAuthReadOnly` 中间件，允许通过 Bearer sk-xxx 免登录兑换；并在兑换成功时同步增加当前 token 的额度，方便 API Key 二次分发场景维持账实一致。充值日志会追加 token/key 名称，便于后续在使用记录中追踪具体兑换目标。

**涉及文件（5 个）**：

### 1. `controller/token_test.go`

在 token 控制器测试的临时 DB helper 中保存并恢复 `model.DB` / `model.LOG_DB`，避免本补丁新增的外部包兑换测试与现有 controller 包测试互相污染全局 DB。

### 2. `controller/user.go`

在 `TopUp` 函数后面（`type UpdateUserSettingRequest struct` 之前）插入 `TokenRedeem` 函数：

```go
// TokenRedeem 通过 API Key (sk-xxx) 认证的兑换码兑换接口。
// 与 TopUp 逻辑一致，但认证方式不同：TopUp 需要用户登录 Session，
// 而 TokenRedeem 使用 TokenAuthReadOnly 中间件，允许外部工具
// （如 neko-api-key-tool）通过 Bearer sk-xxx 免登录兑换。
func TokenRedeem(c *gin.Context) {
	userId := c.GetInt("id")
	lock := getTopUpLock(userId)
	if !lock.TryLock() {
		common.ApiErrorI18n(c, i18n.MsgUserTopUpProcessing)
		return
	}
	defer lock.Unlock()
	req := topUpRequest{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	quota, err := model.Redeem(req.Key, userId)
	if err != nil {
		if errors.Is(err, model.ErrRedeemFailed) {
			common.ApiErrorI18n(c, i18n.MsgRedeemFailed)
			return
		}
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    quota,
	})
}
```

**依赖**：复用同文件中已有的 `topUpRequest`、`getTopUpLock`，以及 `model.RedeemByToken`、`common.ApiError` 等。

### 3. `model/redemption.go`

新增 token 场景兑换逻辑 `RedeemByToken`：

- 在同一事务内增加用户钱包 quota
- 同时增加当前 token 的 `remain_quota`
- 读取当前 token 名称，并写入充值使用记录
- 一并核销兑换码

这样不会出现“钱包加了但 key 没加”的中间不一致状态。

### 4. `router/api-router.go`

在 `usageRoute` 块结束后、`redemptionRoute` 块开始前，插入一行路由注册：

```go
// 通过 API Key (sk-xxx) 免登录兑换，供外部工具（如 neko-api-key-tool）使用
apiRouter.POST("/token/redeem", middleware.CORS(), middleware.CriticalRateLimit(), middleware.TokenAuthReadOnly(), controller.TokenRedeem)
```

**定位标志**：找到 `tokenUsageRoute.GET("/", controller.GetTokenUsage)` 所在块的结束括号 `}`，在其后插入上述路由。

### 5. `controller/user_token_redeem_test.go`

补充控制器回归测试，重点验证：

- `Bearer sk-xxx` 可直接进入兑换链路
- 成功后用户钱包增加
- 成功后当前 token 的 `remain_quota` 同步增加
- 成功后的充值使用记录包含 token/key 名称

### 回归验证

建议最小验证命令：

```bash
go test ./controller -run '^TestTokenRedeem' -v
```

验证重点：

- `Bearer sk-xxx` 可直接兑换
- 成功后用户钱包增加
- 成功后当前 token 额度同步增加
- 成功后使用记录显示兑换到的 token/key 名称

---

## 002-task-refund-restore-token-quota.patch

**功能**：异步视频任务失败退款时，按环境变量开关恢复 token/key 额度。

**背景**：对于做 API Key 二次分发的用户，仅退款到钱包不够，失败任务还需要按开关决定是否恢复 token 可用额度。当前实现引入环境变量 `TASK_REFUND_RESTORE_TOKEN_QUOTA`，默认关闭；开启后，失败退款会同时恢复 token 额度。

**涉及文件（5 个）**：

### 1. `common/constants.go`

新增运行时配置变量：

```go
var TaskRefundRestoreTokenQuota bool
```

### 2. `common/init.go`

读取环境变量并记录启动日志：

```go
TaskRefundRestoreTokenQuota = GetEnvOrDefaultBool("TASK_REFUND_RESTORE_TOKEN_QUOTA", false)
```

### 3. `service/task_billing.go`

将 `RefundTaskQuota` 中的 token 恢复逻辑改为按开关执行，并在日志 other 字段记录开关状态：

```go
if common.TaskRefundRestoreTokenQuota {
	taskAdjustTokenQuota(ctx, task, -quota)
}
```

### 4. `service/task_billing_test.go`

补充两类测试：

- 开关关闭：失败后只退资金来源
- 开关开启：失败后退资金来源并恢复 token 额度
- `UpdateVideoTasks` 失败路径：命中真实视频轮询服务层后，失败任务触发退款

### 5. `scripts/seed_task_refund_fixture.go`

离线生成 `002` 容器验收所需的 fixture：

- 建立最小表结构
- 写入测试用户、测试 token、测试视频渠道
- 写入一条已预扣、待视频轮询失败的异步任务
- 任务字段显式带上：
  - 视频渠道类型
  - 真实 `task.platform`
  - 真实 `private_data.upstream_task_id`
- 提供 `inspect` 模式回读任务 / 钱包 / token 结果

### 6. `scripts/mock_video_failure_server.go`

宿主机上的本地 mock 上游服务：

- 提供 `GET /v1/videos/{task_id}` 失败响应
- 提供 `/healthz` 供脚本等待启动完成
- 提供 `/stats` 供脚本确认容器确实命中过视频轮询接口

### 7. `scripts/verify_task_refund_restore_token_quota.sh`

更接近业务路径的黑盒验收脚本，覆盖两轮场景：

- `TASK_REFUND_RESTORE_TOKEN_QUOTA=false`
- `TASK_REFUND_RESTORE_TOKEN_QUOTA=true`

脚本验证点：

- 关闭 `TASK_TIMEOUT_MINUTES`，避免落到 timeout sweep 兜底路径
- 通过 mock `/stats` 确认命中过真实视频轮询路径
- 用户登录后 `GET /api/user/self` 中的钱包额度变化
- `GET /api/usage/token/` 中的 key 剩余额度变化
- 最终任务状态为 `FAILURE`

建议验收命令：

```bash
bash scripts/verify_task_refund_restore_token_quota.sh new-api:verify-20260406
```

---

## 003-mask-billing-amounts-in-errors.patch

**功能**：下游客户端错误响应金额脱敏。

**背景**：预扣费失败、额度不足或部分上游错误文案可能带出具体金额 / 额度数值，例如 `用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900`。这些数值可能暴露本地成本价、预扣费策略或上游额度细节，不适合透传给下游客户。

**涉及文件（6 个）**：

### 1. `common/str.go`

新增 `MaskBillingAmountsForClient`，优先按计费语义标签脱敏，货币符号只作为无标签场景的保守兜底：

- `¥0.056700` -> `¥***`
- `＄0.060000` -> `＄***`
- `token remain quota: 120` -> `token remain quota: ***`
- `balance: credits 12.50` -> `balance: credits ***`
- `balance: (estimated) 12.50` -> `balance: (estimated) ***`
- `balance: tier 2 credits 12.50` -> `balance: tier 2 credits ***`
- `balance: credits 12.50 request id 123` -> `balance: credits *** request id 123`
- `need=69900` -> `need=***`

### 2. `types/error.go`

在 `ToOpenAIError` / `ToClaudeError` / `MaskSensitiveError` 中调用金额脱敏，覆盖同步 relay 的 OpenAI / Claude 风格错误响应和错误日志展示文本。

### 3. `service/error.go`

在 `TaskErrorWrapper` / `TaskErrorFromAPIError` 中调用金额脱敏，覆盖异步任务错误响应。

### 4. `common/billing_amount_mask_test.go`

覆盖中文预扣费金额、全角美元符号、自定义单位前缀、金额后带数字型元数据、英文额度标签、订阅 `need=` 文案，以及 `status_code` / `request id` 不被误伤。

### 5. `types/error_test.go`

覆盖 OpenAI / Claude 错误转换和错误日志展示文本中的金额脱敏。

### 6. `service/error_test.go`

覆盖异步任务错误包装中的金额脱敏。

### 回归验证

```bash
go test ./common -run TestMaskBillingAmountsForClient -count=1
go test ./types -run 'TestNewAPIError(To|MaskSensitiveErrorWithStatusCode)' -count=1
go test ./service -run 'Test(TaskError.*MasksBillingAmounts|ResetStatusCode)' -count=1
```

---

## 004-sora-reference-video-double-price.patch

**功能**：Sora 兼容 `/v1/videos` 请求中，白名单模型携带参考视频时按双倍计价。

**背景**：部分视频生成模型在请求体 `content` 中携带 `video_url` 参考视频时，上游成本不同于普通文生视频请求。为了避免影响其他模型，本补丁只对环境变量 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 白名单中的模型生效。

**涉及文件（4 个）**：

### 1. `relay/common/relay_info.go`

在 `TaskSubmitReq` 中新增 `Content []map[string]any`，用于保留 `/v1/videos` JSON 请求体顶层 `content` 数组。

### 2. `relay/channel/task/sora/constants.go`

新增环境变量白名单加载逻辑。默认白名单为空，不配置时任何模型都不会触发参考视频双倍计价。多个模型用英文逗号分隔：

```bash
SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS=seedance-2.0,seedance-2.0-pro
```

### 3. `main.go`

在 `godotenv.Load(".env")` 和 `common.InitEnv()` 之后调用 `soratask.ReloadReferenceVideoDoublePriceModelsFromEnv()`，确保 `.env` 中的 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 能被读取。

### 4. `relay/channel/task/sora/adaptor.go`

在 `EstimateBilling` 中追加判断：

- 模型名在环境变量白名单内
- `content` 中存在 `type == "video_url"` 或 `video_url` 字段

两个条件同时满足时返回 `video_input: 2`，由现有 `OtherRatios` 机制乘入最终额度。

### 5. `relay/channel/task/sora/adaptor_test.go`

新增回归测试：

- 环境变量包含 `seedance-2.0` 时，`seedance-2.0` + `content.video_url` 返回 `video_input: 2`
- 环境变量为空时，`seedance-2.0` + `content.video_url` 不返回 `video_input`
- 环境变量变更只有显式 reload 后才会生效，避免包级初始化提前读取 `.env` 前的空值

### 回归验证

```bash
go test ./relay/channel/task/sora
go test ./relay/common
```

---

## 005-task-list-via-apikey.patch

**功能**：通过 API Key (`Bearer sk-xxx`) 免登录查询“当前 key 创建的异步任务列表 / task_id”。

**背景**：现有项目已经支持用 API Key 查询单个异步视频任务状态，但如果客户端没有保存提交时返回的 `task_id`，就无法只靠 API Key 找回“这个 key 创建过哪些任务”。本补丁新增一个只读列表接口，供外部控制台、轮询工具或 API Key 二次分发场景使用。

**涉及文件（6 个）**：

### 1. `router/api-router.go`

在 `taskRoute` 下新增一条只读接口：

```go
taskRoute.GET("/token/self", middleware.TokenAuthReadOnly(), controller.GetUserTokenTask)
```

### 2. `controller/task.go`

新增 `GetUserTokenTask`：

- 从 `TokenAuthReadOnly()` 上下文中读取 `id` 和 `token_id`
- 复用现有分页参数和任务筛选参数
- 返回结构与 `/api/task/self` 保持一致

### 3. `model/task.go`

任务表新增独立 `token_id` 列，并新增按 token 维度查任务的方法：

- `TaskGetAllUserTokenTask`
- `TaskCountAllUserTokenTask`

同时增加兼容逻辑：

- 新任务优先使用独立 `token_id`
- 老任务若独立列为空或为 `0`，首次查询时按用户批量回填 `private_data.token_id`
- 回填完成后，列表与总数查询都只走独立 `token_id` 列，保持数据库分页

### 4. `controller/relay.go`

异步任务提交成功后，除了继续写 `private_data.token_id`，还同步写入任务表独立 `token_id` 列，保证新任务后续能走索引列查询。

### 5. `controller/task_token_test.go`

补充控制器回归测试，重点验证：

- `Bearer sk-xxx` 可直接访问新接口
- 仅返回当前 token 创建的任务
- 老任务仅有 `private_data.token_id` 时也能命中
- `task_id` 过滤参数仍然生效

### 6. `controller/user_token_redeem_test.go`

调整测试 helper，确保多用户测试场景下 `users.username` / `users.aff_code` 不会撞唯一索引。

### 回归验证

```bash
go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask)' -count=1
```

---

## 补丁维护规范

1. **文件命名**：`NNN-简短描述.patch`，按序号排列
2. **每个补丁**：在本文件中记录功能说明 + 涉及文件 + 手动恢复步骤
3. **更新上游后**：先 `git apply`，失败则按文档手动改，最后 `git diff > patches/NNN-xxx.patch` 更新补丁文件
