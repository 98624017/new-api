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

这类请求的上游成本与普通文生视频请求不同，需要对指定模型做双倍计价。

## 2. 目标

- 仅在 Sora 任务适配器的 `/v1/videos` 提交链路中生效
- 仅当模型名在环境变量白名单内时生效
- 仅当请求体 `content` 中包含参考视频时生效
- 命中后通过现有 `OtherRatios` 机制追加 `video_input: 2`

不解决：

- 不改动其他视频模型的默认计费
- 不新增后台配置页面
- 不改变 `seconds`、`size` 现有计费逻辑

## 3. 业务规则

- 环境变量名：`SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS`
- 默认值为空，不配置时任何模型都不会触发参考视频双倍计价
- 多个模型用英文逗号分隔，例如：

```bash
SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS=seedance-2.0,seedance-2.0-pro
```

- 请求模型不在白名单内时，即使 `content` 包含 `video_url` 也不双倍计价
- 判断参考视频的条件：
  - `content[].type == "video_url"`；或
  - `content[]` 对象中存在 `video_url` 字段
- 双倍计价与现有时长、分辨率倍率相乘

示例：

```text
最终额度 = 基础额度 × seconds × size × video_input
```

## 4. 影响范围

- `relay/common/relay_info.go`
  - `TaskSubmitReq` 增加 `Content` 字段，用于保留 `/v1/videos` 请求体中的顶层 `content`
- `relay/channel/task/sora/constants.go`
  - 从 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 加载参考视频双倍计价模型白名单
- `main.go`
  - 在 `.env` 加载和 `common.InitEnv()` 之后显式刷新 Sora 参考视频双倍计价白名单
- `relay/channel/task/sora/adaptor.go`
  - 在 `EstimateBilling` 中识别白名单模型和参考视频
- `relay/channel/task/sora/adaptor_test.go`
  - 覆盖环境变量命中白名单双倍计价和未配置时不生效

## 5. 风险点

- 白名单在 `.env` 和环境变量加载后刷新，修改后需要重启服务
- 若未来请求体结构变化，不再使用顶层 `content` 或 `video_url`，需要扩展识别逻辑
- 若模型映射规则把计费模型名改写成其他名称，需要确认白名单里使用的是最终计费模型名还是原始请求模型名

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
- `relay/relay_task.go` 中 `OtherRatios` 乘算逻辑是否调整

## 8. 当前状态

- 已实现环境变量模型白名单限定
- 已实现 `content` 中参考视频识别
- 已补充 Sora 适配器测试
- 已生成 `patches/004-sora-reference-video-double-price.patch`
