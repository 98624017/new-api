# 实施计划

## 1. 渠道状态合同

- [x] 新增 classic 渠道 API 服务函数，封装 `POST /api/channel/:id/status`。
- [x] 先增加启用/禁用请求合同测试并确认旧实现不能满足该合同。
- [x] 修改 `useChannelsData.manageChannel` 使用专用状态函数。
- [x] 成功后调用现有 `refresh()`，删除对 `record.status` 的原地修改。
- [x] 增加分页回归测试，并让普通列表与搜索列表在当前页越界时重载最后有效页。
- [x] 保持删除、优先级、权重、标签与多密钥操作的现有 API 合同不变。

## 2. 维护横幅

- [x] 删除 `PageLayout` 中的横幅 import 与挂载点。
- [x] 保留独立横幅组件、专属 CSS 和翻译资源，确认没有通过 CSS 隐藏仍挂载的组件。

## 3. 二开登记

- [x] 新增 `docs/customizations/011-classic-channel-status-and-banner.md`。
- [x] 更新 `docs/customizations/README.md` 与 `patches/README.md`。
- [x] 更新 `scripts/verify_patches.sh`，将 classic 状态 API 测试纳入重放门禁。
- [x] 将 classic 分页回归测试纳入重放门禁。
- [x] 从当前集成树相对 `HEAD` 的运行时代码差异生成 `patches/011-classic-channel-status-and-banner.patch`。

## 4. 验证

- [x] 运行 classic 状态 API 定向测试。
- [x] 运行 classic 生产构建。
- [x] 运行 `graphify update .` 更新知识图谱（2,066 个文件的 AST 扫描完成；当前 Graphify 版本因 11,287 节点强制生成 HTML 而在可视化阶段失败，未产生图文件差异）。
- [x] 运行 `timeout 120s make verify-patches`（冷缓存两次超时；预热同一 Go 缓存后完整通过）。
- [x] 检查 `git diff --check` 与最终差异，确认没有调试代码、无关格式化或未登记源码改动。

## 风险与回滚点

- 横幅资源会成为暂未使用代码；这是经用户确认的最小改动取舍，后续需要恢复时只需重新挂载。
- `refresh()` 依赖当前搜索与筛选状态：复用既有入口，不新建并行查询路径。
- patch 必须在 001-010 后可应用并逐字复现当前集成树；失败时先修正 011 patch，不改动既有补丁。
