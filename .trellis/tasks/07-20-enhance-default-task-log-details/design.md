# 技术设计：增强新版任务日志上游详情

## 边界与数据流

```text
tasks 表 data/properties/fail_reason
  -> TaskModel2Dto
  -> GET /api/task 或 /api/task/self
  -> TaskLog 前端契约
  -> 任务日志详情按钮
  -> TaskDetailsDialog
```

后端已经按现有权限返回任务 DTO，本次不增加接口、不查询额外数据，也不接触 `private_data`。前端将 `TaskLog.data` 定义为 `unknown`，在单一格式化边界处理对象、数组、标量、JSON 字符串、普通字符串和空值，避免各组件自行断言上游格式。

## 组件设计

### 入口

- 将当前只在有 `fail_reason` 时出现内容的“详情”列改为每行固定的“查看详情”按钮。
- 保留 `fail_reason` 作为列 accessor，避免移动端现有 `cells.get('fail_reason')` 布局失效。
- 按钮使用现有 `Button` 与图标体系，具备可访问名称，阻止事件冒泡，并在桌面/移动视图中复用同一 cell。

### 弹窗

新增 `TaskDetailsDialog`，复用现有受控 `Dialog`、`StatusBadge`、`CopyButton` 和音频预览能力，采用安静、紧凑的运维工具布局：

- 概览：任务 ID、平台、动作、状态、进度、模型。
- 时间：创建、更新、提交、开始、结束、耗时。
- 归属与计费：分组、额度、管理员可见的渠道和用户信息。
- 失败原因：完整换行显示并可复制。
- 结果：结果 URL、视频链接或 Suno 音频预览。
- 上游响应：完整展示 `data` 的格式化原始文本并可一键复制。
- 原始任务数据：完整 DTO JSON，默认折叠并可复制，用于与 Classic 的完整记录能力对齐。

大文本区域使用固定最大高度和双向滚动，不截断复制源。弹窗在移动端接近全屏，在桌面端限制宽度与视口高度；Base UI 负责焦点圈定、Escape 关闭和返回焦点。

## 数据规范化

在 usage-logs feature 的 `lib/` 中定义稳定的任务详情格式化函数：

- `formatTaskPayload(value: unknown): string`：JSON 字符串先解析并美化；对象、数组和标量安全序列化；普通字符串原样返回；空值返回空字符串。
- `getTaskAudioClips(data: unknown)`：只投影包含 `audio_url` 的对象数组，供现有音频组件消费。
- `getTaskFailureReason(status, failReason, resultUrl)`：仅将真实失败信息投影到 UI；旧版成功任务中与 `result_url` 相同的 `fail_reason` 视为结果 URL 并隐藏。

格式化函数是 API `unknown` 到 UI 文本的唯一边界，并用确定性单元测试保护不同输入形态。

## 类型与权限

- 补齐 `TaskLog` 与后端 `TaskDto` 已有字段：`start_time`、`group`、`quota`、`properties`、`result_url` 等。
- `data` 使用 `unknown`，不假定 Axios 解码后一定是字符串。
- 不新增基于前端的权限判断；普通用户只能从 `/api/task/self` 获取自己的任务，管理员从 `/api/task` 获取现有管理范围的数据。
- 管理员专属的渠道、用户信息沿用 `isAdmin` 条件展示。

## i18n

新增文案仅来自 `t(...)` 字面量。通过临时 `scripts/add-missing-keys.mjs` 写入 en、zh、zh-TW、fr、ja、ru、vi，之后运行 `bun run i18n:sync` 并删除临时脚本。优先复用已经存在的 `Upstream Response`、`Copy to clipboard`、`Raw Data` 等键。

## 性能与兼容性

- 不引入 JSON 编辑器或语法高亮依赖；原生 `<pre>` 足以满足排障，避免增加日志页首屏 bundle。
- 仅在弹窗打开的行组件内执行完整 DTO 序列化；简单布尔值不使用 `useMemo`。
- 不修改 API、数据库或 Classic 前端，无数据迁移和部署顺序要求。
- 回滚只需移除详情组件并恢复原详情 cell；后端数据不受影响。

## 本地二开维护

新增 `010-default-task-log-details`：

- `docs/customizations/010-default-task-log-details.md`
- `patches/010-default-task-log-details.patch`
- 更新两个 README 的登记和验证说明。

补丁以锁定上游 `7c28993f6bd9e92616f3f578212577f8b7c40b45` 为基线，只包含该二开的源码、测试与翻译差异。
