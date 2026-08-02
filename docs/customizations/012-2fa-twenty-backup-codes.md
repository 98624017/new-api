# 012-2fa-twenty-backup-codes

适配上游基线：`7c28993f6bd9e92616f3f578212577f8b7c40b45`。

## 1. 背景

上游每次只生成 4 个两步验证备用码，在多次紧急恢复场景中不足。本仓库选择在不改变单码安全语义的前提下，增加一次生成的数量。

## 2. 目标

- 每次初始化或主动重新生成固定创建 20 个互不重复的备用码
- 保持一码一用、bcrypt 哈希、API 返回结构和数据库表结构不变
- 使 Default 与 Classic 在移动端单列、桌面端双列的受限高度列表中完整查看并复制代码

## 3. 行为规则

- `common.BackupCodeCount` 是新初始化和重新生成数量的唯一来源，固定为 20。
- 生成函数继续返回 `XXXX-XXXX` 大写字母数字码，对单批次的随机碰撞重试，保证互不重复。
- 初始化和重新生成仍复用现有的生成、哈希、存储和一次性消费路径；`backup_codes` 和 `backup_codes_remaining` API 字段不变。
- 现有已启用 2FA 用户的备用码记录不变；只有其主动重新生成时才会用 20 个新码替换全部旧码。
- 两套前端保留现有提示和“复制全部”操作，仅代码网格在内部滚动，不增加下载、打印或新文案。

## 4. 影响文件

- `common/totp.go`
- `common/totp_test.go`
- `web/default/src/features/profile/components/dialogs/two-fa-backup-dialog.tsx`
- `web/default/src/features/profile/components/dialogs/two-fa-setup-dialog.tsx`
- `web/classic/src/components/settings/personal/components/TwoFASetting.jsx`
- `scripts/verify_patches.sh`

## 5. 风险与兼容性

- 备用码验证仍逐条执行 bcrypt 比对；20 个码的最坏路径比原来约增加 5 倍计算量，但该路径为低频安全操作。
- 不迁移、不批量重置旧用户的备用码，因此升级不会产生旧码失效或重发明文码的风险。
- 本次不改变现有“读取后标记”的并发消费窗口；同一备用码的并发请求仍可能重复成功。

## 6. 回归验证

```bash
go test ./common -run '^TestGenerateBackupCodes$' -count=1
go test ./common -count=1
bun run --cwd web/default typecheck
bun run --cwd web/default build
bun run --cwd web/classic build
make verify-patches
```
