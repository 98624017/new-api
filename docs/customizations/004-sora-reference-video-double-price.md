# 004-sora-reference-video-double-price

## 1. 背景

部分走 `/v1/videos` 的 Sora 兼容请求会在 `content` 数组中携带参考视频：

```json
{
  "type": "video_url",
  "role": "reference_video",
  "video_url": {
    "url": "https://example.com/reference.mp4"
  }
}
```

这类请求的上游成本与普通文生视频请求不同。当前支持两种计费策略：

```text
默认策略：最终额度 = 基础额度 × 生成时长 × size × video_input(2)
精确策略：最终额度 = 基础额度 × (生成时长 + 参考视频总时长) × size
```

默认保持旧的简单双倍计价；只有显式开启精确参考视频时长计费时，才会检测视频 URL 时长。

## 2. 目标

- 仅在 Sora 任务适配器的 `/v1/videos` 提交链路中生效
- 仅当模型名在环境变量白名单内时生效
- 仅当请求体 `content` 中包含参考视频时生效
- 默认关闭精确时长检测，命中后通过现有 `OtherRatios` 机制追加 `video_input: 2`
- 开启精确时长检测后，先检测所有参考视频 URL 的时长，再把 `seconds` 改为 `生成时长 + 参考视频总时长`
- 精确时长检测开启时，参考视频时长检测失败会拒绝请求，不提交上游

不解决：

- 不改动其他视频模型的默认计费
- 不新增后台配置页面
- 不改变 `size` 现有计费逻辑

## 3. 业务规则

- 模型白名单环境变量：`SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS`
- 精确时长计费开关：`SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED`
- 模型白名单默认值为空，不配置时任何模型都不会触发参考视频额外计费
- 精确时长计费开关默认关闭，不配置或配置为空时保持旧的参考视频双倍计费
- 多个模型用英文逗号分隔，例如：

```bash
SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS=seedance-2.0,seedance-2.0-pro
SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED=true
```

- 请求模型不在白名单内时，即使 `content` 包含 `video_url` 也不额外计费
- 判断参考视频的条件：
  - `content[].type == "video_url"`；或
  - `content[]` 对象中存在 `video_url` 字段
- 精确时长计费关闭时，只要识别到参考视频就按 `video_input: 2` 计费，不下载或解析 URL
- 精确时长计费开启时：
  - 多个参考视频按总秒数累加
  - 每次提交的参考视频时长检测总超时为 30 秒
  - 视频时长检测优先使用 HTTP Range 读取前 1MiB 元数据；如果无法解析，再回退到完整受限下载
  - Range 与完整下载都无法解析出 MP4/MOV 时长，或检测超时时，返回 `reference_video_duration_unavailable`
  - 检测失败或超时时在本地请求校验阶段拒绝，不提交上游任务
  - 加计后的 `seconds` 继续与现有分辨率倍率相乘

示例：

```text
默认：最终额度 = 基础额度 × seconds × size × video_input
精确：最终额度 = 基础额度 × (生成时长 + 参考视频总时长) × size
```

## 4. 影响范围

- `relay/common/relay_info.go`
  - `TaskSubmitReq` 增加 `Content` 字段，用于保留 `/v1/videos` 请求体中的顶层 `content`
- `relay/channel/task/sora/constants.go`
  - 从 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 加载参考视频计费模型白名单
  - 从 `SORA_REFERENCE_VIDEO_DURATION_BILLING_ENABLED` 加载精确时长计费开关
- `main.go`
  - 在 `.env` 加载和 `common.InitEnv()` 之后显式刷新 Sora 参考视频计费白名单
- `relay/channel/task/sora/adaptor.go`
  - 未开启精确计费时，白名单模型携带参考视频返回 `video_input: 2`
  - 开启精确计费时，在请求校验阶段检测参考视频总时长，并在 `EstimateBilling` 中把 `seconds` 设置为生成时长与参考视频总时长之和
- `relay/channel/task/sora/video_duration.go`
  - 提取参考视频 URL
  - 为参考视频时长检测设置 30 秒总超时
  - Range 优先探测 MP4/MOV 元数据时长
  - Range 失败后回退完整受限下载
- `service/download.go`
  - Worker 请求支持 context 取消，确保参考视频检测超时时 Worker 模式也能及时返回
- `relay/channel/task/sora/adaptor_test.go`
  - 覆盖多参考视频累加、Range 失败回退完整下载、解析失败拒绝、超时拒绝，以及未配置时不生效

## 5. 风险点

- 白名单在 `.env` 和环境变量加载后刷新，修改后需要重启服务
- 精确时长计费开关也在启动时加载，修改后需要重启服务
- 若未来请求体结构变化，不再使用顶层 `content` 或 `video_url`，需要扩展识别逻辑
- 若模型映射规则把计费模型名改写成其他名称，需要确认白名单里使用的是最终计费模型名还是原始请求模型名
- 开启精确计费后，参考视频 URL 需要能被服务端或 Worker 访问；不可访问、超过下载上限或无法解析时会直接拒绝请求
- 开启精确计费后，Range 探测失败会触发完整受限下载，参考视频较大或多个视频时会增加提交延迟和带宽消耗，但总检测时间最多 30 秒

## 6. 测试方案

最小验证命令：

```bash
go test ./relay/channel/task/sora
go test ./relay/common
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `relay/common/relay_info.go` 中 `TaskSubmitReq` 是否重构
- `main.go` 中 `.env` / `common.InitEnv()` 的启动顺序是否变化
- `relay/channel/task/sora/adaptor.go` 中 `EstimateBilling` 是否调整
- `relay/channel/task/sora/constants.go` 中环境变量加载是否调整
- `relay/channel/task/sora/video_duration.go` 的 MP4/MOV 时长解析是否仍适配请求格式
- `relay/relay_task.go` 中 `OtherRatios` 乘算逻辑是否调整

## 8. 当前状态

- 已实现环境变量模型白名单限定
- 已实现 `content` 中参考视频识别
- 已实现默认旧双倍计费
- 已实现精确参考视频时长计费开关
- 已实现参考视频时长检测和多视频累加
- 已实现参考视频时长检测 30 秒超时保护
- 已实现 Range 优先、完整受限下载回退
- 已实现精确计费开启时检测失败或超时拒绝请求
- 已补充 Sora 适配器测试
- 已生成 `patches/004-sora-reference-video-double-price.patch`
