# 008-seedance-asset-library-videos

## 1. 背景

当前 Seedance 通过兼容 `/v1/videos` 的直连代理接入 NewAPI。Seedance2 上游新增真人形象 IP 资产库能力：下游传入真人图片 URL 后，上游异步处理并最终返回 `AssetId`，后续视频生成可使用该资产 ID。

本定制将资产库创建复用 NewAPI 既有 OpenAI Videos 异步任务链路，不新增下游资产专用端点。

## 2. 目标

- 使用 `POST /v1/videos` 提交资产任务
- 使用 `GET /v1/videos/{task_id}` 查询资产任务
- 通过虚拟模型 `seedance-asset` 识别资产任务
- 复用现有任务入库、后台轮询、用户隔离和失败退款机制
- 资产任务不叠加视频时长、分辨率或参考视频倍率
- 保留代理返回的顶层 `asset_id` 和 `metadata.seedance.asset_id`
- 复用 `/api/task/token/self` 让只持有 API Key 的下游查询当前 key 创建过的资产任务
- 提供 `POST /api/task/token/asset/delete`，让只持有 API Key 的下游删除当前 key 创建的成功资产任务

不解决：

- 不新增专用“我的资产列表”下游 API
- 不新增 `DELETE /v1/videos/{task_id}` OpenAI Videos 风格删除路由
- 不写入 NewAPI 默认模型/价格表
- 不支持单请求批量创建多个资产

## 3. 业务规则

资产任务请求示例：

```json
{
  "model": "seedance-asset",
  "prompt": "林春芽",
  "input_reference": "https://example.com/person.png"
}
```

- `prompt` 是资源名称/资产显示名，不是 NewAPI 用户名
- 图片 URL 字段优先级：`input_reference -> image -> images[0] -> files[0]`
- 只校验并使用第一张图片；多图处理和 `ignored_image_count` 由上游代理响应体现
- 图片 URL 必须是 `http://` 或 `https://`
- 显式 localhost、回环 IP、私网 IP、链路本地地址会在 NewAPI 侧提前拒绝
- `seedance-asset` 仍需管理员在后台配置模型、渠道和按次价格
- `/api/task/token/self` 是当前 API Key 维度的任务列表：模型层按 `user_id + token_id` 查询，只返回当前 key 创建的新任务，不返回同一用户其他 key 创建的任务
- 下游要列出可用真人资产时，先调用 `/api/task/token/self?status=SUCCESS`，再在客户端过滤 `data.model == "seedance-asset"`、`properties.origin_model_name == "seedance-asset"` 或 `data.metadata.seedance.kind == "asset"`
- 可用真人资产筛选必须排除 `data.deleted == true` 或 `data.metadata.seedance.deleted == true`
- 可用于视频生成的资源地址优先读取 `data.asset_uri`，其次读取 `data.metadata.seedance.asset_uri`；若只有 `asset_id`，客户端可拼成 `asset://<asset_id>`
- 删除资产时下游只传当前 NewAPI 返回的 `task_id`，不传上游 `resource_id`
- NewAPI 校验当前 API Key 对该任务的归属后，把任务记录里的上游任务 ID 转发到渠道 `base_url` 的 `POST /api/task/token/asset/delete`
- 当渠道上游仍是 NewAPI 时，上游 NewAPI 会继续按自己的任务记录转发；当渠道上游是 Go 代理时，Go 代理直接按 `asset_req_...` 查询并删除真实资产资源
- 删除成功后不删除任务历史、不退款、不改任务状态，只在 task data 和 `metadata.seedance` 标记 `deleted=true`、`deleted_at`

API Key 任务列表示例：

```http
GET /api/task/token/self?status=SUCCESS&p=1&page_size=20
Authorization: Bearer <NewAPI API Key>
```

客户端筛选真人资产示例：

```js
const assets = page.data.items.filter((task) => {
  const data = task.data || {};
  const metadata = data.metadata?.seedance || {};

  return task.status === "SUCCESS"
    && (
      data.model === "seedance-asset"
      || task.properties?.origin_model_name === "seedance-asset"
      || metadata.kind === "asset"
    )
    && !data.deleted
    && !metadata.deleted
    && (data.asset_uri || metadata.asset_uri);
});
```

API Key 删除资产示例：

```http
POST /api/task/token/asset/delete
Authorization: Bearer <NewAPI API Key>
Content-Type: application/json
```

```json
{
  "task_id": "asset_req_1780830000_xxx"
}
```

成功响应：

```json
{
  "success": true,
  "message": "",
  "data": {
    "task_id": "asset_req_1780830000_xxx",
    "deleted": true,
    "deleted_at": 1780830000,
    "asset_id": "asset-xxx",
    "asset_uri": "asset://asset-xxx"
  }
}
```

## 4. 影响范围

### 1. `relay/channel/task/sora/adaptor.go`

- 新增 `seedance-asset` 资产任务识别
- 资产任务校验只要求：
  - `model=seedance-asset`
  - `prompt` 非空
  - 存在公网图片 URL
- `EstimateBilling` 对资产任务返回空倍率，避免叠加视频 `seconds`、`size` 和参考视频倍率
- `prepareReferenceVideoBilling` 对资产任务直接跳过
- `DoResponse` 改为基于原始 JSON 覆盖公开 `id/task_id`，避免结构体重编码丢失 `asset_id` 等扩展字段

### 2. `relay/common/relay_info.go`

`TaskSubmitReq` 新增 `Files []string`，用于读取 OpenAI Videos 风格 `files` 字段中的资产图片 URL。

### 3. `relay/channel/task/sora/adaptor_test.go`

新增回归测试：

- `seedance-asset` 不返回视频计费倍率
- 私网/回环 URL 被拒绝
- `files[0]` 图片 URL 可作为资产图片输入
- `DoResponse` 覆盖公开任务 ID 时保留顶层 `asset_id` 和 metadata

### 4. `controller/task.go`

- 新增 `DeleteUserTokenAsset`
- 使用 `TokenAuthReadOnly()` 上下文中的 `user_id + token_id + task_id` 查任务
- 校验任务为已成功、未删除的 `seedance-asset` 资产任务
- 从任务 `PrivateData.UpstreamTaskID` 读取上游任务 ID；旧数据无该字段时回退 `TaskID`
- 调原渠道 `POST /api/task/token/asset/delete`，请求体 `{"task_id":"<upstream_task_id>"}`
- 删除成功后保留任务历史，只标记 `deleted` 和 `deleted_at`

### 5. `model/task.go`

- 新增 `GetByUserTokenTaskId`，按当前用户、当前 API Key 和任务 ID 查询单个任务

### 6. `router/api-router.go`

- 新增 `POST /api/task/token/asset/delete`
- 使用 `TokenAuthReadOnly()`，不走 `/v1/videos` relay 分发，不触发计费

### 7. `controller/task_token_test.go`

- 补充资产删除回归测试：
  - 当前 API Key 创建的成功资产可以删除
  - 删除调用统一转发到上游 `POST /api/task/token/asset/delete`
  - 同用户其他 API Key 的资产不能删除
  - 非资产、未成功、已删除的任务会拒绝

## 5. 风险点

- `seedance-asset` 不写入默认模型/价格表；未在后台配置时，请求会因渠道/计费配置缺失失败
- NewAPI 只做轻量 URL 校验，不做 DNS 解析或 DNS 重绑定防护
- 资产任务仍走 `/v1/videos`，响应 `object` 保持 `video`，需要文档说明这是资产模式
- 如果上游代理后续调整资产响应字段，需确认 `asset_id` 和 metadata 是否仍被保留
- 删除调用使用任务记录中的渠道，并按渠道现有 key 选择逻辑取 key；如果渠道 key 后续轮换，行为与现有任务轮询链路保持一致
- 旧数据没有 `PrivateData.UpstreamTaskID` 时会回退本地 `TaskID`；如果该任务的本地 ID 不是上游可识别 ID，删除会由上游返回失败

## 6. 测试方案

最小验证命令：

```bash
go test ./relay/channel/task/sora ./relay/common ./relay ./controller
go test ./controller -run 'Asset|TaskToken' -count=1
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `relay/channel/task/sora/adaptor.go` 中 `ValidateRequestAndSetAction`、`EstimateBilling`、`DoResponse` 是否调整
- `relay/common/relay_info.go` 中 `TaskSubmitReq` 字段是否调整
- `/v1/videos` 任务提交和轮询流程是否改变公开任务 ID 替换方式
- `/api/task/token/self` 的鉴权上下文字段 `id` / `token_id` 是否调整
- `model.Channel.GetNextEnabledKey()` 和渠道 base URL 获取逻辑是否调整

## 8. 当前状态

- 已实现 `seedance-asset` 资产任务校验分支
- 已实现资产任务计费倍率绕过
- 已保留资产响应扩展字段
- 已实现 API Key 维度资产删除接口
- 已补充 Sora 适配器回归测试
- 已补充资产删除 controller 回归测试
- 已生成 `patches/008-seedance-asset-library-videos.patch`
