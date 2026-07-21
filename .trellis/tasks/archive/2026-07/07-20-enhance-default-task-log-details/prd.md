# 增强新版任务日志上游详情

## Goal

在 `web/default` 的任务日志中恢复便于排查异步上游任务的详情查看能力，使管理员能够查看任务 API 已返回的上游响应 `data`、失败原因和任务元数据，而无需切换到 Classic 前端或打开浏览器开发者工具。

该能力作为新的本地二开 `010-default-task-log-details` 维护，避免后续同步上游时再次丢失。

## Confirmed Facts

- 后端 `TaskDto` 已返回 `data`、`properties`、`fail_reason`、`result_url` 等字段，本任务不需要修改数据库或 API 契约。
- Classic 前端支持悬浮查看截断的失败详情，并可点击任务 ID 查看完整任务记录。
- Default 前端当前只展示 `fail_reason`；除 Suno 音频预览外，没有展示 `data` 的通用入口。
- Default 前端曾在提交 `308e3e347` 中加入包含 `Upstream Response` 的任务详情弹窗，随后被整套 UI 回滚提交 `337169e0a` 一并删除。
- `private_data` 明确不通过 API 返回，因此前端详情不得尝试展示上游密钥或内部私有任务字段。

## Requirements

- 在 Default 前端的每条任务日志中提供明确、可键盘操作的详情入口；所有任务统一通过点击详情按钮打开弹窗，不使用大内容悬浮层。
- 详情至少展示任务标识、平台、动作、状态、时间、渠道/用户（仅管理员现有权限范围内）、失败原因、请求属性、结果地址和上游响应 `data`。
- 上游响应支持对象、数组、标量、JSON 字符串和非 JSON 字符串，不因异常数据导致页面崩溃。
- 长内容必须可滚动、可换行并支持复制，不能撑坏桌面或移动端布局。
- 继续保留现有音频、视频和失败原因查看能力。
- 所有新增 UI 文案使用 `t(...)`，并覆盖项目支持的全部 locale。
- 不新增运行时依赖，不扩大后端数据暴露范围，不修改 Classic 前端行为。
- 新增并登记 `docs/customizations/010-default-task-log-details.md` 与 `patches/010-default-task-log-details.patch`。

## Acceptance Criteria

- [x] 普通用户和管理员都能在各自已有权限范围内，从 Default 的每条任务日志打开完整详情。
- [x] 任务 `data` 有内容时显示“上游响应”，并能复制原始内容。
- [x] `data` 为空时不显示空的上游响应区块，其他任务详情仍可查看。
- [x] 失败任务完整展示并可复制 `fail_reason`。
- [x] 旧版成功视频任务将回填到 `fail_reason` 的结果 URL 视为结果，不显示为失败。
- [x] 对象、数组、JSON 字符串、普通字符串及超长内容均可稳定展示。
- [x] 详情交互支持键盘、焦点管理、Escape 关闭和可访问名称。
- [x] 桌面端与移动端无文本溢出或控件重叠。
- [x] 新增文案已同步到所有 locale，i18n 检查通过。
- [x] 相关类型检查、lint、定向测试和前端构建通过。
- [x] `010` 自定义文档与补丁已登记，`make verify-patches` 通过。

## Out of Scope

- 修改任务数据的采集、存储或脱敏逻辑。
- 暴露 `private_data` 或其他后端未返回字段。
- 修改 Classic 前端现有交互。
- 新增任务详情独立 API。

## Notes

- 这是带 i18n、响应式交互和本地补丁维护的复杂任务；规划完成前需补充 `design.md` 与 `implement.md`。
- 用户已确认所有任务统一采用点击详情弹窗查看完整上游响应，不保留大内容悬浮交互。
