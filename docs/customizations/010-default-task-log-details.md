# 010-default-task-log-details

适配上游基线：`7c28993f6bd9e92616f3f578212577f8b7c40b45`。

## 1. 背景

Default 前端的任务日志当前只在“详情”列展示 `fail_reason`。成功任务通常没有失败原因，因此即使后端 `/api/task` 和 `/api/task/self` 已返回上游任务响应 `data`，管理员和普通用户也无法在新版界面查看，只能切换 Classic 前端或打开浏览器开发者工具。

Classic 前端允许点击任务 ID 查看整条任务记录，但其大文本悬浮和通用内容弹窗不适合直接移植到移动端。该二开在 Default 前端提供统一任务详情弹窗，恢复排查能力并保持新版组件体系、响应式布局和可访问性。

## 2. 目标

- Default 前端每条任务记录均提供固定的“查看详情”入口
- 弹窗展示任务元数据、时间、计费、失败原因、结果、请求属性和完整上游响应
- 支持复制完整上游响应和原始任务 JSON
- 支持对象、数组、标量、JSON 字符串、普通字符串和空值
- 保留现有视频结果代理和 Suno 音频预览能力
- 普通用户沿用 `/api/task/self` 权限，只查看自己的任务；管理员沿用 `/api/task` 权限
- 不修改后端 API、数据库和 Classic 前端

不解决：

- 不采集后端当前没有保存的上游响应
- 不暴露 `private_data`、渠道密钥或上游内部任务 ID
- 不增加独立任务详情 API
- 不修改任务响应的脱敏与存储规则

## 3. 交互规则

- 桌面表格和移动任务卡片复用同一个“查看详情”按钮
- 所有任务都能打开详情，不再依赖是否存在 `fail_reason`
- 大文本只在弹窗中展示，不使用可能遮挡表格的悬浮层
- 上游响应有内容时展示“上游响应”区块；空响应不展示空区块
- 失败原因、结果地址、上游响应和原始任务数据均可复制
- `SUCCESS` 任务的 `fail_reason` 与 `result_url` 相同时按旧版结果 URL 处理，不显示失败区块或失败色入口
- 原始任务数据默认折叠，避免影响主要排障信息的扫描
- 长内容限制可视高度并允许滚动、换行，移动端弹窗宽度受视口约束

## 4. 数据边界

后端 `TaskDto` 已返回：

- `data`
- `properties`
- `fail_reason`
- `result_url`
- 任务状态、时间、渠道、用户、分组和额度字段

前端把 `data` 和 `properties` 视为 `unknown`，统一经过任务 payload 格式化边界：

- JSON 字符串先解析并美化
- 普通字符串原样显示
- 对象、数组和标量安全序列化
- `null`、`undefined` 和空白字符串视为空响应
- 无法序列化的值降级为可读字符串，不中断任务日志页面

`Task.PrivateData` 的 JSON 标签为 `-`，不在 `TaskDto` 中返回，因此该弹窗不会扩大敏感数据暴露范围。

## 5. 影响范围

### 1. `web/default/src/features/usage-logs/types.ts`

补齐 `TaskLog` 与后端 `TaskDto` 一致的字段，并将 `data` / `properties` 定义为不受信任的 `unknown` 输入。

### 2. `web/default/src/features/usage-logs/lib/task-details.ts`

集中处理上游响应解析、格式化、Suno 音频条目投影和旧版成功结果 URL 判定，避免组件分散断言 payload 结构。

### 3. `web/default/src/features/usage-logs/lib/task-details.test.ts`

覆盖对象、数组、JSON 字符串、普通字符串、标量、空值、无法序列化值、音频数组投影，以及成功/失败任务的旧版结果 URL 判定。

### 4. `web/default/src/features/usage-logs/components/dialogs/task-details-dialog.tsx`

新增任务详情弹窗，展示完整排障信息、复制入口、结果链接、音频预览和折叠的原始任务 JSON。

### 5. 任务日志表格与移动卡片

- `components/columns/task-logs-columns.tsx`：所有任务固定展示详情按钮，弹窗只在打开时挂载
- `components/usage-logs-mobile-card.tsx`：移动卡片将原“结果”标签改为“详情”

### 6. `web/default/src/i18n/locales/*.json`

为 en、zh、zh-TW、fr、ja、ru、vi 补齐任务详情相关文案。

### 7. `scripts/verify_patches.sh`

在前端定制定向测试组中加入任务 payload 格式化测试。

## 6. 风险点

- 极大的上游响应仍会占用浏览器内存；当前只在用户打开单条任务详情时格式化，不在表格渲染阶段批量处理
- `result_url` 对旧任务可能回退为 `fail_reason`；仅当状态为 `SUCCESS` 且两者规范化后相同时隐藏失败态，真实失败任务仍完整展示原因
- 上游响应只按文本安全渲染，不执行 HTML，也不提供 JSON 编辑能力
- 前端显示内容取决于任务落库时保存的 `data`，历史空数据无法由该二开补回

## 7. 测试方案

最小验证命令：

```bash
cd web/default
bun test src/features/usage-logs/lib/task-details.test.ts
bun run typecheck
bun run build
cd ../..
make verify-patches
```

浏览器验收：

- 桌面和移动视口均能从每条任务打开详情
- 对象、数组、普通字符串和长 JSON 均完整显示并可复制
- 空 `data` 不显示“上游响应”区块
- 旧版成功视频任务不显示失败原因或失败色入口，结果链接仍可打开
- 失败任务即使 `fail_reason` 与 `result_url` 相同也仍展示失败原因
- Escape 关闭后焦点返回详情按钮
