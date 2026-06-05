# 007-seedance-reference-video-double-price

## 1. 背景

Seedance/Doubao 视频生成渠道支持 OpenAI Videos 风格 `/v1/videos` 请求。下游请求体可通过顶层字段携带参考视频，例如：

```json
{
  "model": "doubao-seedance-2-0-260128-2",
  "prompt": "Use the reference video motion style",
  "input_video": ["https://example.com/reference.mp4"]
}
```

这类视频参考请求需要与普通文生视频区分计费。此前 Sora 兼容渠道已有参考视频双倍计费和精确按秒计费能力；本定制只为 Seedance/Doubao 增加简单双倍计费，不启用精确按秒规则。

## 2. 目标

- 仅在 Seedance/Doubao 视频任务适配器中生效
- `DoubaoVideo` 渠道走 Seedance OpenAI Videos 顶层字段格式；`VolcEngine` 渠道继续保留旧 Doubao `content[]` 请求格式
- 仅当模型名在环境变量白名单内时生效
- 识别 OpenAI Videos 风格顶层参考视频字段
- 命中后通过现有 `OtherRatios` 机制追加 `video_input: 2`
- 保留原有 `metadata.content` 中 `video_url` 的视频输入计费逻辑

不解决：

- 不新增后台配置页面
- 不下载或解析参考视频时长
- 不改变其他渠道的参考视频计费

## 3. 业务规则

- 模型白名单环境变量：`SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS`
- 模型白名单默认值为空，不配置时任何模型都不会触发 Seedance 参考视频双倍计费
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
- OpenAI Videos 风格顶层字段仅在 `DoubaoVideo` 渠道参与计费判断；`VolcEngine` 旧渠道不会因为顶层 `input_video` / `files` 等字段触发双倍计费
- `files` 中只有图片或音频 URL 时不触发双倍计费
- 已有 `metadata.content` 中 `video_url` 的视频输入识别保留：
  - 白名单模型按 `video_input: 2`
  - 非白名单模型继续使用原 `videoInputRatioMap` 中的模型专属倍率

## 4. 影响范围

- `relay/channel/task/doubao/constants.go`
  - 从 `SEEDANCE_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 加载参考视频双倍计费模型白名单
- `main.go`
  - 在 `.env` 加载和 `common.InitEnv()` 之后显式刷新 Seedance/Doubao 参考视频计费白名单
- `relay/channel/task/doubao/adaptor.go`
  - 按渠道类型分流上游请求：`DoubaoVideo` 使用 `/v1/videos` 并保留顶层 `files` / `input_video` 等字段；`VolcEngine` 使用旧 `/api/v3/contents/generations/tasks`
  - 在 `EstimateBilling` 中识别白名单模型和参考视频字段
  - 命中 OpenAI Videos 风格参考视频时返回 `video_input: 2`
  - 保留原有 `metadata.content` 视频输入计费路径
- `service/task_polling.go`
  - 轮询初始化时把渠道类型传给视频适配器，保证查询任务也能按 `DoubaoVideo` / `VolcEngine` 分流
- `relay/channel/task/doubao/adaptor_test.go`
  - 覆盖白名单、顶层视频字段、图片/音频非视频字段、未配置白名单、原 metadata 路径，以及两个渠道请求体/URL 分流

## 5. 风险点

- 白名单在 `.env` 和环境变量加载后刷新，修改后需要重启服务
- `files` 通过 URL 扩展名判断是否为视频；无扩展名或扩展名不标准的视频 URL 不会触发
- 若未来 Seedance 请求体字段变化，需要扩展识别逻辑
- 若模型映射规则改变计费模型名，需要确认白名单使用的是原始请求模型名还是计费模型名

## 6. 测试方案

最小验证命令：

```bash
go test ./relay/channel/task/doubao
```

完整二开校验：

```bash
make verify-patches
```

## 7. 升级关注点

上游同步时重点关注：

- `relay/channel/task/doubao/adaptor.go` 中 `EstimateBilling` 是否调整
- `relay/channel/task/doubao/constants.go` 中模型列表和环境变量加载是否调整
- `main.go` 中 `.env` / `common.InitEnv()` 的启动顺序是否变化
- `relay/relay_task.go` 中 `OtherRatios` 乘算逻辑是否调整

## 8. 当前状态

- 已实现环境变量模型白名单限定
- 已实现 OpenAI Videos 风格顶层参考视频识别
- 已实现 `DoubaoVideo` 新格式和 `VolcEngine` 旧格式分流
- 已实现参考视频双倍计费
- 已保留原 `metadata.content` 视频输入计费逻辑
- 已补充 Doubao/Seedance 适配器测试
- 已生成 `patches/007-seedance-reference-video-double-price.patch`
