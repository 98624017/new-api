# 二开总览

本仓库基于上游 `new-api` 维护，优先保持主线贴近上游。  
所有本地二开必须同时具备以下三项：

- 文档：`docs/customizations/NNN-*.md`
- 补丁：`patches/NNN-*.patch`
- 测试：最小回归验证命令或自动化测试

## 维护原则

- 二开优先做成独立补丁，不直接把长期差异散落在工作区
- 编号是永久 ID，`docs/customizations` 与 `patches` 必须一一对应
- patch 只承载代码差异，不承载完整业务背景
- 业务背景、规则、风险、测试方式统一记录在 `docs/customizations`
- 新增二开时，必须同步更新本文件

## 当前二开清单

### 001-token-redeem-via-apikey

- 目标：允许通过 API Key (`Bearer sk-xxx`) 直接兑换兑换码
- 影响范围：兑换接口、Token 认证链路
- 关联文件：
  - `controller/user.go`
  - `router/api-router.go`
  - `patches/001-token-redeem-via-apikey.patch`

### 002-task-refund-restore-token-quota

- 目标：异步视频任务失败退款时，按环境变量开关恢复 token/key 额度
- 影响范围：异步任务计费、任务轮询、token quota 调整
- 当前状态：已实现，并已生成 `patches/002-task-refund-restore-token-quota.patch`

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
go test ./controller -run '^TestTokenRedeem' -v
go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded)' -v
go build ./...
```
