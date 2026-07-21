# 实施计划：增强新版任务日志上游详情

## 1. 前端契约与格式化

- [x] 补齐 `TaskLog` 对后端 `TaskDto` 的字段定义，并将 `data` 收紧为 `unknown` 输入边界。
- [x] 新增任务 payload 格式化与音频投影逻辑。
- [x] 添加确定性单元测试，覆盖对象、数组、JSON 字符串、普通字符串、标量、空值及不可序列化兜底。

## 2. 详情交互

- [x] 新增 `TaskDetailsDialog`，复用现有 Dialog、Button、StatusBadge、CopyButton 和音频预览组件。
- [x] 展示概览、时间、归属/计费、失败原因、结果、完整上游响应和原始任务 JSON。
- [x] 将任务日志详情列改为所有记录均可点击的详情入口，并确保移动卡片复用该入口。
- [x] 验证长文本、空响应、失败任务、视频任务和 Suno 音频任务的布局与交互。
- [x] 修复旧版成功视频任务将结果 URL 显示为失败原因的问题，并统一弹窗与详情入口的失败态判定。

## 3. 国际化

- [x] 运行 `bun run i18n:sync` 获取基线报告。
- [x] 仅通过 `scripts/add-missing-keys.mjs` 写入所有 locale 的新增或恢复文案。
- [x] 再次运行缺键检查与 `bun run i18n:sync`，删除临时脚本。

## 4. 本地二开资料

- [x] 新增并登记 `docs/customizations/010-default-task-log-details.md`。
- [x] 从锁定上游基线生成 `patches/010-default-task-log-details.patch`，确保只包含本二开差异。
- [x] 更新 `docs/customizations/README.md` 与 `patches/README.md`。

## 5. 验证与审查

- [x] 运行任务详情定向单元测试。
- [x] 对涉及文件执行格式检查和 lint。
- [x] 运行 `bun run typecheck` 与 `bun run build`。
- [x] 启动本地前端，通过浏览器检查桌面与移动视口、键盘操作、复制、长 JSON 滚动和空状态。
- [x] 运行 `graphify update .`。
- [x] 运行 `make verify-patches`，确认补丁链可重放及项目强制门禁通过。

验证说明：已将 `.gitignore` 中遗漏的本地 API 参考文件、Nginx 排障文档和 `.gstack/` 规则同步到其所属的 `005-project-maintenance-workflow.patch`。旧版成功视频任务回归修复完成后，仓库根目录直接执行 `make verify-patches` 通过：001-010 重放一致、11 个前端测试、双前端构建、Go 全量编译及全部定向回归均成功。

## 风险与回滚点

- `TaskLog.data` 的实际 JSON 形态不固定：由共享格式化边界和单元测试兜底。
- 大响应可能影响渲染：不做语法高亮，不在表格 cell 内格式化，仅在打开详情时处理。
- 移动端仍按 `fail_reason` cell ID 取详情：保留 accessor，避免额外改动移动卡片结构。
- `make verify-patches` 可能耗时且会执行干净构建；必须在 patch 生成后运行，失败时区分本改动、既有问题与环境限制。
