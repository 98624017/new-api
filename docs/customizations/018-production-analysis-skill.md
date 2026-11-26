# 018 生产备份排查手册

项目通过 `.agents/skills/newapi-production-analysis/` 提供生产备份只读排查流程。入口 `SKILL.md` 引用 `references/r2-backups.md` 获取和校验快照，引用 `references/data-analysis.md` 解释额度、兑换和渠道事件。

同步上游时按编号重放 `patches/018-production-analysis-skill.patch`。若出现冲突，按补丁恢复上述三个文件及其相对链接；上游如有同名流程，按以下依据核对并合入本地规则。

- 对照备份服务的 `internal/backup/manifest.go`、`dump_filter.go`。声明排除 `public.logs.other.po` 时，报告列出该证据缺口。旧 manifest 缺少 `excluded_fields` 时，排除范围标为“未声明”。
- 对照 `middleware/auth.go` 的认证入口。普通 relay 令牌和附渠道 ID 的管理员令牌，都按各自的完整主体精确匹配 `tokens.key`。
- 对照 `controller/redemption.go` 和 `model/redemption.go`。以带兑换码 ID 的入账流水核实历史金额。快照面额可在兑换后修改，同名创建批次只有在能唯一关联时才可用于单码归因。

在仓库根目录执行 `timeout 120s make verify-patches`。门禁在锁定的上游基线上重放补丁，并逐字节比较手册的三个文件。
