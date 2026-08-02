# 011-classic-channel-status-and-banner

适配上游基线：`7c28993f6bd9e92616f3f578212577f8b7c40b45`。

## 1. 背景

后端已把渠道状态切换从通用渠道编辑接口拆到 `POST /api/channel/:id/status`，并明确拒绝通过 `PUT /api/channel/` 修改 `status`。Default 前端已迁移，Classic 前端仍向旧接口发送 `{ id, status }`，导致启用和禁用均返回“无效的参数”。

上游 Issue #5891、#5918 和关闭的 PR #5919 记录了相同问题与专用接口方案。该 PR 未合并是因为上游准备移除 Classic 前端，而不是修复方向有误。本仓库仍保留 Classic，因此需要作为本地二开维护。

Classic 全局布局还会挂载旧版前端停止维护横幅。本地继续维护 Classic 后，该提示不再符合部署预期。

## 2. 目标

- Classic 渠道启用和禁用使用后端专用状态接口
- 操作成功后按当前分页、搜索、类型和状态筛选重新加载服务端数据
- 状态切换使当前页越界时，自动回退并加载最后一个有效页
- 不再原地修改表格行或调用方持有的渠道对象
- Classic 全局布局不再挂载停止维护横幅
- 保留横幅组件、样式和翻译资源，降低补丁范围并方便恢复

## 3. 行为规则

### 渠道状态

- 启用：`POST /api/channel/{id}/status`，请求体 `{"status": 1}`
- 手动禁用：`POST /api/channel/{id}/status`，请求体 `{"status": 2}`
- 后端返回业务成功后显示成功提示并调用现有 `refresh()`
- 刷新后若请求页超过新总数对应的末页，普通列表和搜索列表都切换到末页并重新请求；空结果直接回到第 1 页
- 后端返回业务失败时显示原错误消息，不更新列表状态
- 删除、优先级、权重和标签操作继续使用原有接口

### 维护横幅

- `PageLayout` 不再 import 或渲染 `ClassicFrontendDeprecationBanner`
- 不使用 CSS `display: none`，因此组件不会挂载，也不会执行其本地存储逻辑
- 不删除组件、专属 CSS 或翻译键；恢复时重新挂载组件即可

## 4. 影响文件

- `web/classic/src/services/channel.js`
- `web/classic/src/services/channel.test.js`
- `web/classic/src/hooks/channels/channelPagination.js`
- `web/classic/src/hooks/channels/channelPagination.test.js`
- `web/classic/src/hooks/channels/useChannelsData.jsx`
- `web/classic/src/components/layout/PageLayout.jsx`
- `scripts/verify_patches.sh`

## 5. 风险与兼容性

- 不修改后端 API、数据库或 Default 前端。
- 状态成功后会增加一次列表查询，以服务端结果统一处理状态筛选和标签聚合。
- 仅当服务端总数表明当前页越界且仍有结果时，才额外请求最后有效页；总数为零时不会重复请求。
- 横幅资源成为暂未使用代码，这是为减少多语言文件和样式改动而保留的可恢复实现。
- 若后续同步上游并继续保留 Classic，需要确认专用状态接口路径及状态值仍未变化。

## 6. 回归验证

```bash
bun test web/classic/src/hooks/channels/channelPagination.test.js web/classic/src/services/channel.test.js
bun run --cwd web/classic build
make verify-patches
```

定向测试覆盖启用和禁用的请求合同，以及总数缩减和空结果时的页码归一化。`make verify-patches` 会从项目锁定的上游基线重放全部补丁，执行这些测试并构建 Classic 前端。
