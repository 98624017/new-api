# W Project HOLO API Reference
# W Project HOLO API 接口文档

**Version / 版本:** v1.7.1 (2026-05-24)
**Base URL / 基础地址:** `https://api.dealonhorizon.us`

---

## Authentication / 认证

All requests require an API key via header:
所有请求需要通过请求头传递 API Key：

```
Authorization: Bearer YOUR_API_KEY
```
or / 或
```
X-API-Key: YOUR_API_KEY
```

---

## Quick Guide / 快速选择

| Task / 任务 | Endpoint / 接口 | Note / 说明 |
|---|---|---|
| Generate 1 image / 生成单张图片 | `POST /v1/generate` | Async, poll for result / 异步，轮询结果 |
| Reference-to-Image / 参考生图 | `POST /v1/generate` | Same model + image_url → auto R2I / 同模型名+图片自动识别 |
| Generate 1 video / 生成单个视频 | `POST /v1/generate` | Same endpoint / 同一接口 |
| Check result / 查询结果 | `GET /v1/tasks/{id}` | Poll every 5-10s / 每5-10秒轮询 |
| Download file / 下载文件 | `GET /v1/tasks/{id}/file` | 24h retention / 保留24小时 |
| List tasks / 任务列表 | `GET /v1/tasks` | Filter by status / 按状态过滤 |
| Account info / 账户信息 | `GET /me` | Balance, usage / 余额和使用量 |

> **All generation (image + video) uses `POST /v1/generate`**, one request per image or video.
>
> **所有生成（图片+视频）统一使用 `POST /v1/generate`**，每次生成一张图片或一个视频。

---

## Endpoints / 接口列表

### 1. Submit Generation Task / 提交生成任务

`POST /v1/generate`

Submit an image or video generation task. Returns immediately with a task ID.
提交图片或视频生成任务，立即返回 task_id，通过轮询获取结果。

**Request — Text-to-Image / 文字生图:**
```json
{
  "model": "gemini-3.0-pro-image-landscape",
  "messages": [
    {"role": "user", "content": "A blue butterfly on a flower"}
  ]
}
```

**Request — Text-to-Image Square / 文字生图 方形:**
```json
{
  "model": "gemini-3.1-flash-image-square",
  "messages": [
    {"role": "user", "content": "A minimalist logo design with geometric shapes"}
  ]
}
```

**Request — Text-to-Image 4:3 / 文字生图 4:3:**
```json
{
  "model": "gemini-3.0-pro-image-four-three-2k",
  "messages": [
    {"role": "user", "content": "A cinematic still of a cyberpunk city at night"}
  ]
}
```

**Request — Reference-to-Image / 参考生图 (R2I):**

> R2I uses the same image model name — when `image_url` is included in the request, it is automatically detected as R2I and billed at R2I pricing.
>
> R2I 使用同样的图片模型名 — 请求中包含 `image_url` 时自动识别为 R2I，按 R2I 价格计费。

```json
{
  "model": "gemini-3.0-pro-image-landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/reference.jpg"}},
      {"type": "text", "text": "Generate a similar image with autumn colors"}
    ]}
  ]
}
```

**Request — Reference-to-Image 2K / 参考生图 2K:**
```json
{
  "model": "gemini-3.1-flash-image-square-2k",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/style-ref.jpg"}},
      {"type": "text", "text": "Recreate this composition with a watercolor painting style"}
    ]}
  ]
}
```

**Request — Reference-to-Image 4K / 参考生图 4K:**
```json
{
  "model": "gemini-3.0-pro-image-portrait-4k",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQ..."}},
      {"type": "text", "text": "A portrait in the same style but with different lighting"}
    ]}
  ]
}
```

**Request — GPT-images2 (Text-to-Image) / GPT-images2 文生图:**
```json
{
  "model": "GPT-images2 16:9-4K",
  "messages": [
    {"role": "user", "content": "A cinematic wide shot of a cyberpunk city at dusk"}
  ]
}
```

> Model name contains a space — pass it verbatim. Use `GPT-images2` for default (1024×1024), or `GPT-images2 {variant}` to choose a specific size variant (see model table below).
> 模型名包含空格，请原样传递。`GPT-images2` 默认 1024×1024，`GPT-images2 {variant}` 指定具体尺寸（见下方模型表）。

**Request — GPT-images2 (Reference-to-Image) / GPT-images2 参考生图:**
```json
{
  "model": "GPT-images2 1:1",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/reference.jpg"}},
      {"type": "text", "text": "Turn this into a vintage comic book style illustration"}
    ]}
  ]
}
```

**Request — Text-to-Video / 文字转视频:**
```json
{
  "model": "veo_3_1_t2v_fast_landscape",
  "messages": [
    {"role": "user", "content": "A drone shot flying over a tropical island"}
  ]
}
```

**Request — Image-to-Video / 图片转视频 (I2V):**
```json
{
  "model": "veo_3_1_i2v_fast_landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}},
      {"type": "text", "text": "Slowly zoom in with gentle camera movement"}
    ]}
  ]
}
```

> **First + Last Frame Mode / 首尾帧模式 (auto / 自动):**
> When you send **2 image_urls** with an i2v model, the system automatically switches to first-last-frame video generation — the first image is the starting frame, the second is the ending frame, and the video is interpolated between them. **No special model name needed**, just pass two images to any i2v model.
> 当你给 i2v 模型传 **2 张 image_url** 时，系统自动切换到首尾帧视频生成 — 第一张作为起始帧，第二张作为结束帧，中间动画自动插值生成。**不需要切换模型名**，给任何 i2v 模型传 2 张图即可。

**Request — First + Last Frame I2V / 首尾帧图片转视频 (i2v + 2 张图自动触发):**
```json
{
  "model": "veo_3_1_i2v_fast_portrait",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/start.jpg"}},
      {"type": "image_url", "image_url": {"url": "https://example.com/end.jpg"}},
      {"type": "text", "text": "Smooth morph transition between the two frames"}
    ]}
  ]
}
```

**Request — Text-to-Video Quality / 文字转视频 高质量:**
```json
{
  "model": "veo_3_1_t2v_landscape",
  "messages": [
    {"role": "user", "content": "A cinematic sunset over the ocean with volumetric clouds"}
  ]
}
```

**Request — Text-to-Video Quality 1080p / 文字转视频 高质量 1080p:**
```json
{
  "model": "veo_3_1_t2v_landscape_1080p",
  "messages": [
    {"role": "user", "content": "A cinematic aerial shot of a mountain range at golden hour"}
  ]
}
```

**Request — Text-to-Video Quality 4K / 文字转视频 高质量 4K:**
```json
{
  "model": "veo_3_1_t2v_portrait_4k",
  "messages": [
    {"role": "user", "content": "A vertical cinematic shot of a waterfall in a lush forest"}
  ]
}
```

**Request — Image-to-Video Quality / 图片转视频 高质量:**
```json
{
  "model": "veo_3_1_i2v_s_landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}},
      {"type": "text", "text": "Gentle parallax movement with cinematic depth of field"}
    ]}
  ]
}
```

**Request — Image-to-Video Quality 1080p / 图片转视频 高质量 1080p:**
```json
{
  "model": "veo_3_1_i2v_s_landscape_1080p",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/start.jpg"}},
      {"type": "text", "text": "Cinematic dolly zoom with bokeh background"}
    ]}
  ]
}
```

**Request — Image-to-Video Quality 4K (first + last frame) / 图片转视频 高质量 4K (首尾帧):**
```json
{
  "model": "veo_3_1_i2v_s_portrait_4k",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/start.jpg"}},
      {"type": "image_url", "image_url": {"url": "https://example.com/end.jpg"}},
      {"type": "text", "text": "Character turns around slowly, background transitions to night"}
    ]}
  ]
}
```

> 2 image_urls 自动触发首尾帧,1 张图即标准 i2v / Two image_urls trigger first-last-frame mode automatically; one image is standard i2v.

**Request — Reference-to-Video / 参考转视频 (R2V):**

> R2V (`veo_3_1_r2v_*`) uses up to 3 images as **style references** (not start/end frames). The video is generated freshly with the references guiding overall look. This is different from i2v + 2 images (first-last-frame mode above).
> R2V (`veo_3_1_r2v_*`) 把图作为**风格参考**（最多 3 张），视频从头生成，参考图只引导整体风格。这与 i2v + 2 张图（首尾帧）不同。

```json
{
  "model": "veo_3_1_r2v_fast_landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/ref1.jpg"}},
      {"type": "image_url", "image_url": {"url": "https://example.com/ref2.jpg"}},
      {"type": "text", "text": "Generate a video using these reference images as style guide"}
    ]}
  ]
}
```

**Request — Text-to-Video Lite / 文字转视频 轻量:**
```json
{
  "model": "veo_3_1_t2v_lite_landscape",
  "messages": [
    {"role": "user", "content": "A cat walking through a sunny garden"}
  ]
}
```

**Request — Image-to-Video Lite / 图片转视频 轻量:**
```json
{
  "model": "veo_3_1_i2v_lite_portrait",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}},
      {"type": "text", "text": "Animate with gentle floating motion"}
    ]}
  ]
}
```

> Lite models are faster and cheaper than Fast, ideal for previews and high-volume use.
> Lite 模型比 Fast 更快更便宜，适合预览和大批量使用。

**Request — Text-to-Image 2K / 文字生图 2K:**
```json
{
  "model": "gemini-3.0-pro-image-landscape-2k",
  "messages": [
    {"role": "user", "content": "A beautiful sunset over mountains in high detail"}
  ]
}
```

**Request — Text-to-Image 4K / 文字生图 4K:**
```json
{
  "model": "gemini-3.0-pro-image-portrait-4k",
  "messages": [
    {"role": "user", "content": "A portrait of a woman in oil painting style, ultra detailed"}
  ]
}
```

**Request — Text-to-Video 1080p / 文字转视频 1080p:**
```json
{
  "model": "veo_3_1_t2v_fast_landscape_1080p",
  "messages": [
    {"role": "user", "content": "A timelapse of clouds moving over a city skyline"}
  ]
}
```

**Request — Text-to-Video 4K / 文字转视频 4K:**
```json
{
  "model": "veo_3_1_t2v_fast_portrait_4k",
  "messages": [
    {"role": "user", "content": "A vertical video of rain falling on a window"}
  ]
}
```

**Request — I2V 1080p / 图片转视频 1080p:**
```json
{
  "model": "veo_3_1_i2v_fast_landscape_1080p",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/start.jpg"}},
      {"type": "text", "text": "Slowly zoom in with cinematic depth of field"}
    ]}
  ]
}
```

**Request — I2V 4K / 图片转视频 4K:**
```json
{
  "model": "veo_3_1_i2v_fast_portrait_4k",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4AAQ..."}},
      {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,/9j/4BBR..."}},
      {"type": "text", "text": "Character slowly turns around, background blurs"}
    ]}
  ]
}
```

**Request — R2V 1080p / 参考转视频 1080p:**
```json
{
  "model": "veo_3_1_r2v_fast_landscape_1080p",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/ref.jpg"}},
      {"type": "text", "text": "Create a cinematic video based on this reference"}
    ]}
  ]
}
```

**Request — R2V 4K / 参考转视频 4K:**
```json
{
  "model": "veo_3_1_r2v_fast_portrait_4k",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/ref1.jpg"}},
      {"type": "image_url", "image_url": {"url": "https://example.com/ref2.jpg"}},
      {"type": "image_url", "image_url": {"url": "https://example.com/ref3.jpg"}},
      {"type": "text", "text": "Generate a vertical 4K video using these references as style guide"}
    ]}
  ]
}
```

> Quality models produce higher fidelity output but take longer to generate.
> Quality 模型生成质量更高，但耗时更长。

---

#### Duration Variants & R2V Lite / 时长选项 + R2V 轻量

**Request — T2V 4 seconds Fast / 文字转视频 4 秒 快速:**
```json
{
  "model": "veo_3_1_t2v_fast_4s_landscape",
  "messages": [
    {"role": "user", "content": "A drone flyover of a tropical island"}
  ]
}
```

**Request — T2V 6 seconds Quality / 文字转视频 6 秒 高质量:**
```json
{
  "model": "veo_3_1_t2v_quality_6s_portrait",
  "messages": [
    {"role": "user", "content": "Cinematic close-up of raindrops falling on glass"}
  ]
}
```

**Request — T2V 4 seconds Lite / 文字转视频 4 秒 轻量:**
```json
{
  "model": "veo_3_1_t2v_lite_4s_landscape",
  "messages": [
    {"role": "user", "content": "Quick preview: a bird flying across a blue sky"}
  ]
}
```

**Request — T2V 6 seconds Lite / 文字转视频 6 秒 轻量 (highest-volume preview use case / 大批量预览常用):**
```json
{
  "model": "veo_3_1_t2v_lite_6s_portrait",
  "messages": [
    {"role": "user", "content": "Vertical preview: ocean waves at sunset"}
  ]
}
```

**Request — I2V 4 seconds Fast / 图片转视频 4 秒 快速:**
```json
{
  "model": "veo_3_1_i2v_fast_4s_portrait",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/portrait.jpg"}},
      {"type": "text", "text": "Subtle head turn with hair flowing"}
    ]}
  ]
}
```

**Request — I2V 6 seconds Quality / 图片转视频 6 秒 高质量:**
```json
{
  "model": "veo_3_1_i2v_quality_6s_landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/scene.jpg"}},
      {"type": "text", "text": "Camera slowly pulls back revealing the wider scene"}
    ]}
  ]
}
```

**Request — R2V Lite / 参考转视频 轻量 (新档, 720p only, 8s):**
```json
{
  "model": "veo_3_1_r2v_lite_landscape",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/ref.jpg"}},
      {"type": "text", "text": "Cinematic motion based on this reference"}
    ]}
  ]
}
```

**Request — Sora-2 12 seconds (Text-to-Video) / Sora-2 12 秒 文字转视频:**
```json
{
  "model": "Sora-2-12",
  "size": "1280x720",
  "seconds": "12",
  "messages": [
    {"role": "user", "content": "a small bird flying over a green meadow"}
  ]
}
```

**Request — Sora-2 (Image-to-Video, optional reference) / Sora-2 图片转视频 (可选参考图):**
```json
{
  "model": "Sora-2-12",
  "size": "1280x720",
  "seconds": "12",
  "messages": [
    {"role": "user", "content": [
      {"type": "image_url", "image_url": {"url": "https://example.com/scene.jpg"}},
      {"type": "text", "text": "Animate this scene with gentle camera motion"}
    ]}
  ]
}
```

> **Sora-2 注意事项 / Notes:**
> - Model name **case-sensitive**:`Sora-2-12`(严格大小写)
> - `seconds` 必须是**字符串** `"12"`
> - `size` is **required** in the request body. Only `1280x720` (landscape) and `720x1280` (portrait) are accepted; other values return 400.
>   `size` 字段必填，只支持 `1280x720`（横屏）或 `720x1280`（竖屏），其他值会返回 400。
> - Reference image is optional (max 1 image). With image → image-to-video; without → text-to-video.
>   参考图可选（最多 1 张），有图为图生视频，无图为文生视频。
> - Generation typically takes 2–5 minutes per video.
>   单条视频生成通常需 2–5 分钟。

> **Tier Choice Guide / 档位选择建议：**
> - **Lite / 轻量**: lowest cost, may take longer to start — ideal for previews and batch jobs / 最低价，启动时间可能略长，适合预览和批量任务
> - **Fast / 快速**: balanced speed and quality, default for most use cases / 速度与质量平衡，多数场景默认
> - **Quality / 高质量**: highest fidelity, longest wait / 最高画质，耗时最长

Both formats are supported for R2I/I2V/R2V images:
R2I/I2V/R2V 图片支持两种格式：
- **URL**: `"url": "https://example.com/image.jpg"` — auto downloaded / 自动下载转换
- **Base64**: `"url": "data:image/jpeg;base64,/9j/4AAQ..."` — sent directly / 直接发送

Supported image formats / 支持格式: JPEG, PNG, WebP

**Response / 响应 (202 Accepted):**
```json
{
  "task_id": "abc123def456",
  "status": "queued",
  "position": 12,
  "cost": 12,
  "model": "gemini-3.0-pro-image-landscape",
  "created_at": "2026-03-26T12:00:00+00:00"
}
```

| Field / 字段 | Description / 说明 |
|---|---|
| `task_id` | Unique task identifier / 任务唯一标识 |
| `status` | `queued` = waiting in queue / 排队中 |
| `position` | Queue position / 队列位置 |
| `cost` | Credits deducted / 扣除积分数 |

**Error Responses / 错误响应:**
| Status / 状态码 | Meaning / 含义 |
|--------|---------|
| 401 | Missing or invalid API key / API Key 缺失或无效 |
| 400 | Invalid model, bad JSON, or image download failed / 无效模型、JSON格式错误或图片下载失败 |
| 402 | Insufficient credits / 积分不足 |
| 429 | Rate limit or daily limit exceeded / 频率限制或每日限额已达 |
| 503 | Service at capacity or paused / 服务繁忙或暂停 |

---

### 2. Query Task Status / 查询任务状态

`GET /v1/tasks/{task_id}`

Poll this endpoint to track task progress. Recommended interval: 5-10 seconds.
轮询此接口获取任务进度，建议间隔 5-10 秒。

**Response — Queued / 排队中:**
```json
{
  "task_id": "abc123",
  "status": "queued",
  "position": 8,
  "model": "gemini-3.0-pro-image-landscape",
  "cost": 12,
  "created_at": "2026-03-26 12:00:00"
}
```

**Response — Processing / 处理中:**
```json
{
  "task_id": "abc123",
  "status": "processing",
  "model": "gemini-3.0-pro-image-landscape",
  "cost": 12,
  "created_at": "2026-03-26 12:00:00",
  "started_at": "2026-03-26 12:00:30"
}
```

**Response — Completed / 已完成:**
```json
{
  "task_id": "abc123",
  "status": "completed",
  "model": "gemini-3.0-pro-image-landscape",
  "task_type": "t2i",
  "cost": 12,
  "created_at": "2026-03-26 12:00:00",
  "completed_at": "2026-03-26 12:01:05",
  "expires_at": "2026-03-28 12:01:05",
  "result": {
    "file_url": "/v1/tasks/abc123/file",
    "file_ext": "png",
    "file_size": 1234567,
    "duration_ms": 45000,
    "type": "t2i"
  }
}
```

| Field / 字段 | Description / 说明 |
|---|---|
| `file_url` | Authenticated relative path (requires API Key) / 鉴权相对路径,需 API Key,详见下方两种下载方式 |
| `file_ext` | File extension: png, jpg, mp4 / 文件扩展名 |
| `file_size` | File size in bytes / 文件大小（字节） |
| `duration_ms` | Generation time in ms / 生成耗时（毫秒） |
| `type` | Task type: t2i, r2i, t2v, i2v, r2v / 任务类型 |
| `expires_at` | File expiry time (24h retention) / 文件过期时间（保留24小时） |

> **下载地址有两种 / Two download URL forms — pick the one that fits:**
>
> **方式 1 / Option 1:鉴权下载(`file_url` 拼 Base URL)** — 只有持 API Key 的客户能下,任何人无 key 拿到 URL 也无效。Authenticated, requires API Key. Anyone without the key gets 401.
>
> ```
> 完整下载 URL = https://api.dealonhorizon.us + file_url
>              = https://api.dealonhorizon.us/v1/tasks/abc123/file
> ```
>
> ```bash
> curl -o result.mp4 -H "X-API-Key: YOUR_API_KEY" \
>   "https://api.dealonhorizon.us/v1/tasks/abc123/file"
> ```
>
> **方式 2 / Option 2:公开直链(无需 API Key,任何人可下,可嵌入 `<video>`/`<img>` / shareable, no auth required)** — Cloudflare CDN 公开存储桶,**全球加速、支持 Range 断点续传、24h 保留**。Public Cloudflare R2 CDN with range-request support, retained 24h.
>
> ```
> 公开直链 = https://media.dealonhorizon.us/{task_id}.{file_ext}
>          = https://media.dealonhorizon.us/abc123.mp4
> ```
>
> 拼法:`task_id` 来自任务响应,`file_ext` 来自 `result.file_ext`(`mp4` / `png` / `jpg` / `webp`)。
> Construct by combining `task_id` (from task response) with `result.file_ext`.
>
> ```bash
> # 直链分享 / Shareable direct link
> curl -o result.mp4 "https://media.dealonhorizon.us/abc123.mp4"
> # 浏览器直接打开亦可,可嵌入网页 <video src="..."> / Works in browser, embeddable
> ```
>
> ⚠️ 方式 2 任何人拿到 URL 都能下,**不要分享给不信任的人**。需要私密的请用方式 1。
> Option 2 is fully public — don't share if the content is sensitive. Use Option 1 for private/access-controlled downloads.

**Response — Failed / 失败:**
```json
{
  "task_id": "abc123",
  "status": "failed",
  "error": "Content policy violation",
  "refunded": true
}
```

All failed tasks are automatically refunded. / 所有失败任务自动退还积分。

**Response — Cancelled / 已取消:**
```json
{
  "task_id": "abc123",
  "status": "cancelled",
  "refunded": true
}
```

---

### 3. Download Result File / 下载结果文件

`GET /v1/tasks/{task_id}/file`

Download the generated image or video file. Only available for `completed` tasks.
下载生成的图片或视频文件，仅对已完成的任务有效。

**Response:** Binary file with appropriate `Content-Type` header.
- Images / 图片: `image/png`, `image/jpeg`, `image/webp`
- Videos / 视频: `video/mp4`

Files are retained for **24 hours** after completion.
文件在完成后保留 **24 小时**。

---

### 4. List Tasks / 任务列表

`GET /v1/tasks`

List your generation tasks with optional filtering.
查询自己的生成任务列表，支持过滤和分页。

**Query Parameters / 查询参数:**
| Param / 参数 | Type / 类型 | Description / 说明 |
|-------|------|-------------|
| `status` | string | Filter / 过滤: `queued`, `processing`, `completed`, `failed`, `cancelled` |
| `limit` | int | Max results, default 50, max 200 / 最大结果数 |
| `offset` | int | Pagination offset / 分页偏移 |

**Response / 响应:**
```json
{
  "tasks": [
    {
      "task_id": "abc",
      "model": "gemini-3.0-pro-image-landscape",
      "task_type": "t2i",
      "status": "completed",
      "cost": 12,
      "created_at": "2026-03-26 12:00:00",
      "completed_at": "2026-03-26 12:01:05"
    }
  ],
  "total": 150,
  "queued": 3,
  "processing": 1,
  "completed": 140,
  "failed": 6
}
```

---

### 5. Cancel Task / 取消任务

`DELETE /v1/tasks/{task_id}`

Cancel a queued task. Credits are refunded immediately. Only `queued` tasks can be cancelled.
取消排队中的任务，积分立即退还。仅 queued 状态可取消。

**Response / 响应:**
```json
{
  "ok": true,
  "task_id": "abc123",
  "refunded": 12
}
```

---

### 6. Account Info / 账户信息

`GET /me`

Get your current key's balance and usage stats.
查询当前 key 的余额和使用量。

**Response / 响应:**
```json
{
  "id": 8,
  "name": "MyKey",
  "credits": 29800.0,
  "frozen_credits": 0.0,
  "img_30d": 150,
  "vid_30d": 45,
  "today_img": 12,
  "today_vid": 3,
  "daily_used": 15,
  "daily_credits_used": 180.0,
  "daily_limit": 0,
  "rpm_limit": 0,
  "account_id": 7,
  "key_label": "prod-1",
  "account_name": "MyCompany",
  "tier_thresholds": [...],
  "effective_pricing": { "...": "see /me response for your personal pricing" }
}
```

| Field / 字段 | Description / 说明 |
|---|---|
| `credits` | Current key balance / 当前 key 余额 |
| `frozen_credits` | Credits reserved for in-progress tasks / 进行中任务冻结的积分 |
| `img_30d` / `vid_30d` | 30-day rolling count (affects pricing tier) / 30天滚动量（影响定价等级） |
| `today_img` / `today_vid` | Today's generation count / 今日生成量 |
| `daily_used` | Today's total requests / 今日总请求数 |
| `daily_credits_used` | Today's total credits consumed / 今日消耗积分 |
| `daily_limit` | Daily request limit (0 = unlimited) / 每日请求限制（0=无限） |
| `rpm_limit` | Requests per minute limit (0 = unlimited) / 每分钟请求限制（0=无限） |
| `tier_thresholds` | Volume tier boundaries / 阶梯用量分界线 |
| `effective_pricing` | Your personalized per-model pricing / 您的专属每模型定价 |

> Per-model pricing is returned by `GET /me` in the `effective_pricing` field. Refer to that response for your account's current pricing.
>
> 各模型实时定价由 `GET /me` 接口的 `effective_pricing` 字段返回，请以该响应为准。

---

### 7. Account Management / 账户管理（多 Key）

> These endpoints require **account password login** via the Dashboard. API key login only provides single-key view.
>
> 这些接口需要通过 Dashboard **使用账户密码登录**。API key 登录只能查看单个 key。

**Account Overview / 账户概览:** `GET /me/account`

```json
{
  "account": {
    "id": 7,
    "name": "MyCompany",
    "credit_pool": 0.0
  },
  "total_credits": 50000.0,
  "keys": [
    {"id": 10, "name": "prod-1", "key_label": "production", "credits": 30000, "is_active": true},
    {"id": 11, "name": "prod-2", "key_label": "staging", "credits": 20000, "is_active": true}
  ]
}
```

**Create Key / 创建 Key:** `POST /me/account/keys`
```json
{"name": "new-key", "label": "testing"}
```

**Delete Key / 删除 Key:** `DELETE /me/account/keys/{key_id}`

**Allocate Credits / 分配积分:** `POST /me/account/allocate`
```json
{"from_key_id": 10, "to_key_id": 11, "amount": 5000}
```

**Change Password / 修改密码:** `POST /me/account/password`
```json
{"old_password": "...", "new_password": "..."}
```

**Account Stats / 账户统计:** `GET /me/account/stats`

---

### 8. Usage History / 使用记录

`GET /me/usage`

Query usage history with optional date filter.
查询使用记录，可按日期过滤。

| Param / 参数 | Type / 类型 | Description / 说明 |
|-------|------|-------------|
| `date` | string | Filter by date / 按日期过滤: `YYYY-MM-DD` |

---

### 9. Credit History / 积分交易记录

`GET /me/transactions`

| Param / 参数 | Type / 类型 | Description / 说明 |
|-------|------|-------------|
| `limit` | int | Max results, default 50, max 500 / 最大结果数 |
| `offset` | int | Pagination offset / 分页偏移 |
| `date` | string | Filter by date / 按日期: `YYYY-MM-DD` |
| `type` | string | Filter / 过滤: `charge`(消费), `refund`(退款), `topup`(充值), `adjust`(调整) |
| `task_type` | string | Filter / 过滤: `t2i`, `r2i`, `t2v`, `i2v`, `r2v` |

---

### 10. List Models / 模型列表

`GET /v1/models`

Returns all available models.
返回所有可用模型。

---

### 11. Service Health / 服务状态

`GET /health` *(no auth required / 无需认证)*

Check if the service is online before submitting tasks.
提交任务前检查服务是否在线。

```json
{
  "service": "api",
  "status": "ok",
  "capacity": "available"
}
```

| Field / 字段 | Description / 说明 |
|---|---|
| `status` | `ok` = online / 在线 |
| `capacity` | `available` = accepting tasks / 可接受任务, `busy` = high load / 高负载 |

---

### 12. Announcements / 公告

`GET /banner` *(no auth required / 无需认证)*

Get active announcements.
获取当前有效公告。

```json
{
  "text": "System maintenance at 03:00 UTC",
  "visible": true,
  "banners": []
}
```

| Field / 字段 | Description / 说明 |
|---|---|
| `text` | Current announcement text (empty if none) / 当前公告文字（无公告时为空） |
| `visible` | Whether announcement is active / 公告是否生效 |
| `banners` | List of all active banners / 所有生效公告列表 |

---

## Available Models / 可用模型

### Image Generation / 图片生成 (Text-to-Image / 文字生图)

| Model / 模型 | Resolution / 分辨率 |
|-------|-----------|
| `gemini-3.0-pro-image-{orientation}` | Standard / 标准 |
| `gemini-3.0-pro-image-{orientation}-2k` | 2K |
| `gemini-3.0-pro-image-{orientation}-4k` | 4K |
| `gemini-3.1-flash-image-{orientation}` | Standard / 标准 |
| `gemini-3.1-flash-image-{orientation}-2k` | 2K |
| `gemini-3.1-flash-image-{orientation}-4k` | 4K |
| `imagen-4.0-generate-preview-{orientation}` | Standard / 标准 |

**Gemini orientations / Gemini 方向:** `landscape`, `portrait`, `square`, `four-three`, `three-four`

**Imagen orientations / Imagen 方向:** `landscape`, `portrait`

### Image Generation / 图片生成 (Reference-to-Image / 参考生图 R2I)

> R2I uses the same image model names as Text-to-Image. When the request contains `image_url`, it is automatically detected as R2I and billed at R2I pricing.
>
> R2I 使用与文字生图完全相同的模型名。当请求中包含 `image_url` 时，自动识别为 R2I 并按 R2I 价格计费。

| Model / 模型 | Resolution / 分辨率 |
|-------|-----------|
| `gemini-3.0-pro-image-{orientation}` + image_url | Standard / 标准 |
| `gemini-3.0-pro-image-{orientation}-2k` + image_url | 2K |
| `gemini-3.0-pro-image-{orientation}-4k` + image_url | 4K |
| `gemini-3.1-flash-image-{orientation}` + image_url | Standard / 标准 |
| `gemini-3.1-flash-image-{orientation}-2k` + image_url | 2K |
| `gemini-3.1-flash-image-{orientation}-4k` + image_url | 4K |

**Orientations / 方向:** `landscape`, `portrait`, `square`, `four-three`, `three-four`

### Image Generation / 图片生成 (GPT-images2)

> OpenAI-based `gpt-image-2` series. Supports both Text-to-Image and Reference-to-Image (same model name + `image_url` auto-detects R2I). Model name contains a space — pass it verbatim.
>
> 基于 OpenAI `gpt-image-2` 系列。同时支持文生图和参考生图（带 `image_url` 自动识别为 R2I）。模型名包含空格，请原样传递。

| Model / 模型 | Output / 实际尺寸 | Notes / 说明 |
|---|---|---|
| `GPT-images2` | 1024×1024 | Default / 默认 |
| `GPT-images2 1:1` | 1024×1024 | Square 1K / 方形 1K |
| `GPT-images2 1:1-2K` | 1920×1920 | Square 2K / 方形 2K |
| `GPT-images2 3:2-2K` | 1920×1280 | Landscape 3:2 / 横版 3:2 |
| `GPT-images2 2:3-2K` | 1280×1920 | Portrait 2:3 / 竖版 2:3 |
| `GPT-images2 16:9-2K` | 1920×1088 | Widescreen 2K / 宽屏 2K |
| `GPT-images2 16:9-4K` | 3840×2160 | Widescreen 4K / 宽屏 4K |
| `GPT-images2 9:16-4K` | 2160×3840 | Vertical 4K / 竖屏 4K |

### Video Generation / 视频生成 (Text-to-Video / 文字转视频)

**Orientations / 方向:** `landscape`, `portrait`
**Tiers / 档位:** `lite` (lowest cost, may take longer to start / 最低价，启动时间可能略长)、`fast` (balanced speed and quality / 速度与质量平衡)、`quality` (highest fidelity / 最高画质)

#### 8 seconds (default) / 8 秒（默认）— supports 1080p / 4K

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_t2v_lite_{orientation}` | Lite | 720p |
| `veo_3_1_t2v_fast_{orientation}` | Fast | 720p |
| `veo_3_1_t2v_fast_{orientation}_1080p` | Fast | 1080p |
| `veo_3_1_t2v_fast_{orientation}_4k` | Fast | 4K |
| `veo_3_1_t2v_{orientation}` | Quality | 720p |
| `veo_3_1_t2v_{orientation}_1080p` | Quality | 1080p |
| `veo_3_1_t2v_{orientation}_4k` | Quality | 4K |

#### 4 seconds / 4 秒 — 720p only

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_t2v_lite_4s_{orientation}` | Lite | 720p |
| `veo_3_1_t2v_fast_4s_{orientation}` | Fast | 720p |
| `veo_3_1_t2v_quality_4s_{orientation}` | Quality | 720p |

#### 6 seconds / 6 秒 — 720p only

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_t2v_lite_6s_{orientation}` | Lite | 720p |
| `veo_3_1_t2v_fast_6s_{orientation}` | Fast | 720p |
| `veo_3_1_t2v_quality_6s_{orientation}` | Quality | 720p |

---

### Video Generation / 视频生成 (Image-to-Video / 图片转视频)

#### 8 seconds (default) / 8 秒（默认）— supports 1080p / 4K

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_i2v_lite_{orientation}` | Lite | 720p |
| `veo_3_1_i2v_fast_{orientation}` | Fast | 720p |
| `veo_3_1_i2v_fast_{orientation}_1080p` | Fast | 1080p |
| `veo_3_1_i2v_fast_{orientation}_4k` | Fast | 4K |
| `veo_3_1_i2v_s_{orientation}` | Quality | 720p |
| `veo_3_1_i2v_s_{orientation}_1080p` | Quality | 1080p |
| `veo_3_1_i2v_s_{orientation}_4k` | Quality | 4K |

#### 4 seconds / 4 秒 — 720p only

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_i2v_lite_4s_{orientation}` | Lite | 720p |
| `veo_3_1_i2v_fast_4s_{orientation}` | Fast | 720p |
| `veo_3_1_i2v_quality_4s_{orientation}` | Quality | 720p |

#### 6 seconds / 6 秒 — 720p only

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_i2v_lite_6s_{orientation}` | Lite | 720p |
| `veo_3_1_i2v_fast_6s_{orientation}` | Fast | 720p |
| `veo_3_1_i2v_quality_6s_{orientation}` | Quality | 720p |

---

### Video Generation / 视频生成 (Reference-to-Video / 参考转视频)

> **Note:** R2V supports only 8s duration. R2V Quality tier is not available.
> **注意：** R2V 仅支持 8 秒时长。R2V 不提供 Quality 档。

| Model / 模型 | Tier / 档位 | Resolution / 分辨率 |
|-------|------|-----------|
| `veo_3_1_r2v_lite_{orientation}` | Lite | 720p |
| `veo_3_1_r2v_fast_{orientation}` | Fast | 720p |
| `veo_3_1_r2v_fast_{orientation}_1080p` | Fast | 1080p |
| `veo_3_1_r2v_fast_{orientation}_4k` | Fast | 4K |

---

### Video Generation / 视频生成 (Sora-2)

> OpenAI Sora-2 model,通过统一入口 `POST /v1/generate` 提交 (同 API Key、同任务轮询协议)。**`size` 字段必填**。参考图可选(最多 1 张,messages 里的 `image_url`),无图为文生视频,有图为图生视频。
>
> Submit via the same `POST /v1/generate` endpoint as veo/grok. **`size` is required.** Optional reference image (max 1, via `image_url` in messages); without image = text-to-video, with image = image-to-video.

| Model / 模型 | Duration / 时长 | Sizes / 支持尺寸 |
|---|---|---|
| `Sora-2-12` | 12 秒 | `1280x720`, `720x1280` |

**注意 / Notes:**
- Model name **case-sensitive** (`Sora-2-12`,首字母大写 S)
- `seconds` 字段必须是**字符串** `"12"`
- 单次生成通常 2–5 分钟

**提交示例 / Submit example (12 秒, 竖屏):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Sora-2-12",
    "size": "720x1280",
    "seconds": "12",
    "messages": [{"role":"user","content":"a calm river at sunset"}]
  }'
```

**图生视频 / Image-to-Video (附 1 张参考图):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Sora-2-12",
    "size": "720x1280",
    "seconds": "12",
    "messages": [{"role":"user","content":[
      {"type":"image_url","image_url":{"url":"https://example.com/ref.jpg"}},
      {"type":"text","text":"the woman walks forward, camera pulls back"}
    ]}]
  }'
```

返回 task 后 `GET /v1/tasks/{id}` 轮询、`GET /v1/tasks/{id}/file` 下载 mp4 — 与 veo/grok 完全一致。
Returns a task; poll and download identically to veo/grok flow.

---

### Video Generation / 视频生成 (Grok)

> xAI Grok 视频模型,通过统一入口 `POST /v1/generate` 提交 (与 veo/sora 同 endpoint,同 API Key,同任务轮询协议)。
> Submit via the same `POST /v1/generate` endpoint as veo/sora — same API key, same task polling protocol.

| Model / 模型 | Duration / 时长 | Resolution / 分辨率 |
|---|---|---|
| `grok-imagine-video` | 默认 6s / Default 6s | 720p |
| `grok-imagine-video-6s-480p` | 6 seconds / 6 秒 | 480p |
| `grok-imagine-video-6s-720p` | 6 seconds / 6 秒 | 720p |
| `grok-imagine-video-10s-480p` | 10 seconds / 10 秒 | 480p |
| `grok-imagine-video-10s-720p` | 10 seconds / 10 秒 | 720p |
| `grok-imagine-video-12s-480p` | 12 seconds / 12 秒 | 480p |
| `grok-imagine-video-12s-720p` | 12 seconds / 12 秒 | 720p |
| `grok-imagine-video-16s-480p` | 16 seconds / 16 秒 | 480p |
| `grok-imagine-video-16s-720p` | 16 seconds / 16 秒 | 720p |
| `grok-imagine-video-20s-480p` | 20 seconds / 20 秒 | 480p |
| `grok-imagine-video-20s-720p` | 20 seconds / 20 秒 | 720p |

**模型名顺序固定:** `grok-imagine-video-{秒}s-{分辨率}p` — seconds 在前、resolution 在后,大小写敏感。
Suffix order is fixed: seconds first, resolution second, case-sensitive.

**文生视频 / Text-to-Video (t2v):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video-10s-720p",
    "prompt": "A drone shot flying over mountains at dusk"
  }'
```

**图生视频 / Image-to-Video (i2v) — 附 1 张参考图:**

支持两种参考图传法,二选一:

— 方式 A:top-level `image` 字段,base64 data URI (推荐,简洁):

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video-10s-720p",
    "prompt": "the cat slowly turns its head and blinks",
    "image": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEBLAEsAAD/..."
  }'
```

— 方式 B:OpenAI chat-completions 兼容格式 (messages 数组里塞 `image_url`):

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video-10s-720p",
    "messages": [{"role":"user","content":[
      {"type":"image_url","image_url":{"url":"data:image/jpeg;base64,/9j/4AAQSkZJRg..."}},
      {"type":"text","text":"the cat slowly turns its head and blinks"}
    ]}]
  }'
```

**i2v 注意 / Notes:**
- 只接受 **base64 data URI** (`data:image/jpeg;base64,...` / `data:image/png;base64,...`),不支持远程 http(s) URL — 客户端请自行把图片转 base64 嵌入
- Only **base64 data URI** is accepted (`data:image/...;base64,...`); remote http(s) URLs are not fetched. Encode the image client-side.
- 单次最多 1 张参考图 / Max 1 reference image per request
- t2v 和 i2v **计费一致**(同模型同时长同价),i2v 不加价 / Same price as t2v at the same SKU

**返回与轮询 / Return + Poll:** 与 veo/sora 完全一致 — 返回 `{ "task_id": "...", "status": "queued", ... }`,之后 `GET /v1/tasks/{id}` 轮询、`GET /v1/tasks/{id}/file` 下载 mp4。
Same as veo/sora — returns task, poll `GET /v1/tasks/{id}`, download via `GET /v1/tasks/{id}/file`.

> **唯一接入域名是 `api.dealonhorizon.us`**,请不要使用其他子域名(其他子域名为内部服务,不接受客户 API Key,会返 403)。
> **Use `api.dealonhorizon.us` as the only endpoint.** Other subdomains are internal services and reject customer API keys (returns 403).

---

### Video Generation / 视频生成 (Omni Flash)

> Google Omni Flash 视频模型,通过统一入口 `POST /v1/generate` 提交 (与 veo/sora/grok 同 endpoint,同 API Key,同任务轮询协议)。
> Submit via the same `POST /v1/generate` endpoint as veo/sora/grok — same API key, same task polling protocol.
>
> **关键差异 / Key difference:** 横竖屏通过 request body 的 `aspect_ratio` 字段控制,**不在模型名里**(跟 veo 不一样)。时长 4s/6s/8s/10s 可以写在模型名后缀,也可以用 body 的 `seconds` 字段。
> Orientation is controlled via the `aspect_ratio` body field — **not** in the model name (unlike veo). Duration can be either in the model name suffix (`_4s/_6s/_8s/_10s`) or via the `seconds` body field.

**Orientations / 方向:** `aspect_ratio: "16:9"` (landscape / 横屏) 或 `aspect_ratio: "9:16"` (portrait / 竖屏)
**Durations / 时长:** 4s / 6s / 8s / 10s (默认 10s)
**Resolutions / 分辨率:** 720p (默认) / 1080p / 4K

#### Text-to-Video (t2v) / 文字转视频

| Model / 模型 | Duration / 时长 | Resolution / 分辨率 |
|---|---|---|
| `omni_flash` | 10s (默认) | 720p |
| `omni_flash_1080p` | 10s (默认) | 1080p |
| `omni_flash_4k` | 10s (默认) | 4K |
| `omni_flash_4s` / `_6s` / `_8s` / `_10s` | 4s / 6s / 8s / 10s | 720p |
| `omni_flash_4s_1080p` / `_6s_1080p` / `_8s_1080p` / `_10s_1080p` | 4s / 6s / 8s / 10s | 1080p |
| `omni_flash_4s_4k` / `_6s_4k` / `_8s_4k` / `_10s_4k` | 4s / 6s / 8s / 10s | 4K |

#### Components (Reference-to-Video) / 参考转视频

支持多张参考图(messages 里多个 `image_url`),用于组合多个素材生成视频。
Supports multiple reference images for compositing.

| Model / 模型 | Duration / 时长 | Resolution / 分辨率 |
|---|---|---|
| `omni_flash_components` | 10s (默认) | 720p |
| `omni_flash_components_1080p` / `_4k` | 10s (默认) | 1080p / 4K |
| `omni_flash_components_4s` / `_6s` / `_8s` / `_10s` | 4s / 6s / 8s / 10s | 720p |
| `omni_flash_components_4s_1080p` / `_6s_1080p` / `_8s_1080p` / `_10s_1080p` | 4s / 6s / 8s / 10s | 1080p |
| `omni_flash_components_4s_4k` / `_6s_4k` / `_8s_4k` / `_10s_4k` | 4s / 6s / 8s / 10s | 4K |

#### Video Edit / 视频编辑

对已有视频做修改/再生成(改场景、改风格、改人物动作等),输入 1 段视频 + 编辑提示词。
Edits an existing video (scene change / style transfer / motion remap, etc.) — input one video URL + edit prompt.

**输入方式 / Input:** body 顶层 `input_video`(推荐)或 `video_url` 字段传源视频,提示词放 `messages` 文本。三种形态任选其一:
Source video via top-level `input_video` (preferred) or `video_url` — three forms supported:

| 形态 / Form | 用途 / Use case | 示例 / Example |
|---|---|---|
| **http(s) URL** | 公开可拉取的视频 / public URL | `"input_video": "https://your-cdn.com/src.mp4"` |
| **base64 data URI** | 小视频内联 ≤8 MB / small inline | `"input_video": "data:video/mp4;base64,AAAA..."` |
| **holo_upload_id** | 大视频先上传(推荐 ≥8 MB) / pre-uploaded large file | `"input_video": {"holo_upload_id": "<uuid>"}` |

> 用 `holo_upload_id` 前先调 `POST /v1/uploads/presign-multipart` 走分段上传,完成后把返回的 `upload_id` 直接塞进 `input_video`。详见下文 [上传源视频 / Upload Source Video](#upload-source-video--上传源视频)。
> To use `holo_upload_id`, first multipart-upload the video via `POST /v1/uploads/presign-multipart`, then pass the returned `upload_id` directly. See [Upload Source Video](#upload-source-video--上传源视频) below.

字段名兼容 / Field aliases: `input_video` / `video_url` / `reference_video` 等价 / interchangeable.

| Model / 模型 | Resolution / 分辨率 |
|---|---|
| `omni_flash_edit` | 720p |
| `omni_flash_edit_1080p` | 1080p |
| `omni_flash_edit_4k` | 4K |

> **时长 / Duration:** 输出视频时长不在模型名里,通过 body `seconds` 字段指定(`4` / `6` / `8` / `10`)。不传时按模型/后端默认。
> Output duration is set via body `seconds` (`4` / `6` / `8` / `10`), not in model name. Falls back to backend default if omitted.
>
> **横竖屏 / Orientation:** **必须**通过 body `aspect_ratio` 显式传(`"16:9"` 横屏 / `"9:16"` 竖屏)。**不传默认横屏(`16:9`)**,即使源视频是竖屏也会被强制转横。后端**不读源视频元数据**自动判向。
> Orientation **must** be set explicitly via body `aspect_ratio` (`"16:9"` landscape / `"9:16"` portrait). **Defaults to landscape (`16:9`) if omitted**, even if the source video is portrait — output will be force-rotated to landscape. Backend does **not** auto-detect source orientation.

**文生视频示例 / Text-to-Video example (8s, 竖屏 1080p):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_8s_1080p",
    "aspect_ratio": "9:16",
    "messages": [{"role":"user","content":"cinematic shot, a calm river at sunset with golden light"}]
  }'
```

**多图参考示例 / Components example (6s, 竖屏 720p, 多图):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_components_6s",
    "aspect_ratio": "9:16",
    "messages": [{"role":"user","content":[
      {"type":"image_url","image_url":{"url":"https://example.com/ref-1.jpg"}},
      {"type":"image_url","image_url":{"url":"https://example.com/ref-2.jpg"}},
      {"type":"text","text":"combine the references into one coherent cinematic scene"}
    ]}]
  }'
```

**视频编辑示例 A / Video Edit example A (1080p, http URL):**

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_edit_1080p",
    "input_video": "https://your-cdn.com/source.mp4",
    "messages": [{"role":"user","content":"change the background to a cyberpunk neon city at night, keep the character motion unchanged"}]
  }'
```

**视频编辑示例 B / Video Edit example B (1080p, 大文件预上传 / pre-uploaded):**

```bash
# Step 1: 上传后(见下方上传流程),把拿到的 upload_id 塞进 input_video
# Step 1: After upload (see flow below), pass upload_id into input_video
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_edit_4k",
    "input_video": {"holo_upload_id": "8f3c1a72-...-...-..."},
    "messages": [{"role":"user","content":"add a slow zoom-in and cinematic color grade"}]
  }'
```

**视频编辑示例 C / Video Edit example C (参考视频 + 参考图片 + 文字 prompt / video + reference images + text):**

> `omni_flash_edit_*` 支持在参考视频之外**额外附带 0-3 张参考图片**,用作风格/角色/服饰等引导。`input_video` 与 `messages.content[].image_url` 同时存在即触发该组合模式。
> `omni_flash_edit_*` accepts **0-3 additional reference images** alongside the source video — for style / character / wardrobe guidance. Combine by passing both `input_video` and `image_url` parts in `messages.content`.

```bash
curl -X POST https://api.dealonhorizon.us/v1/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_edit_1080p",
    "input_video": "https://your-cdn.com/source.mp4",
    "aspect_ratio": "9:16",
    "messages": [{"role":"user","content":[
      {"type":"image_url","image_url":{"url":"https://example.com/style-ref.jpg"}},
      {"type":"image_url","image_url":{"url":"https://example.com/character-ref.jpg"}},
      {"type":"text","text":"restyle the video using these references — keep the motion, swap the outfit and lighting"}
    ]}]
  }'
```

> **⚠️ `aspect_ratio` 强烈建议显式传 / Strongly recommend setting `aspect_ratio` explicitly:**
> - **省略 / omit** → 输出**默认横屏 `16:9`**,源视频是竖屏也会被强制转横 / defaults to landscape `16:9`, even if source is portrait — output is force-rotated
> - **显式传 / explicit** → 按你指定的 `"16:9"`(横屏 / landscape)或 `"9:16"`(竖屏 / portrait)输出
> - **`seconds` 字段同理**:不传按后端默认;显式传 `"4"/"6"/"8"/"10"` 才能精确控制 / Same for `seconds`: pass `"4"/"6"/"8"/"10"` to control precisely; falls back to backend default if omitted.
> - 想竖屏 → **必须传** `"aspect_ratio": "9:16"`。后端**不会自动读源视频元数据**判向 / For portrait output, you **must** pass `"aspect_ratio": "9:16"`. Backend does **not** parse source video metadata to auto-detect.

**注意 / Notes:**
- 模型名用 **underscore** 命名(`omni_flash_*`),不是 hyphen / Model name uses **underscore** (`omni_flash_*`), not hyphen
- 必须传 `messages` 字段(OpenAI chat completions 格式),不是 `prompt` / Use `messages` (OpenAI chat completions format), not `prompt`
- 横竖屏走 body `aspect_ratio`,模型名里不带 `_landscape` / `_portrait` / Orientation via body `aspect_ratio`, not in model name
- 参考图支持 **base64 data URI** 和 **http(s) URL** 两种 / Reference images accept both base64 data URI and http(s) URL
- `omni_flash_edit_*` 参考图最多 **3 张**,`omni_flash_components_*` 参考图最多 **3 张**(详见 go2flow ModelCatalog) / `omni_flash_edit_*` accepts up to **3** reference images; `omni_flash_components_*` accepts up to **3**
- 各模型实时定价请查 `GET /me` 的 `effective_pricing` 字段 / Per-model pricing returned by `GET /me` in the `effective_pricing` field

**返回与轮询 / Return + Poll:** 与 veo/sora/grok 完全一致 — 返回 `{ "task_id": "...", "status": "queued", ... }`,之后 `GET /v1/tasks/{id}` 轮询、`GET /v1/tasks/{id}/file` 下载 mp4。
Same as veo/sora/grok — returns task, poll `GET /v1/tasks/{id}`, download via `GET /v1/tasks/{id}/file`.

---

#### Upload Source Video / 上传源视频

用于 omni edit 源视频(以及其他需要大文件参考的模型)。3 个端点走一遍即可,**完成后把 `upload_id` 直接塞进 `input_video` 字段**。
Used for omni edit source videos (and other large reference inputs). Three calls, then pass `upload_id` into `input_video`.

**限制 / Limits:**

| 项 / Item | 值 / Value |
|---|---|
| 单段最小 / Min part size | 5 MB (S3 协议硬性 / S3 protocol) |
| 单文件上限 / Max file size | **5 GB** |
| 段数上限 / Max parts | 2000 |
| 每小时配额 / Hourly quota | 1000 次上传 / 50 GB |
| 上传链接有效期 / PUT URL TTL | 20 min |
| 保留时长 / Retention | 7 天未引用自动清理 / 7 days unless pinned to a task |

> 大客户超过该额度联系运维开通 account-level 提额 / Enterprise users needing higher quotas — contact ops for an account-level uplift.

支持 MIME / Supported MIMEs: `video/mp4`, `video/quicktime`, `video/webm` (以及 `image/jpeg`,`image/png`,`image/webp` for reference images)

**Step 1: 初始化分段上传 / Initiate multipart**

`POST /v1/uploads/presign-multipart`

```json
{
  "filename":     "my-clip.mp4",
  "mime_type":    "video/mp4",
  "size_bytes":   134217728,
  "purpose":      "reference",
  "part_count":   26,
  "client_nonce": "<your-uuid-v4-or-8+-char-idempotency-key>"
}
```

返回 / Returns:

```json
{
  "mode": "r2-multipart",
  "upload_id": "8f3c1a72-...",
  "part_urls": [
    {"part_number": 1, "url": "https://...presigned-put-1..."},
    {"part_number": 2, "url": "https://...presigned-put-2..."},
    ...
  ],
  "min_part_bytes": 5242880,
  "expires_in": 1200
}
```

说明 / Notes:
- `part_count` 自己分:`ceil(size_bytes / chunk_size)`,每段 ≥ 5 MB(最后一段可小)。 / Pick `part_count` so each part is ≥5 MB (last part can be smaller).
- `client_nonce` 是幂等键。重发同一 `(filename, mime_type, size_bytes, client_nonce)` 会拿回同一 `upload_id` 和新签名的 part URLs。 / `client_nonce` is the idempotency key — replaying the same call returns the same `upload_id` with fresh part URLs.
- `purpose` 视频用 `"reference"`。 / Use `"reference"` for video.

**Step 2: 并发 PUT 各段 / Upload parts (parallel OK)**

对每个 `part_urls[i].url` 发 PUT 请求,body 是对应字节区间,**不要带任何额外 header**。收 200 后保留响应 header 里的 `ETag`(去掉外层引号)。
PUT each part to its URL with the corresponding byte range; **do not add extra headers**. Capture the `ETag` from each response (strip surrounding quotes).

```bash
curl -X PUT "<part_urls[0].url>" \
  --data-binary "@chunk-0.bin"
# 取响应里的 ETag, 如 "a1b2c3d4..."
```

**Step 3: 完成上传 / Complete**

`POST /v1/uploads/complete-multipart`

```json
{
  "upload_id": "8f3c1a72-...",
  "parts": [
    {"PartNumber": 1, "ETag": "a1b2c3d4..."},
    {"PartNumber": 2, "ETag": "e5f6g7h8..."}
  ]
}
```

> 字段名兼容小驼峰 `partNumber` / `etag`。 / Field names also accept lowerCamel: `partNumber` / `etag`.

返回 `{"ok": true, "upload_id": "..."}` 后即可在 `/v1/generate` 用 `"input_video": {"holo_upload_id": "<upload_id>"}`。
Returns `{"ok": true, "upload_id": "..."}` — now usable in `/v1/generate` as `"input_video": {"holo_upload_id": "<upload_id>"}`.

**取消上传 / Abort (optional):**

`POST /v1/uploads/abort-multipart` body `{"upload_id": "..."}` — 中途放弃可调,幂等;不调也会自然过期。
Idempotent abort for abandoned uploads; safe to skip (presign URLs expire naturally).

**(可选)手动签 GET URL / (Optional) manual GET URL:**

`GET /v1/uploads/{upload_id}/url` 返回一条 6h 有效的签名 GET URL,可在自己流程里调试或拼到 `input_video` 字符串字段使用 —— 但**直接传 `holo_upload_id` 更省一次往返**。
Returns a 6h-valid signed GET URL for the uploaded object. Use only when you need to plug the URL elsewhere — for `/v1/generate`, passing `holo_upload_id` directly is one fewer round-trip.

---

## Content Safety / 内容安全

Requests that violate Google's content policies are automatically rejected and **refunded in full**.
违反 Google 内容政策的请求会被自动拒绝，积分**全额退还**。

Common rejection reasons / 常见拒绝原因:

| Code / 错误码 | Meaning / 含义 |
|------|---------|
| `PUBLIC_ERROR_PROMINENT_PEOPLE_UPLOAD` | Input image contains a public figure / 输入图含名人 |
| `PUBLIC_ERROR_PROMINENT_PEOPLE_FILTER_FAILED` | Output resembles a public figure / 输出类似名人 |
| `PUBLIC_ERROR_SEXUAL` | Sexual or explicit content / 色情内容 |
| `PUBLIC_ERROR_VIOLENCE` | Violence or dangerous activities / 暴力内容 |
| `PUBLIC_ERROR_DANGEROUS` | Dangerous content / 危险内容 |

---

## Error Handling / 错误处理

### Submission Errors / 提交错误（立即返回）

| Status / 状态码 | Meaning / 含义 | Action / 建议 |
|--------|---------|--------|
| 202 | Task accepted / 任务已接受 | Poll `/v1/tasks/{id}` / 轮询获取结果 |
| 400 | Invalid request / 请求无效 | Fix parameters / 检查参数 |
| 401 | Missing or invalid API key / API Key 无效 | Check your key / 检查 API Key |
| 402 | Insufficient credits / 积分不足 | Top up / 充值 |
| 429 | Rate limit exceeded / 频率限制 | Wait and retry / 等待后重试 |
| 503 | Service unavailable / 服务不可用 | Retry later / 稍后重试 |

### Task Failure / 任务失败（通过轮询获取）

All failed tasks are **automatically refunded** (`"refunded": true`).
所有失败任务**自动退还积分**。

---

## Quick Start / 快速上手

### Image Generation / 图片生成

```python
import requests
import time

API_KEY = "your_api_key_here"  # 替换为你的 API Key
BASE = "https://api.dealonhorizon.us"
HEADERS = {"Authorization": f"Bearer {API_KEY}"}

# 1. Submit / 提交
resp = requests.post(f"{BASE}/v1/generate", headers=HEADERS, json={
    "model": "gemini-3.0-pro-image-landscape",
    "messages": [{"role": "user", "content": "A sunset over mountains"}]
})
task = resp.json()
print(f"Task {task['task_id']} queued at position {task['position']}")

# 2. Poll / 轮询
while True:
    status = requests.get(f"{BASE}/v1/tasks/{task['task_id']}", headers=HEADERS).json()
    if status["status"] == "completed":
        # 3. Download / 下载
        file_resp = requests.get(f"{BASE}{status['result']['file_url']}", headers=HEADERS)
        ext = status["result"].get("file_ext", "png")
        with open(f"result.{ext}", "wb") as f:
            f.write(file_resp.content)
        print(f"Saved! ({status['result']['file_size']} bytes)")
        break
    elif status["status"] == "failed":
        print(f"Failed: {status.get('error')} (refunded: {status.get('refunded')})")
        break
    time.sleep(5)
```

### Video Generation / 视频生成

```python
# Text-to-Video / 文字转视频
resp = requests.post(f"{BASE}/v1/generate", headers=HEADERS, json={
    "model": "veo_3_1_t2v_fast_landscape",
    "messages": [{"role": "user", "content": "A drone shot flying over a tropical island"}]
})
# Poll same as image / 轮询方式和图片完全一样

# Image-to-Video / 图片转视频
resp = requests.post(f"{BASE}/v1/generate", headers=HEADERS, json={
    "model": "veo_3_1_i2v_fast_landscape",
    "messages": [{"role": "user", "content": [
        {"type": "image_url", "image_url": {"url": "https://example.com/photo.jpg"}},
        {"type": "text", "text": "Slowly zoom in with gentle camera movement"}
    ]}]
})
# Poll same as image / 轮询方式和图片完全一样
```

### Batch Processing / 批量处理

```python
# For batch generation, submit multiple /v1/generate requests
# 批量生成时，提交多个 /v1/generate 请求

import concurrent.futures

prompts = [f"Beautiful landscape scene #{i}" for i in range(50)]
tasks = []

# Submit all tasks / 提交所有任务
for prompt in prompts:
    resp = requests.post(f"{BASE}/v1/generate", headers=HEADERS, json={
        "model": "gemini-3.0-pro-image-landscape",
        "messages": [{"role": "user", "content": prompt}]
    })
    tasks.append(resp.json()["task_id"])
    time.sleep(0.5)  # Respect rate limits / 注意频率限制

# Poll all tasks / 轮询所有任务
for task_id in tasks:
    while True:
        st = requests.get(f"{BASE}/v1/tasks/{task_id}", headers=HEADERS).json()
        if st["status"] in ("completed", "failed"):
            if st["status"] == "completed":
                # Download file / 下载文件
                file_resp = requests.get(f"{BASE}{st['result']['file_url']}", headers=HEADERS)
                with open(f"{task_id[:8]}.{st['result']['file_ext']}", "wb") as f:
                    f.write(file_resp.content)
            break
        time.sleep(3)
```

---

## Limits / 使用限制

- **Queue Timeout / 排队超时**: If the system is busy, your task may wait in queue. If it cannot start in time, it will be cancelled and credits refunded automatically. / 系统繁忙时任务会排队，超时未处理将自动取消并退款。
- **File Retention / 文件保留**: 24 hours after generation / 生成后保留 24 小时
- **Fair Queuing / 公平排队**: All users share equal priority / 所有用户享有平等优先级

---

## Dashboard / 控制面板

Web dashboard / 网页控制面板: `https://api.dealonhorizon.us/dashboard`

- **API Key login** / API Key 登录: View your balance, usage history, and transaction records / 查看余额、使用记录、交易记录
- **Account password login** / 账户密码登录: Full account management — all keys, allocate credits, create/delete keys / 完整账户管理
- Light/dark theme, Chinese/English / 明暗主题、中英文切换
