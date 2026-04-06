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

**功能**：兑换码免登录兑换 — 通过 API Key (sk-xxx) 认证兑换，供 neko-api-key-tool 等外部工具使用；兑换成功时同时补到用户钱包和当前 token/key 额度。

**背景**：上游的兑换接口 `POST /api/user/topup` 需要用户登录 Session。此补丁新增 `POST /api/token/redeem`，使用 `TokenAuthReadOnly` 中间件，允许通过 Bearer sk-xxx 免登录兑换；并在兑换成功时同步增加当前 token 的额度，方便 API Key 二次分发场景维持账实一致。

**涉及文件（4 个）**：

### 1. `controller/user.go`

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

### 2. `model/redemption.go`

新增 token 场景兑换逻辑 `RedeemByToken`：

- 在同一事务内增加用户钱包 quota
- 同时增加当前 token 的 `remain_quota`
- 一并核销兑换码

这样不会出现“钱包加了但 key 没加”的中间不一致状态。

### 3. `router/api-router.go`

在 `usageRoute` 块结束后、`redemptionRoute` 块开始前，插入一行路由注册：

```go
// 通过 API Key (sk-xxx) 免登录兑换，供外部工具（如 neko-api-key-tool）使用
apiRouter.POST("/token/redeem", middleware.CORS(), middleware.CriticalRateLimit(), middleware.TokenAuthReadOnly(), controller.TokenRedeem)
```

**定位标志**：找到 `tokenUsageRoute.GET("/", controller.GetTokenUsage)` 所在块的结束括号 `}`，在其后插入上述路由。

### 4. `controller/user_token_redeem_test.go`

补充控制器回归测试，重点验证：

- `Bearer sk-xxx` 可直接进入兑换链路
- 成功后用户钱包增加
- 成功后当前 token 的 `remain_quota` 同步增加

### 回归验证

建议最小验证命令：

```bash
go test ./controller -run '^TestTokenRedeem' -v
```

验证重点：

- `Bearer sk-xxx` 可直接兑换
- 成功后用户钱包增加
- 成功后当前 token 额度同步增加

---

## 002-task-refund-restore-token-quota.patch

**功能**：异步视频任务失败退款时，按环境变量开关恢复 token/key 额度。

**背景**：对于做 API Key 二次分发的用户，仅退款到钱包不够，失败任务还需要按开关决定是否恢复 token 可用额度。当前实现引入环境变量 `TASK_REFUND_RESTORE_TOKEN_QUOTA`，默认关闭；开启后，失败退款会同时恢复 token 额度。

**涉及文件（4 个）**：

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

---

## 补丁维护规范

1. **文件命名**：`NNN-简短描述.patch`，按序号排列
2. **每个补丁**：在本文件中记录功能说明 + 涉及文件 + 手动恢复步骤
3. **更新上游后**：先 `git apply`，失败则按文档手动改，最后 `git diff > patches/NNN-xxx.patch` 更新补丁文件
