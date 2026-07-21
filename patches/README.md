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
- 所有 patch 是否能按编号顺序应用到当前项目锁定的原版 new-api 基准 `7c28993f6bd9e92616f3f578212577f8b7c40b45`
- patch 所属文件在重放树中是否与当前集成树逐字一致
- 重放树是否能完成前端干净安装、共享锁屏测试、default/classic 构建、Go 编译和 9 组定向回归
- 当前工作区如果有源码改动，是否同步修改了至少一个 `patches/*.patch`

### 自动应用（推荐先尝试）

```bash
# 在当前项目锁定的原版 new-api 基准上，逐个应用补丁
git apply patches/001-api-key-self-service.patch

# 如有空白差异，加 --ignore-whitespace
git apply --ignore-whitespace patches/001-api-key-self-service.patch

# 如有轻微上下文偏移，用 3way merge
git apply --3way patches/001-api-key-self-service.patch
```

默认校验基准是 `7c28993f6bd9e92616f3f578212577f8b7c40b45`。如果需要验证新的上游基准，可显式设置：

```bash
PATCH_BASE_REF=upstream/main make verify-patches
```

覆盖基准只用于验证已经针对该提交重建的补丁，不能用来跳过补丁重建。

### 冲突时手动恢复

如果 `git apply` 失败，根据下面每个补丁的说明手动修改即可。

---

## 001-api-key-self-service.patch

**功能**：API Key 自助能力。允许通过 API Key (`Bearer sk-xxx`) 免登录兑换兑换码，并查询当前 key 创建的异步任务列表。

**背景**：上游的兑换接口 `POST /api/user/topup` 需要用户登录 Session，异步任务列表也依赖登录态。本补丁面向 API Key 二次分发、外部控制台和轮询工具，集中提供只持有 API Key 时需要的自助能力。

**涉及文件（9 个）**：

### 1. `controller/token_test.go`

在 token 控制器测试的临时 DB helper 中保存并恢复 `model.DB` / `model.LOG_DB`，避免外部包控制器测试互相污染全局 DB。

### 2. `controller/user.go`

新增 `TokenRedeem`：

- 使用 `TokenAuthReadOnly()` 上下文中的用户和 token 信息
- 复用原有 `topUpRequest`
- 调用 `model.RedeemByToken`
- 成功后返回兑换额度

### 3. `model/redemption.go`

新增 token 场景兑换逻辑 `RedeemByToken`：

- 在同一事务内增加用户钱包 quota
- 同时增加当前 token 的 `remain_quota`
- 读取当前 token 名称，并写入充值使用记录
- 一并核销兑换码

### 4. `router/api-router.go`

新增两条 API Key 认证路由：

```go
apiRouter.POST("/token/redeem", middleware.CORS(), middleware.CriticalRateLimit(), middleware.TokenAuthReadOnly(), controller.TokenRedeem)
taskRoute.GET("/token/self", middleware.TokenAuthReadOnly(), controller.GetUserTokenTask)
```

### 5. `controller/task.go`

新增 `GetUserTokenTask`：

- 从 `TokenAuthReadOnly()` 上下文中读取 `id` 和 `token_id`
- 复用现有分页参数和任务筛选参数
- 返回结构与 `/api/task/self` 保持一致

### 6. `model/task.go`

任务表新增独立 `token_id` 列，并新增按 token 维度查任务的方法：

- `TaskGetAllUserTokenTask`
- `TaskCountAllUserTokenTask`
- 列表与总数查询直接走独立 `token_id` 列
- 不为历史任务做懒回填，补丁上线前未写入该列的任务默认查不到

### 7. `controller/relay.go`

异步任务提交成功后，除了继续写 `private_data.token_id`，还同步写入任务表独立 `token_id` 列，保证新任务后续能走索引列查询。

### 8. `controller/user_token_redeem_test.go`

补充 API Key 兑换回归测试：

- `Bearer sk-xxx` 可直接进入兑换链路
- 成功后用户钱包增加
- 成功后当前 token 的 `remain_quota` 同步增加
- 成功后的充值使用记录包含 token/key 名称

### 9. `controller/task_token_test.go`

补充 API Key 任务列表回归测试：

- `Bearer sk-xxx` 可直接访问新接口
- 仅返回当前 token 创建的任务
- `task_id` 过滤参数仍然生效

### 回归验证

```bash
go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask)' -count=1
```

---

## 002-task-refund-restore-token-quota.patch

**功能**：异步视频任务失败退款时，按环境变量开关恢复 token/key 额度。

**背景**：对于做 API Key 二次分发的用户，仅退款到钱包不够，失败任务还需要按开关决定是否恢复 token 可用额度。当前实现引入环境变量 `TASK_REFUND_RESTORE_TOKEN_QUOTA`，默认关闭；开启后，失败退款会同时恢复 token 额度。

**涉及文件（7 个）**：

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

补充开关关闭、开关开启和 `UpdateVideoTasks` 失败路径测试。

### 5. `scripts/seed_task_refund_fixture.go`

离线生成 `002` 容器验收所需的 fixture。

### 6. `scripts/mock_video_failure_server.go`

宿主机上的本地 mock 上游服务，提供失败视频任务响应、健康检查和命中统计。

### 7. `scripts/verify_task_refund_restore_token_quota.sh`

黑盒验收脚本，覆盖 `TASK_REFUND_RESTORE_TOKEN_QUOTA=false` 和 `true` 两轮场景。

### 回归验证

```bash
go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded)' -v
bash scripts/verify_task_refund_restore_token_quota.sh new-api:verify-20260406
```

---

## 003-mask-billing-amounts-in-errors.patch

**功能**：下游客户端错误响应金额脱敏。

**背景**：预扣费失败、额度不足或部分上游错误文案可能带出具体金额 / 额度数值，例如 `用户剩余额度: ¥0.056700, 需要预扣费额度: ¥0.069900`。这些数值可能暴露本地成本价、预扣费策略或上游额度细节，不适合透传给下游客户。

**涉及文件（6 个）**：

### 1. `common/str.go`

新增 `MaskBillingAmountsForClient`，优先按计费语义标签脱敏，货币符号只作为无标签场景的保守兜底。

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

**功能**：Sora 兼容 `/v1/videos` 请求中，白名单模型携带参考视频时默认按旧双倍计价；开启环境变量后按“生成时长 + 参考视频总时长”精确计价。

**背景**：部分视频生成模型在请求体 `content` 中携带 `video_url` 参考视频时，上游成本不同于普通文生视频请求。为兼顾稳定性和新计费规则，本补丁默认保持旧的 `video_input: 2` 计费；设置 `SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED=true` 后，才会检测参考视频时长并按精确规则计费。

**涉及文件（6 个）**：

### 1. `relay/common/relay_info.go`

在 `TaskSubmitReq` 中新增 `Content []map[string]any`，用于保留 `/v1/videos` JSON 请求体顶层 `content` 数组。

### 2. `relay/channel/task/sora/constants.go`

新增环境变量白名单和精确计费开关加载逻辑。默认白名单为空；默认不开启精确参考视频时长计费。

### 3. `main.go`

在 `.env` 和 `common.InitEnv()` 之后调用 `soratask.ReloadReferenceVideoDoublePriceModelsFromEnv()`。

### 4. `relay/channel/task/sora/adaptor.go`

在校验阶段识别白名单模型和精确计费开关：默认只在 `EstimateBilling` 中返回 `video_input: 2`；开启精确计费后检测参考视频总时长，并把 `seconds` 设置为 `生成时长 + 参考视频总时长`。

### 5. `relay/channel/task/sora/video_duration.go`

新增参考视频时长检测：精确计费开启时，先用 HTTP Range 读取前 1MiB 尝试解析 MP4/MOV 元数据；失败后回退完整受限下载；仍无法解析或 30 秒内无法完成检测时拒绝请求。

### 6. `service/download.go`

为 Worker 请求增加 context 版本，确保参考视频检测超时时 Worker 模式也能及时返回本地错误。

### 回归验证

```bash
make verify-patches
```

Sora 适配器测试文件由后续共享该文件的 `007` 补丁首次创建，完整补丁链会统一验证 004/007/008/009。

---

## 005-project-maintenance-workflow.patch

**功能**：项目维护工作流。保留本地上游同步、补丁校验、构建说明和 multipart 回归修复等工程化差异。

**背景**：当前项目除了业务二开外，还需要保留一组用于同步上游、验证补丁和稳定构建的本地维护文件。该补丁用于确保“项目锁定的原版 new-api + patches” 能重放到当前现状。

**涉及文件（15 个）**：

### 1. `.github/workflows/docker-image-manual-ghcr.yml`

保留手动 GHCR Docker 构建流程。该流程不接受分支、标签或提交输入，只允许从当前仓库 `main` 构建；准备阶段固定完整提交 SHA，各架构复用同一 SHA，发布前运行 API Key 自助兑换回归，并串行更新版本标签与 `latest`，防止上游标签或并发任务误覆盖生产镜像。

### 2. `.github/workflows/sync-upstream.yml`

保留上游同步 workflow。运行时暂存 `patches/` 和 `docs/customizations/`，再从 `upstream/<branch>` 创建同步分支、应用补丁并恢复登记文件；随后安装 Go/Bun 并执行完整重放验证。

### 3. `.github/workflows/electron-build.yml`

移除与当前部署链路无关的 Electron 桌面应用构建 workflow，避免误触发非 Docker 构建。

目标上游的 `docker-build.yml` / `docker-image-branch.yml` 已覆盖 amd64、arm64 和分支镜像构建，因此不恢复旧的独立 alpha/arm64 workflow。

### 4. `.gitignore`

保留本地生成物忽略规则，包含 graphify 输出、`.tmp-newapi-verify`、Playwright CLI 与 gstack 目录、`meituapi/`、本地 API 参考文件和 Nginx 排障文档等验证/素材产物。

### 5. `AGENTS.md`

保留项目内 agent 工作约定。

### 6. `README.md` / `README.zh_CN.md`

保留本地维护说明入口。

### 7. `makefile`

保留 `verify-patches` 等本地维护命令。

### 8. `relay/common/relay_utils.go`

保留 multipart 请求体处理回归修复。

### 9. `relay/common/relay_utils_test.go`

覆盖 multipart 请求体回归测试。

### 10. `scripts/sync_upstream_local.sh`

本地上游同步脚本。

### 11. `scripts/verify_patches.sh`

校验补丁配对、顺序应用、重放树一致性，并在临时重放树中执行前端干净安装、双前端构建、Go 编译和 9 组回归。每个编译或测试子命令最多运行 120 秒。

### 12. `tools/skills/newapi-upstream-sync/SKILL.md`

本地上游同步 skill 说明。

### 13. `web/classic/package.json` / `web/bun.lock`

固定 classic 使用兼容的 `date-fns@2.30.0` 与 `date-fns-tz@1.3.8`，避免 Bun workspace 将旧版时区库错误解析到 default 使用的 `date-fns@4`。

### 回归验证

```bash
make verify-patches
```

---

## 006-frontend-lock.patch

**功能**：前端弱隐藏锁屏。设置 `FRONTEND_LOCK_PASSWORD` 后，浏览器访问前端页面需要先输入密码。

**背景**：内部服务域名有时会同时暴露前端入口。本补丁用于降低普通访客直接看到管理页面入口的概率，但不提供真正安全隔离。

**涉及文件（20 个）**：

### 1. `main.go`

新增双前端注入逻辑，在服务启动时读取 `FRONTEND_LOCK_PASSWORD` 并分别注入 default/classic 的 `window.__FRONTEND_LOCK_PASSWORD__`。

### 2. `main_test.go`

覆盖空密码跳过注入、正常密码注入，以及 `</script>` 等字符被 JSON 安全转义。

### 3. `web/shared/frontend-lock.ts` / `web/shared/frontend-lock.test.ts`

共享存储 key、30 天 TTL、密码指纹、校验逻辑和 localStorage 异常降级，并用 Bun 测试覆盖。

### 4. `web/default/**`

在 default 根渲染接入原生风格锁屏、公告加载、i18n 文案和开发环境密码注入。

### 5. `web/classic/**`

在 classic 根渲染接入 Semi UI 锁屏，并复用共享状态逻辑。

### 6. `Dockerfile`

在 Docker 前端构建阶段复制 `web/shared`，保证两套 Rsbuild 均能解析共享模块。

### 回归验证

```bash
go test . -run TestInjectFrontendLockPassword -count=1
bun test web/shared/frontend-lock.test.ts
bun run --cwd web/default build:check
bun run --cwd web/classic build
make verify-patches
```

---

## 007-seedance-reference-video-double-price.patch

**功能**：通过 Sora/OpenAI 视频任务路径接入的 Seedance 模型，白名单模型携带参考视频时按双倍计费。

**背景**：Seedance 在当前部署中复用 NewAPI 的 Sora/OpenAI 视频任务机制，上游兼容 `/v1/videos`。下游可通过 `files`、`input_video`、`video_url`、`reference_video` 等顶层字段携带参考视频，这类请求需要与普通文生视频区分计费。

**涉及文件（2 个）**：

### 1. `relay/channel/task/sora/adaptor.go`

在 `EstimateBilling` 中识别 OpenAI Videos 顶层参考视频字段，命中白名单模型后返回 `video_input: 2`；保留原有 `content[].video_url` 视频输入计费路径。

### 2. `relay/channel/task/sora/adaptor_test.go`

覆盖 Seedance 白名单模型顶层参考视频双倍计费、图片/音频不触发、未配置白名单不触发、Seedance 白名单加载，以及 Sora JSON 请求体字段透传。

### 回归验证

```bash
go test ./relay/channel/task/sora
make verify-patches
```

---

## 008-seedance-asset-library-videos.patch

**功能**：复用 `/v1/videos` 异步任务链路，通过 `seedance-asset` 模型提交 Seedance 真人形象资产库任务并查询 `AssetId`。

**背景**：Seedance2 上游提供真人形象 IP 资产库 API。当前部署希望继续使用 NewAPI 的 OpenAI Videos 任务机制、API Key 鉴权、任务入库、轮询、计费和用户隔离能力，不新增下游资产专用端点。

**涉及文件（8 个）**：

### 1. `relay/channel/task/sora/adaptor.go`

新增 `seedance-asset` 资产任务识别：

- 校验只要求模型名、资源名称/资产显示名和公网图片 URL
- 资产任务不执行参考视频时长检测
- 资产任务 `EstimateBilling` 返回空倍率，不叠加 `seconds`、`size` 或参考视频倍率
- `DoResponse` 基于原始 JSON 覆盖公开 `id/task_id`，避免丢失顶层 `asset_id` 和 metadata

### 2. `relay/common/relay_info.go`

`TaskSubmitReq` 新增 `Files []string`，用于读取 OpenAI Videos 风格 `files` 字段中的资产图片 URL。

### 3. `relay/channel/task/sora/adaptor_test.go`

补充资产任务回归测试：

- `seedance-asset` 不返回视频计费倍率
- 私网/回环 URL 被拒绝
- `files[0]` 可作为资产图片输入
- 替换公开任务 ID 时保留 `asset_id` 和 metadata

### 4. `controller/task.go`

新增 API Key 维度资产删除接口：

- 按当前 `user_id + token_id + task_id` 查找任务
- 校验任务为已成功且未删除的 `seedance-asset` 资产任务
- 把任务记录里的上游任务 ID 转发到渠道 `POST /api/task/token/asset/delete`
- 上游 NewAPI wrapper 返回 `success=false` 时视为失败，不标记本地任务已删除
- 删除成功后只在 task data 与 `metadata.seedance` 标记 `deleted/deleted_at`

### 5. `model/task.go`

新增 `GetByUserTokenTaskId`，按当前用户、当前 API Key 和任务 ID 查询单个任务。

### 6. `router/api-router.go`

新增 `POST /api/task/token/asset/delete`，使用 API Key 任务自助链路。

### 7. `controller/task_token_test.go`

补充资产删除 controller 回归测试：

- 当前 API Key 创建的成功资产可以删除
- 删除调用统一转发到上游 `POST /api/task/token/asset/delete`
- 上游 HTTP 200 但 `success=false` 时不会标记本地任务已删除
- 同用户其他 API Key 的资产不能删除
- 非资产、未成功、已删除的任务会拒绝

### 8. `controller/user_token_redeem_test.go`

测试清理逻辑同步清空任务和渠道表，避免 controller 测试之间污染全局测试数据库。

### 回归验证

```bash
go test ./relay/channel/task/sora ./relay/common ./relay ./controller
make verify-patches
```

---

## 009-sora-unknown-status-polling.patch

**功能**：Sora/OpenAI Videos 轮询兼容上游 `status: "unknown"`，并在未知状态已经返回明确结果 URL 时兜底为成功，同时同步对外查询和内容下载路径。

**背景**：当另一个 NewAPI 实例作为本项目的上游 Sora/OpenAI Videos 兼容渠道时，上游任务刚创建或还未完成状态归一时可能返回 `status: "unknown"`。旧版 Sora 适配器不识别该状态，后台轮询会把仍在执行的任务提前标记为 `upstream returned unrecognized message`。部分兼容上游还可能在状态仍为 `unknown` 时已经携带 `url` / `video_url` 等结果链接，需要避免任务一直停留在处理中，也要避免内部已成功但客户端查询仍看到 `unknown` 或内容下载继续请求上游 `/content`。

**涉及文件（4 个）**：

### 1. `relay/channel/task/sora/adaptor.go`

扩展 Sora 任务状态归一化：

- `unknown` -> `IN_PROGRESS`
- 缺失 `status` 但具备视频任务特征且不带明确错误 -> `IN_PROGRESS`
- 缺失 `status` 或 `unknown` 但带有明确 `error.message` / `error.code` -> `FAILURE`
- 缺失 `status` 或 `unknown` 且已经返回明确结果 URL -> `SUCCESS`
- `progress: 100` 但没有明确结果 URL -> 仍保持处理中，不单独视为成功
- 已保存成功结果 URL 的任务对外转换为 `completed`，并在 `metadata.url` 中返回结果 URL
- `submitted` / `not_start` -> `QUEUED`
- `running` -> `IN_PROGRESS`
- `succeeded` -> `SUCCESS`
- `canceled` -> `FAILURE`

空对象或真正不具备视频任务特征的响应仍保留空状态，由轮询层按无法识别响应处理。

### 2. `controller/video_proxy.go`

OpenAI/Sora content 代理优先使用已保存的非本地代理 `ResultURL`。当 `ResultURL` 为空或仍是本地 `/v1/videos/{task_id}/content` 代理 URL 时，保持原有上游 `/v1/videos/{upstream_id}/content` 请求行为。

### 3. `relay/channel/task/sora/adaptor_test.go`

补充状态兼容回归测试：

- OpenAI 视频响应 `status: "unknown"` 保持处理中
- 缺失 `status` 但包含视频任务字段时保持处理中
- `status: "unknown"` 但包含顶层 `video_url`、`metadata.url` 或 `output.video_url.url` 时进入成功并保留结果 URL
- `status: "unknown"` 且仅有 `progress: 100` 时仍保持处理中
- 缺失 `status` 但包含明确错误时进入失败
- 同时包含明确错误和结果 URL 时，错误优先，任务进入失败
- `{}` 仍保留空状态，不吞掉真正无法识别的响应
- 已保存成功结果 URL 的任务对外转换为 `completed`，并在 `metadata.url` 中返回结果 URL

### 4. `controller/video_proxy_test.go`

补充 content 下载回归测试：

- Sora 成功任务已保存直链时，`/v1/videos/{id}/content` 命中保存的结果 URL，不再请求上游 `/content`

### 回归验证

```bash
go test ./relay/channel/task/sora -count=1
go test ./controller -run 'TestVideoProxyUsesStoredResultURLForSoraTask' -count=1
make verify-patches
```

---

## 补丁维护规范

1. **文件命名**：`NNN-简短描述.patch`，按序号排列
2. **每个补丁**：在本文件中记录功能说明 + 涉及文件 + 手动恢复步骤
3. **更新上游后**：先 `git apply`，失败则按文档手动改，最后 `git diff > patches/NNN-xxx.patch` 更新补丁文件
