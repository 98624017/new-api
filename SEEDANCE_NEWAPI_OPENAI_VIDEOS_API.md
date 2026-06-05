# Seedance NewAPI 视频 API 调用文档

本文档面向最下游客户端，说明如何通过已接入 Seedance 渠道的 NewAPI 实例创建和查询视频任务。

## 1. 基础信息

Base URL:

```text
https://ay.light-ai.cloud
```

请求鉴权：

```http
Authorization: Bearer <NEWAPI_API_KEY>
```

请求体格式：

```http
Content-Type: application/json
```

这里使用的是 NewAPI 分发给客户端的 API Key，不是 Seedance 上游 token。

## 2. 创建视频任务

### 2.1 请求端点

```http
POST /v1/videos HTTP/1.1
Host: ay.light-ai.cloud
Authorization: Bearer <NEWAPI_API_KEY>
Content-Type: application/json
```

完整 URL：

```text
https://ay.light-ai.cloud/v1/videos
```

### 2.2 请求体字段

| 字段 | 类型 | 必填 | 说明 |
|---|---:|---:|---|
| `model` | string | 是 | Seedance 原始上游模型 ID，推荐使用 `doubao-seedance-2-0-260128-2` |
| `prompt` | string | 是 | 视频生成提示词 |
| `duration` | string | 否 | 输出时长，单位秒，范围 `"4"` 到 `"15"`，默认 `"4"` |
| `seconds` | string | 否 | OpenAI Videos 风格时长字段；未传 `duration` 时会作为 `duration` 使用，范围同 `duration` |
| `aspect_ratio` | string | 否 | 输出比例，可选 `"21:9"`、`"16:9"`、`"4:3"`、`"1:1"`、`"3:4"`、`"9:16"`，默认 `"16:9"` |
| `files` | string[] 或 string | 否 | 参考素材公网 URL，可包含图片、视频、音频 |
| `generate_audio` | boolean/string | 否 | 是否生成音频，可选 `true` / `false`，默认 `true` |
| `watermark` | boolean/string | 否 | 是否加水印，可选 `true` / `false`，默认 `false` |
| `resolution` | string | 否 | 清晰度，可选 `"480p"`、`"720p"`、`"1080p"`，默认 `"480p"` |

参数取值范围：

| 字段 | 可选值 / 范围 | 默认值 |
|---|---|---|
| `duration` | `"4"` 到 `"15"` | `"4"` |
| `seconds` | `"4"` 到 `"15"`；未传 `duration` 时生效 | 无 |
| `aspect_ratio` | `"21:9"`、`"16:9"`、`"4:3"`、`"1:1"`、`"3:4"`、`"9:16"` | `"16:9"` |
| `generate_audio` | `true`、`false`、`"true"`、`"false"` | `true` |
| `watermark` | `true`、`false`、`"true"`、`"false"` | `false` |
| `resolution` | `"480p"`、`"720p"`、`"1080p"` | `"480p"` |

参考素材要求：

- `files` 中的图片、视频、音频都必须是公网可访问 URL。
- `files` 可以传单个 URL 字符串，也可以传 URL 数组。
- 除 `files` 外，也兼容 `images`、`image`、`input_reference`、`input_video`、`video_url`、`reference_video`、`audio`、`audios`。
- 上述参考素材字段都支持单个 URL 字符串或 URL 数组；多个字段同时传入时会合并转发。
- 上游模型通常支持最多 9 张图片、3 个视频、3 个音频；`doubao-seedance-2-0-260128-2` / `doubao-seedance-2-0-260128-3` 支持 3 个音视频参考，单个音视频需大于 4 秒，总秒数小于 15 秒。
- 不支持最下游客户端直接上传本地文件；请先上传到公网 URL 后再传入。

## 3. 支持模型

推荐模型：

```text
doubao-seedance-2-0-260128-2
doubao-seedance-2-0-260128-3
```

其他模型暂未上线：

```text
doubao-seedance-2-0-fast-260128
doubao-seedance-2-0-260128
doubao-seedance-2-0-260128-1
```

模型说明：

| 模型 ID | 上线状态 | 上游说明 |
|---|---|---|
| `doubao-seedance-2-0-260128-2` | 已上线 | 快速版；最多 9 图、3 音视频参考；单个音视频大于 4 秒，总秒数小于 15 秒；预计 3-8 分钟 |
| `doubao-seedance-2-0-260128-3` | 已上线 | 满血版；最多 9 图、3 音视频参考；单个音视频大于 4 秒，总秒数小于 15 秒；预计 8-10 分钟 |
| `doubao-seedance-2-0-fast-260128` | 暂未上线 | 快速版；最多 9 图、3 视频、3 音频；预计 3-5 分钟 |
| `doubao-seedance-2-0-260128` | 暂未上线 | 满血版；最多 9 图、3 视频、3 音频；预计 5-10 分钟 |
| `doubao-seedance-2-0-260128-1` | 暂未上线 | 满血版；最多 9 图、3 视频、3 音频；预计 20-50 分钟 |

## 4. 参数默认值

| 字段 | 默认值 |
|---|---|
| `duration` | `"4"` |
| `seconds` | 未传 `duration` 时按 `duration` 处理 |
| `aspect_ratio` | `"16:9"` |
| `generate_audio` | `true` |
| `watermark` | `false` |
| `resolution` | `"480p"` |

建议客户端显式传入 `duration` 和 `aspect_ratio`，避免不同客户端侧默认值不一致。
如果同时传入 `duration` 和 `seconds`，以 `duration` 为准。

## 5. 请求示例

### 5.1 文生视频

```bash
curl -X POST 'https://ay.light-ai.cloud/v1/videos' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "doubao-seedance-2-0-260128-2",
    "prompt": "A cinematic shot of a banana on a clean table, slow camera movement, soft studio light",
    "duration": "4",
    "aspect_ratio": "16:9",
    "generate_audio": true,
    "resolution": "480p"
  }'
```

### 5.2 参考图片生成视频

```bash
curl -X POST 'https://ay.light-ai.cloud/v1/videos' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "doubao-seedance-2-0-260128-2",
    "prompt": "Make the object in the reference image move naturally, cinematic lighting",
    "duration": "4",
    "aspect_ratio": "9:16",
    "files": [
      "https://cdn.example.com/ref-1.jpg"
    ],
    "generate_audio": true,
    "resolution": "480p"
  }'
```

### 5.3 多参考素材生成视频

```bash
curl -X POST 'https://ay.light-ai.cloud/v1/videos' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "doubao-seedance-2-0-260128-2",
    "prompt": "Use the visual reference and audio rhythm to create a short dynamic video",
    "duration": "4",
    "aspect_ratio": "16:9",
    "files": [
      "https://cdn.example.com/ref-image.jpg",
      "https://cdn.example.com/ref-audio.mp3"
    ],
    "generate_audio": true,
    "resolution": "480p"
  }'
```

### 5.4 使用兼容参考字段

```bash
curl -X POST 'https://ay.light-ai.cloud/v1/videos' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "doubao-seedance-2-0-260128-2",
    "prompt": "Use all reference materials to create a short dynamic video",
    "seconds": "4",
    "aspect_ratio": "16:9",
    "images": [
      "https://cdn.example.com/ref-1.jpg",
      "https://cdn.example.com/ref-2.png"
    ],
    "input_video": [
      "https://cdn.example.com/ref-video-1.mp4",
      "https://cdn.example.com/ref-video-2.mp4"
    ],
    "audio": [
      "https://cdn.example.com/ref-audio-1.mp3",
      "https://cdn.example.com/ref-audio-2.mp3"
    ],
    "generate_audio": true,
    "resolution": "480p"
  }'
```

### 5.5 参考视频生成视频

```bash
curl -X POST 'https://ay.light-ai.cloud/v1/videos' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>' \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "doubao-seedance-2-0-260128-2",
    "prompt": "Use the reference video motion style and generate a new short video",
    "duration": "4",
    "aspect_ratio": "16:9",
    "files": [
      "https://cdn.example.com/reference.mp4"
    ],
    "generate_audio": true,
    "resolution": "480p"
  }'
```

## 6. 创建任务响应

创建成功后返回任务信息。

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "doubao-seedance-2-0-260128-2",
  "status": "queued",
  "progress": 0,
  "created_at": 1780280676
}
```

字段说明：

| 字段 | 类型 | 说明 |
|---|---:|---|
| `id` | string | 任务 ID。通过 NewAPI 调用时通常是 `task_*` 形式 |
| `task_id` | string | 任务 ID，通常和 `id` 一致 |
| `object` | string | 固定为 `video` |
| `model` | string | 实际上游模型名 |
| `status` | string | 当前任务状态 |
| `progress` | number | 当前进度 |
| `created_at` | number | Unix 秒级时间戳 |

查询时必须使用创建响应里的 `id` 或 `task_id` 原值，不要自行改成上游数字 ID。

## 7. 查询视频任务

### 7.1 请求端点

```http
GET /v1/videos/{task_id} HTTP/1.1
Host: ay.light-ai.cloud
Authorization: Bearer <NEWAPI_API_KEY>
```

完整 URL：

```text
https://ay.light-ai.cloud/v1/videos/{task_id}
```

查询示例：

```bash
curl 'https://ay.light-ai.cloud/v1/videos/task_xxxxxxxxxxxxxxxxxxxxx' \
  -H 'Authorization: Bearer <NEWAPI_API_KEY>'
```

### 7.2 排队中响应

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "status": "queued",
  "progress": 0,
  "created_at": 1780280676
}
```

### 7.3 生成中响应

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "status": "in_progress",
  "progress": 50,
  "created_at": 1780280676,
  "metadata": {
    "seedance": {
      "Status": 1,
      "StatusText": "处理中"
    }
  }
}
```

### 7.4 已完成响应

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "doubao-seedance-2-0-260128-2",
  "status": "completed",
  "progress": 100,
  "created_at": 1780280676,
  "updated_at": 1780280736,
  "url": "https://cdn.example.com/result.mp4",
  "video_url": "https://cdn.example.com/result.mp4",
  "metadata": {
    "seedance": {
      "Status": 2,
      "StatusText": "已完成",
      "UseToken": 12,
      "DeductToken": 1,
      "UseDuration": 4
    }
  }
}
```

完成后视频地址会同时出现在：

- `url`
- `video_url`

客户端读取任意一个即可，建议优先读取 `video_url`，没有时回退到 `url`。

### 7.5 失败响应

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxxxxxxx",
  "object": "video",
  "status": "failed",
  "progress": 100,
  "created_at": 1780280676,
  "error": {
    "code": "3",
    "message": "token不足"
  },
  "metadata": {
    "seedance": {
      "Status": 3,
      "StatusText": "处理失败",
      "Message": "token不足"
    }
  }
}
```

失败时优先展示：

1. `error.message`
2. `metadata.seedance.Message`
3. `metadata.seedance.StatusText`

## 8. 状态说明

| `status` | 说明 |
|---|---|
| `queued` | 任务已进入队列 |
| `in_progress` | 任务生成中 |
| `completed` | 任务已完成，可读取 `url` 或 `video_url` |
| `failed` | 任务失败，查看 `error.message` 或 `metadata.seedance.Message` |

建议轮询间隔：5-10 秒。

## 9. 最小接入清单

1. 配置 API Base URL：`https://ay.light-ai.cloud`
2. 配置请求头：`Authorization: Bearer <NEWAPI_API_KEY>`
3. 创建任务使用 `POST /v1/videos`
4. 查询任务使用 `GET /v1/videos/{task_id}`
5. 模型优先使用 `doubao-seedance-2-0-260128-2`
6. 时长优先传 `duration: "4"`；也支持 `seconds: "4"`
7. 比例传 `aspect_ratio: "16:9"` 或 `aspect_ratio: "9:16"`
8. 参考素材优先统一传 `files`；兼容 `images`、`input_video`、`audio` 等字段
9. 完成后从 `video_url` 或 `url` 读取视频地址

## 10. 调用注意事项

- 默认时长为 4 秒，默认生成音频，默认清晰度为 `480p`。
- `files` 里的图片、视频、音频必须是公网 URL。
- 不要把本地文件路径、内网 URL、需要登录态的 URL 作为参考素材。
- NewAPI 返回的 `task_*` 是客户端应保存和查询的任务 ID。
- `metadata.seedance` 是排障信息，不建议作为主业务字段依赖。
- 如果 `status="failed"` 且 `error.message` 是余额、token 或账号类错误，说明请求已到达上游，但上游账号不可用或余额不足。
