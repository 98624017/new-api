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

默认执行：

```bash
make verify-patches
```

该命令会在锁定上游提交 `7c28993f6bd9e92616f3f578212577f8b7c40b45`
创建临时 worktree，按编号应用全部补丁，然后验证：

- 补丁重放树与当前集成树中的 patch 所属文件完全一致
- Go 全量编译
- `001` 到 `009` 的后端定向回归
- 双前端锁屏共享逻辑测试
- default 与 classic 两套前端构建

每个编译或测试子命令都由 `timeout 120s` 限制。同步到新的上游提交时，
必须先重建补丁并更新锁定基线，不能把“旧补丁碰巧可应用”当作同步完成。

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

- 本地同步要求工作区完全干净；Trellis 任务或其他未提交文件也必须先隔离
- 这个技能假设远端命名仍为 `origin` / `upstream`
- 如果后续新增 patch，记得同步更新脚本验证清单和本技能内容
- 修改既有二开时，先更新对应 `patches/NNN-*.patch`，再合并
