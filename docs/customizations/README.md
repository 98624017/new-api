# 二开总览

本仓库基于上游 `new-api` 维护，优先保持主线贴近上游。
所有本地二开必须同时具备以下三项：

- 文档：`docs/customizations/NNN-*.md`
- 补丁：`patches/NNN-*.patch`
- 测试：最小回归验证命令或自动化测试

## 维护原则

- 二开优先做成独立补丁，不直接把长期差异散落在工作区
- 二开代码改动必须先同步更新对应 patch，再合并
- 编号是永久 ID，`docs/customizations` 与 `patches` 必须一一对应
- patch 只承载代码差异，不承载完整业务背景
- 业务背景、规则、风险、测试方式统一记录在 `docs/customizations`
- 新增二开时，必须同步更新本文件

## 当前二开清单

### 001-api-key-self-service

- 目标：允许通过 API Key (`Bearer sk-xxx`) 兑换兑换码，并免登录查询当前 key 创建的异步任务列表
- 影响范围：兑换接口、Token 认证链路、充值使用记录、任务列表接口、任务落库字段、新任务 token 维度查询
- 当前状态：由原 `001-token-redeem-via-apikey` 和 `005-task-list-via-apikey` 合并，已生成 `patches/001-api-key-self-service.patch`

### 002-task-refund-restore-token-quota

- 目标：异步视频任务失败退款时，按环境变量开关恢复 token/key 额度
- 影响范围：异步任务计费、任务轮询、token quota 调整
- 当前状态：已实现，并已生成 `patches/002-task-refund-restore-token-quota.patch`

### 003-mask-billing-amounts-in-errors

- 目标：面向下游客户端的错误响应中保留额度错误语义，但脱敏具体金额 / 额度数值
- 影响范围：OpenAI / Claude 风格错误响应、异步任务错误响应、客户端错误脱敏
- 当前状态：已实现，并已生成 `patches/003-mask-billing-amounts-in-errors.patch`

### 004-sora-reference-video-double-price

- 目标：Sora 兼容 `/v1/videos` 请求中，环境变量白名单模型携带参考视频时按“生成时长 + 参考视频总时长”计价
- 影响范围：异步视频任务请求解析、参考视频时长检测、Sora 任务计费估算、任务 `OtherRatios`
- 当前状态：已实现，并已生成 `patches/004-sora-reference-video-double-price.patch`

### 005-project-maintenance-workflow

- 目标：保留本地上游同步、补丁校验、构建说明和 multipart 回归修复等项目维护差异
- 影响范围：CI workflow、README、AGENTS、makefile、同步脚本、补丁校验脚本、relay multipart 工具函数
- 当前状态：已实现，并已生成 `patches/005-project-maintenance-workflow.patch`

### 006-frontend-lock

- 目标：通过 `FRONTEND_LOCK_PASSWORD` 为前端页面增加弱隐藏锁屏，输入密码后进入原页面
- 影响范围：Go 启动时 HTML 注入、React 根渲染、锁屏公告展示
- 当前状态：已实现，并已生成 `patches/006-frontend-lock.patch`

### 007-seedance-reference-video-double-price

- 目标：通过 Sora/OpenAI 视频任务路径接入的 Seedance 模型，环境变量白名单模型携带参考视频时按双倍计费
- 影响范围：Sora 视频任务计费估算、Seedance 顶层参考视频字段检测、任务 `OtherRatios`
- 当前状态：已实现，并已生成 `patches/007-seedance-reference-video-double-price.patch`

### 008-seedance-asset-library-videos

- 目标：复用 `/v1/videos` 异步任务链路，通过 `seedance-asset` 模型提交 Seedance 真人形象资产库任务并查询 `AssetId`
- 影响范围：Sora/OpenAI 视频任务校验、资产任务计费倍率、公开任务 ID 响应重写、任务请求结构
- 当前状态：已实现，并已生成 `patches/008-seedance-asset-library-videos.patch`

## 上游同步标准流程

1. 拉取并合并上游 `new-api`
2. 按编号顺序重放 `patches/*.patch`
3. 执行二开最小回归测试
4. 执行 `go build ./...`
5. 通过后再推送同步分支或创建 PR

## 二开新增流程

1. 在 `docs/customizations` 新增对应文档
2. 完成代码实现
3. 生成对应 patch
4. 补充最小回归测试
5. 更新本文件的二开清单

## 推荐验证命令

```bash
make verify-patches
go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask)' -count=1
go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded)' -v
go test ./common -run TestMaskBillingAmountsForClient -count=1
go test ./types -run 'TestNewAPIError(To|MaskSensitiveErrorWithStatusCode)' -count=1
go test ./relay/channel/task/sora ./relay/common -count=1
go test ./relay/common -run TestValidateBasicTaskRequest_MultipartWithMetadata -count=1
go test . -run TestInjectFrontendLockPassword -count=1
(cd web && bun run build)
go build ./...
```

默认 patch 校验基准是当前项目锁定的原版 new-api：`22e509c1efb2260e1537c78684f1a5e9f053b75a`（`v0.12.11-1-g22e509c1e`）。验证其他上游基准时显式设置 `PATCH_BASE_REF`。
