# 009-sora-unknown-status-polling

## 1. 背景

当本项目把另一个 NewAPI 实例作为上游 Sora/OpenAI Videos 兼容渠道时，上游任务刚创建或还未完成状态归一时，`GET /v1/videos/{task_id}` 可能返回 OpenAI 视频格式的 `status: "unknown"`。

旧实现的 Sora 适配器只识别 `queued`、`pending`、`processing`、`in_progress`、`completed`、`failed`、`cancelled`。遇到 `unknown` 时不会产出内部任务状态，后台轮询随后把响应判定为 `upstream returned unrecognized message`，导致本地任务提前失败。实际情况是上游任务可能仍在执行，后续会成功。

## 2. 目标

- 兼容上游 NewAPI/OpenAI Videos 风格的 `status: "unknown"`
- 兼容大小写和常见同义状态：`submitted`、`not_start`、`running`、`succeeded`、`canceled`
- 对长得像视频任务、临时缺失 `status` 且不带明确 `error` 的响应，按仍在处理中处理
- 对缺失状态或 `unknown` 但带有明确 `error.message` / `error.code` 的响应，按失败处理
- 对缺失状态或 `unknown`、不带明确错误但已经返回明确视频结果 URL 的响应，按成功处理并写入结果 URL
- 对外 `/v1/videos/{id}` 查询使用本地任务状态覆盖上游旧 raw data，避免内部已成功但客户端仍看到 `unknown`
- 对外 `/v1/videos/{id}/content` 在已保存非本地代理结果 URL 时优先拉取该 URL，避免继续请求上游 `/content`
- 不把单独的 `progress: 100` 当作任务成功信号，避免上游状态仍未完成时提前结算
- 保留真正无法识别的空对象或非任务响应继续走原有失败路径

不解决：

- 不修改全局轮询错误处理策略
- 不让所有缺失 `status` 的响应都静默等待
- 不改变其他任务适配器的状态映射

## 3. 业务规则

上游返回示例：

```json
{
  "id": "video_123",
  "object": "video",
  "model": "seedance",
  "status": "unknown",
  "progress": 0
}
```

本地处理为：

- 内部状态：`IN_PROGRESS`
- 轮询继续等待下一次上游状态
- 不触发失败退款
- 不写入 `upstream returned unrecognized message`

若上游只返回 `{}`，仍保持旧行为：Sora 适配器返回空状态，轮询层按无法识别响应处理。

若上游在状态仍为 `unknown` 时已经返回明确结果链接：

```json
{
  "id": "video_123",
  "object": "video",
  "model": "seedance",
  "status": "unknown",
  "video_url": "https://cdn.example.com/result.mp4"
}
```

本地处理为：

- 内部状态：`SUCCESS`
- 结果链接：`https://cdn.example.com/result.mp4`
- 触发原有成功结算路径
- 客户端再次查询 `/v1/videos/{id}` 时看到 `status: "completed"` 和 `metadata.url`
- 客户端请求 `/v1/videos/{id}/content` 时优先从 `metadata.url` / `ResultURL` 对应直链拉取内容

若上游返回无状态但包含明确错误：

```json
{
  "id": "video_123",
  "object": "video",
  "model": "seedance",
  "error": {
    "message": "upstream rejected video task",
    "code": "invalid_request"
  }
}
```

本地处理为：

- 内部状态：`FAILURE`
- 失败原因：`error.message`
- 触发原有失败退款路径

## 4. 影响范围

### 1. `relay/channel/task/sora/adaptor.go`

- `ParseTaskResult` 改为先归一化状态字符串
- `unknown` 和缺失状态但具备视频任务特征的响应映射为 `IN_PROGRESS`
- 缺失状态或 `unknown` 但带有明确 `error.message` / `error.code` 的视频任务响应映射为 `FAILURE`
- 缺失状态或 `unknown` 且包含 `url`、`video_url`、`metadata.url`、`content.video_url`、`output.video_url` 等明确结果链接时映射为 `SUCCESS`
- `progress: 100` 无结果链接时仍保持处理中，不单独视为成功
- `ConvertToOpenAIVideo` 根据本地 `Task.Status` / `Task.Progress` / `PrivateData.ResultURL` 同步对外状态、进度和 `metadata.url`
- 扩展常见兼容状态映射：
  - `submitted`、`not_start` -> `QUEUED`
  - `running` -> `IN_PROGRESS`
  - `succeeded` -> `SUCCESS`
  - `canceled` -> `FAILURE`

### 2. `controller/video_proxy.go`

- OpenAI/Sora content 代理在 `ResultURL` 已经是非本地代理 URL 时优先使用该 URL
- `ResultURL` 为空或仍是本地 `/v1/videos/{task_id}/content` 代理 URL 时，保持原有请求上游 `/v1/videos/{upstream_id}/content` 行为

### 3. `relay/channel/task/sora/adaptor_test.go`

新增回归测试：

- `status: "unknown"` 的 OpenAI 视频响应保持处理中
- 缺失 `status` 但包含视频任务字段的响应保持处理中
- `status: "unknown"` 但包含明确结果 URL 的响应进入成功
- `status: "unknown"` 且仅 `progress: 100` 的响应仍保持处理中
- 缺失 `status` 但包含明确错误的响应进入失败
- 同时包含明确错误和结果 URL 时，错误优先，任务进入失败
- 空对象仍保留空状态，避免吞掉真正无法识别的响应
- 已保存成功结果 URL 的任务对外转换为 `completed`，并在 `metadata.url` 中返回结果 URL

### 4. `controller/video_proxy_test.go`

- 覆盖 Sora 成功任务保存直链时，`/v1/videos/{id}/content` 命中保存的结果 URL，不再请求上游 `/content`

## 5. 风险点

- 如果某个上游错误响应没有使用 `error.message` / `error.code`，但又携带 `id`、`object: "video"` 或 `model`，本地会暂时继续轮询而不是立即失败
- 如果兼容上游把非结果用途的 `url` / `video_url` 放在任务状态响应里，本地可能把它当作结果链接；当前只在 Sora/OpenAI Videos 兼容链路内启用该兜底
- 如果兼容上游返回的结果 URL 需要特殊认证头，本地直链拉取不会把渠道 API Key 泄露给非上游域名；这类 URL 应由上游返回可直接访问的签名链接
- 该补丁只解决 Sora/OpenAI Videos 兼容链路；其他渠道若有类似问题，需要单独在对应适配器处理
- 如果上游长期只返回 `unknown` 且没有结果链接，任务会依赖现有超时清理机制最终失败

## 6. 测试方案

最小验证命令：

```bash
go test ./relay/channel/task/sora -count=1
go test ./controller -run 'TestVideoProxyUsesStoredResultURLForSoraTask' -count=1
make verify-patches
```
