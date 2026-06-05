# 007-seedance-reference-video-double-price

## 1. 背景

Seedance 视频生成在当前部署中通过 NewAPI 的 `Sora` / `OpenAI` 视频任务机制接入：后台渠道类型仍选择 `Sora` 或 `OpenAI`，上游 Base URL 指向兼容 `/v1/videos` 的 Seedance 服务。

客户端请求体可通过 OpenAI Videos 风格顶层字段携带参考视频，例如：

```json
{
  "model": "doubao-seedance-2-0-260128-2",
  "prompt": "Use the reference video motion style",
  "input_video": ["https://example.com/reference.mp4"]
}
```

这类视频参考请求需要与普通文生视频区分计费。此前 Sora 兼容渠道已有 `content[].video_url` 双倍计费和精确按秒计费能力；本定制只为 Seedance 顶层参考视频字段增加简单双倍计费，不启用新的按秒规则。

## 2. 目标

- 在实际使用的 `Sora` / `OpenAI` 视频任务适配器路径中生效
- 不要求后台新增或选择 `DoubaoVideo` 渠道类型
- 仅当模型名在环境变量白名单内时生效
- 识别 OpenAI Videos 风格顶层参考视频字段
- 命中后通过现有 `OtherRatios` 机制追加 `video_input: 2`
- 保留原有 Sora `content[].video_url` 参考视频计费逻辑
- 保留 Doubao/VolcEngine 旧 `metadata.content` 计费逻辑，不改其上游请求格式

不解决：

- 不新增后台配置页面
- 不下载或解析 Seedance 顶层参考视频时长
- 不改变其他非白名单模型的计费

## 3. 业务规则

- Seedance 模型白名单环境变量：`SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS`
- 该变量会并入 Sora 任务适配器的参考视频双倍计费白名单
- 原 Sora 白名单 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 继续生效
- 模型白名单默认值为空，不配置时任何 Seedance 模型都不会触发本定制双倍计费
- 多个模型用英文逗号分隔，例如：

```bash
SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS=doubao-seedance-2-0-260128-2,doubao-seedance-2-0-260128-3
```

- 请求模型不在白名单内时，即使包含参考视频也不按本定制双倍计费
- 判断顶层参考视频的条件：
  - `input_video` 存在非空 URL 字符串或数组项
  - `video_url` 存在非空 URL 字符串或数组项
  - `reference_video` 存在非空 URL 字符串或数组项
  - `files` 中存在明显视频扩展名 URL，例如 `.mp4`、`.mov`、`.webm`
- `files` 中只有图片或音频 URL 时不触发双倍计费
- Sora 适配器会把 JSON 请求体按原始 map 透传给上游，只替换映射后的 `model`，因此上述顶层字段会正常传到 Seedance 上游
- 原有 `content[].video_url` 中的视频输入识别保留：
  - 白名单模型按 `video_input: 2`
  - 开启 Sora 精确按秒计费时继续走原有秒数探测逻辑

## 4. 影响范围

- `relay/channel/task/sora/constants.go`
  - 从 `SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 加载 Seedance 模型白名单，并合并到 Sora 参考视频计费白名单
- `relay/channel/task/sora/adaptor.go`
  - 在 `EstimateBilling` 中识别白名单模型和 OpenAI Videos 顶层参考视频字段
  - 命中顶层参考视频时返回 `video_input: 2`
  - 保留原有 `content[].video_url` 视频输入计费路径
- `relay/channel/task/sora/adaptor_test.go`
  - 覆盖 Seedance 白名单、顶层视频字段、图片/音频非视频字段、未配置白名单、请求体字段透传
- `main.go`
  - 继续在 `.env` 加载和 `common.InitEnv()` 之后刷新 Sora 参考视频计费白名单；Seedance 白名单由 Sora reload 一并加载
- `relay/channel/task/doubao/*`
  - 不承载本次 Seedance OpenAI Videos 顶层字段计费逻辑，保留旧 VolcEngine/Doubao 行为

## 5. 风险点

- 白名单在 `.env` 和环境变量加载后刷新，修改后需要重启服务
- `files` 通过 URL 扩展名判断是否为视频；无扩展名或扩展名不标准的视频 URL 不会触发
- `input_video` / `video_url` / `reference_video` 只判断是否存在非空 URL，不探测真实媒体类型
- 若未来 Seedance 请求体字段变化，需要扩展 Sora 适配器中的识别逻辑
- 若模型映射规则改变计费模型名，需要确认白名单使用的是原始请求模型名还是计费模型名

## 6. 测试方案

最小验证命令：

```bash
go test ./relay/channel/task/sora
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `relay/channel/task/sora/adaptor.go` 中 `EstimateBilling` 是否调整
- `relay/channel/task/sora/constants.go` 中参考视频模型列表和环境变量加载是否调整
- `main.go` 中 `.env` / `common.InitEnv()` 的启动顺序是否变化
- `relay/relay_task.go` 中 `OtherRatios` 乘算逻辑是否调整

## 8. 当前状态

- 已实现环境变量模型白名单限定
- 已实现 OpenAI Videos 风格顶层参考视频识别
- 已接入实际使用的 Sora/OpenAI 视频任务路径
- 已实现参考视频双倍计费
- 已保留原 `content[].video_url` 视频输入计费逻辑
- 已补充 Sora/Seedance 适配器测试
- 已生成 `patches/007-seedance-reference-video-double-price.patch`
