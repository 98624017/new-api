---
name: newapi-upstream-sync
description: Use when working inside the new-api repository and the user wants to sync QuantumNous upstream, preserve the local custom patches, run the required regression checks, and then decide whether to push.
---

# newapi-upstream-sync

## Overview

这是 `new-api` 仓库专用的上游同步技能。

目标是把这类高频操作固定成稳定流程：

1. 拉取 `upstream/main`
2. 保留本仓库的二开内容
3. 运行必须的回归验证
4. 再决定是否提交、推送

## When To Use

仅在当前工作目录是 `new-api` 仓库时使用。

适用场景：

- 用户要求“拉取 upstream 最新代码”
- 用户要求“合并上游并保留二开 patch”
- 用户要求“同步 QuantumNous/new-api”
- 用户想快速完成“拉取 -> 合并 -> 验证 -> 推送”

不适用：

- 当前工作区还有未提交改动且用户没有明确要继续
- 任务是修 bug、做功能开发，而不是同步上游
- 当前仓库不是 `new-api`

## Default Flow

优先使用仓库内命令：

```bash
make sync-upstream-local
```

需要同步后直接推送时：

```bash
PUSH_AFTER_SYNC=1 make sync-upstream-local
```

如需跳过测试：

```bash
SKIP_TESTS=1 make sync-upstream-local
```

## What The Command Does

脚本位置：

```bash
scripts/sync_upstream_local.sh
```

固定执行这些动作：

1. 检查工作区是否干净
2. 拉取 `upstream` 最新代码
3. 创建备份分支 `backup_upstream_sync_时间戳`
4. 合并 `upstream/main`
5. 跑二开相关回归测试
6. 输出 `git status` 和最近提交
7. 只有在 `PUSH_AFTER_SYNC=1` 时才自动推送

## Required Verification

默认验证以下回归点：

```bash
make verify-patches
go test ./controller -run '^TestTokenRedeem' -v
go test ./service -run '^(TestRefundTaskQuota_Wallet|TestRefundTaskQuota_Wallet_RestoreTokenEnabled|TestUpdateVideoTasks_FailureRefund)$' -v
go test ./relay/common -run '^TestValidateBasicTaskRequest_MultipartWithMetadata$' -v
```

原因：

- 第 0 组验证二开 patch 登记完整，并能按顺序应用到 `upstream/main`
- 第 1 组验证 `001-token-redeem-via-apikey`
- 第 2 组验证 `002-task-refund-restore-token-quota`
- 第 3 组验证上游 multipart 任务请求回归的本地兼容修复

## Decision Rules

如果合并时出现冲突：

- 先看 `patches/README.md`
- 再看 `docs/customizations/`
- 优先保留本仓库已经确认过的二开行为
- 不要盲目覆盖 upstream 新逻辑，先判断是否是上游 bug

如果测试失败：

- 先确认是二开逻辑失效，还是上游新引入的问题
- 上游问题优先做最小修复，并加简体中文注释说明“临时兼容上游问题”
- 修复后重新跑对应回归测试

如果验证通过：

- 默认停在本地，等待用户决定是否推送
- 只有用户明确要推送，或设置了 `PUSH_AFTER_SYNC=1`，才执行 push

## Reference Files

同步时优先参考这些文件：

```text
patches/README.md
docs/customizations/README.md
.github/workflows/sync-upstream.yml
```

## Notes

- `.spec-workflow/` 默认允许保持未跟踪状态，不阻塞同步
- 这个技能假设远端命名仍为 `origin` / `upstream`
- 如果后续新增 patch，记得同步更新脚本验证清单和本技能内容
- 修改既有二开时，先更新对应 `patches/NNN-*.patch`，再合并
