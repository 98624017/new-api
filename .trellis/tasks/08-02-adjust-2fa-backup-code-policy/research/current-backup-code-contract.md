# 当前备用码合同调研

## 后端生成与存储

- `common/totp.go` 将 `BackupCodeCount` 固定为 4，`GenerateBackupCodes` 循环生成 `XXXX-XXXX` 格式的 8 位字母数字码。
- `model/twofa.go::CreateBackupCodes` 在事务内删除该用户旧记录，然后逐个 bcrypt 哈希并写入 `TwoFABackupCode`。
- `TwoFABackupCode` 使用“一行一码”结构，消费状态由 `is_used` 和 `used_at` 表示，不包含使用次数。
- `GetUnusedBackupCodeCount` 统计 `is_used=false` 的记录，作为状态接口的 `backup_codes_remaining`。

## 消费路径

- 登录验证：`controller/twofa.go::Verify2FALogin`。
- 禁用 2FA：`controller/twofa.go::Disable2FA`。
- 通用高危操作验证：`controller/secure_verification.go::UniversalVerify`。
- 查看渠道密钥等管理操作：`controller/channel.go::validateTwoFactorAuth`。
- 所有路径最终调用 `TwoFA.ValidateBackupCodeAndUpdateUsage`，保持统一的一次性消费语义。

## 兼容性与性能

- 上游最新 `upstream/main` 仍固定生成 4 个码，没有可直接同步的数量调整。
- 20 个一次性码不改变数据库结构或 API 数据类型；现有记录不会自动变化。
- 服务器不保存备用码明文，因此不能给现有用户静默补发；只能在下一次主动重新生成时返回 20 个新码。
- 当前验证会遍历所有未使用记录并逐个执行 bcrypt 比对。数量增至 20 后，生成和最坏验证计算量约为当前的 5 倍；用户已接受本次不改查找机制。
- 当前消费不是条件原子更新，并发请求存在同一码重复成功窗口；用户认为该边界影响较小，本次明确不处理。

## 前端

- Default 的设置与重新生成弹窗均使用固定双列列表，重新生成弹窗宽度较窄且内容高度为自动。
- Classic 已使用移动单列、桌面双列，但代码区域没有独立高度限制。
- 两套前端都已有“复制全部”能力，无需新增接口或文案；20 个码需要受限高度和内部滚动。

## 二开维护

- 上一编号为 `011-classic-channel-status-and-banner`。
- 本改动属于新的本地运行时定制，需要新增 `012` customization 文档与 patch，登记到两个 README，并纳入 `scripts/verify_patches.sh`。
