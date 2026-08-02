# 实施计划

## 1. 后端生成合同

- [x] 先添加失败测试，断言生成数量为 20、格式有效且无重复。
- [x] 将 `BackupCodeCount` 调整为 20。
- [x] 在生成函数中保证单批代码唯一，保持长度、格式、字符集和错误传播不变。
- [x] 确认初始化与重新生成继续复用同一生成函数，无需修改控制器合同。

## 2. 双前端展示

- [x] 调整 Default 设置弹窗的备用码网格为移动单列、桌面双列，并限制列表高度、启用内部滚动。
- [x] 对 Default 重新生成弹窗应用相同布局，不移动警告或复制按钮。
- [x] 调整 Classic 共用备用码展示网格的最大高度和滚动，保留现有响应式列数、编号与复制行为。
- [x] 确认没有新增用户文案，不修改 locale 文件。

## 3. 二开登记

- [x] 新增 `docs/customizations/012-2fa-twenty-backup-codes.md`。
- [x] 更新 `docs/customizations/README.md` 与 `patches/README.md`。
- [x] 更新 `scripts/verify_patches.sh`，把生成合同测试纳入不重复执行 `common` 包的现有测试组。
- [x] 生成 `patches/012-2fa-twenty-backup-codes.patch`，确保重放结果逐字匹配集成树。

## 4. 最小充分验证

- [x] 运行备用码生成定向 Go 测试。
- [x] 运行受影响 Go 包测试。
- [x] 运行 Default type-check，以及两套前端受影响文件的格式与 lint。
- [x] 构建 Default 和 Classic 前端。
- [x] 运行 `git diff --check`。
- [x] 运行 `graphify update .`；若仍受 HTML 节点规模限制，记录 AST 扫描结果与未产生图差异的事实。
- [x] 在 120 秒外层限制下运行完整 `make verify-patches`。

## 5. 复核与回滚点

- [x] 复核现有用户无迁移、API 无变化、单码仍一次性、旧码只在主动重新生成时失效。
- [x] 复核 20 项列表不会使移动端或桌面端弹窗内容溢出视口。
- [x] 生成/验证成本保持在用户已接受的低频 bcrypt 设计边界内，不引入新哈希格式。
- [x] `012` 补丁重放通过，无需修改既有 `001-011`。

## 验证记录

- `go test ./common -run '^TestGenerateBackupCodes$' -count=1` 通过。
- `go test ./common -count=1` 通过（`62` 项）。
- Default 完成 `typecheck`、受影响 TSX 的 oxfmt/oxlint；Classic 完成受影响 JSX 的 Prettier/ESLint。
- Default 与 Classic 生产构建通过。
- `git diff --check` 通过。
- `graphify update .` 已完成 `2067/2067` 文件的 AST 扫描；HTML 可视化因 `11289` 个节点超出工具限制而未能重建。
- `timeout 120s make verify-patches` 通过：001-012 依次重放、重放树一致性检查、双前端构建、Go 编译和定向回归均成功。
