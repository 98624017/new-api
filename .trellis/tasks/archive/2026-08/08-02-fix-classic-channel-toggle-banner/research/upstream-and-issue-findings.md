# 上游与 Issue 调研

## 本地合同证据

- `controller/channel.go` 的 `UpdateChannel` 会在请求体出现 `status` 时返回无效参数。
- 专用状态接口为 `POST /api/channel/:id/status`，只接受可管理状态 1（启用）和 2（手动禁用）。
- `controller/channel_authz_test.go` 的 `TestUpdateChannelRejectsStatusField` 锁定了旧接口拒绝行为。
- 2026-08-02 运行 `go test ./controller -run '^(TestUpdateChannelRejectsStatusField|TestChannelStatusValidation)$' -count=1`，两项通过。
- classic `web/classic/src/hooks/channels/useChannelsData.jsx` 仍在启用/禁用分支调用 `PUT /api/channel/` 并发送 `status`。
- default `web/default/src/features/channels/api.ts` 已调用专用状态接口。

## 上游状态

- 2026-08-02 刷新 `upstream` 后，`upstream/main` 位于 `0ab020206`。
- 最新上游已删除整个 `web/classic/`，没有可直接同步的正式 classic 修复。
- 引入专用渠道状态接口的提交为 `4aee5f7d5`（`feat: better admin permissions (#5755)`）。该提交同步迁移了 default API，但没有迁移 classic 调用方。

## Issue 与候选修复

- Issue #5891：`https://github.com/QuantumNous/new-api/issues/5891`
  - 记录 classic 点击渠道启用/禁用返回“无效的参数”。
- Issue #5918：`https://github.com/QuantumNous/new-api/issues/5918`
  - 明确定位为 classic 仍调用旧 `PUT /api/channel/`，而后端要求 `POST /api/channel/:id/status`。
- PR #5919：`https://github.com/QuantumNous/new-api/pull/5919`
  - 将 classic 启停切换到专用状态接口，并使用目标状态更新当前行。
  - PR 未合并；维护者给出的关闭原因是 classic 即将移除，而不是实现不正确。

## 采用结论

- 采用 PR #5919 的接口方向。
- 不沿用其直接赋值 `record.status = ...` 的状态更新方式；本地实现成功后重新加载当前查询，避免状态筛选、搜索和标签聚合保留陈旧数据。
- 增加可直接验证专用 URL、方法和载荷的 classic API 单元测试，防止后续再次回退到通用更新接口。
