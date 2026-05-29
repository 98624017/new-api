# W Project HOLO API 开箱即用调用文档

最后更新：2026-05-26
基础地址：`https://api.dealonhorizon.us`
接口版本来源：上传的 `W Project HOLO API Reference v1.7.1 (2026-05-24)`

## 1. 实测结论

你提供的 API Key 已实测可用，但本文档不会写入明文 Key，避免误提交或泄露。

| 检查项 | 结果 | 说明 |
|---|---:|---|
| `GET /me` | `HTTP 200` | Key 有效；账户为 trial，剩余赠送额度为 `150.0` credits，`rpm_limit=60` |
| `GET /v1/tasks?limit=3` | `HTTP 200` | 鉴权和任务列表接口可用；当前任务列表为空 |
| `GET /v1/models` | `HTTP 200` | 模型列表接口可用 |
| `GET /banner` | `HTTP 200` | 当前无公告 |
| `GET /health` | `HTTP 200` | API 在线，但 2026-05-26 实测状态为 `degraded`，容量为 `0/0 generators available` |
| `POST /v1/generate` Omni Flash | `HTTP 202` 后完成 | `omni_flash_4s` 真实生成成功，见下方实测记录 |

注意：`/health` 在实测时显示生成器容量为 0，但一次 `omni_flash_4s` 任务仍然成功进入队列并完成。生产接入仍建议先检查 `/health`，异常时降低提交频率或提示用户稍后重试。

### 1.1 Omni Flash 真实生成记录

| 字段 | 值 |
|---|---|
| 测试时间 | 2026-05-26 |
| 模型 | `omni_flash_4s` |
| 参数 | `aspect_ratio="16:9"`，`seconds="4"` |
| 提交状态 | `HTTP 202` |
| 任务 ID | `c6a8ab9493b8444696636cacd8aabc22` |
| 队列位置 | `1` |
| 扣费 | `42` credits |
| 完成耗时 | 约 `51s` |
| 下载类型 | `video/mp4` |
| 文件大小 | `756142` bytes |
| 本地文件 | `holo-omni-flash-test-c6a8ab9493b8444696636cacd8aabc22.mp4` |
| 公开直链 | `https://media.dealonhorizon.us/c6a8ab9493b8444696636cacd8aabc22.mp4` |

### 1.2 Omni Flash 多参考图 10s 真实生成记录

| 字段 | 值 |
|---|---|
| 测试时间 | 2026-05-26 |
| 模型 | `omni_flash_components_10s` |
| 参数 | `aspect_ratio="16:9"` |
| 输入 | 2 张 `data:image/png;base64,...` 参考图 + 文本 |
| 提交状态 | `HTTP 202` |
| 任务 ID | `eb7bcf3c71614d498c825c7c5f19364a` |
| 队列位置 | `2` |
| 扣费 | `84` credits |
| 完成耗时 | 约 `61s` |
| 下载类型 | `video/mp4` |
| 文件大小 | `2800666` bytes |
| 本地文件 | `holo-components10s-png-eb7bcf3c71614d498c825c7c5f19364a.mp4` |
| 公开直链 | `https://media.dealonhorizon.us/eb7bcf3c71614d498c825c7c5f19364a.mp4` |

补充验证：

- 多参考图使用远程图片 URL 时，接口返回 `400 {"error":"Failed to download image"}`，未扣费。
- 多参考图使用 `data:image/svg+xml;base64,...` 时，接口返回 `400 {"error":"Inline reference rejected by content screening","detail":"bad_mime"}`，未扣费。
- 多参考图使用真正的 `data:image/png;base64,...` 时提交成功。

### 1.3 Omni Flash 视频编辑 10s 真实提交记录

| 字段 | 值 |
|---|---|
| 测试时间 | 2026-05-26 |
| 模型 | `omni_flash_edit` |
| 参数 | `aspect_ratio="16:9"`，`seconds="10"` |
| 输入视频 | `https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4` |
| 提交状态 | `HTTP 402` |
| 返回 | `{"error":"Insufficient credits","cost":112}` |
| 结论 | 当前 Key 余额不足，未创建任务，未扣费 |

## 2. 环境变量准备

所有示例都默认从环境变量读取 Key。不要把真实 Key 写进代码仓库。

```bash
export HOLO_API_KEY="替换为你的 API Key"
export HOLO_BASE_URL="https://api.dealonhorizon.us"
```

推荐请求头二选一：

```http
Authorization: Bearer <HOLO_API_KEY>
```

或：

```http
X-API-Key: <HOLO_API_KEY>
```

本文示例统一使用 `Authorization: Bearer ...`。如果你接入的平台更适合自定义头，也可以改成 `X-API-Key`。

## 3. 公开素材 URL

下面这些 URL 已做 HEAD 检查，适合直接放进示例里测试。实际生产建议使用你自己的 CDN 或对象存储。

| 用途 | URL | 备注 |
|---|---|---|
| 图片参考图 | `https://upload.wikimedia.org/wikipedia/commons/3/3f/Fronalpstock_big.jpg` | JPEG，公开可访问，支持 Range/CORS |
| 视频编辑源视频 | `https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4` | MP4，公开可访问，约 1.1 MB |

## 4. 先检查服务是否可生成

提交生成任务前先看服务健康状态，避免容量不足时提交任务。

```bash
curl -sS "$HOLO_BASE_URL/health"
```

理想情况类似：

```json
{
  "service": "api",
  "status": "ok",
  "capacity": "available"
}
```

如果返回类似下面这样，说明 API 在线但暂时没有可用生成器，建议稍后再提交：

```json
{
  "service": "holo-gen-reception",
  "status": "degraded",
  "capacity": "0/0 generators available"
}
```

## 5. 账户、模型和任务列表

### 5.1 查询账户余额

```bash
curl -sS "$HOLO_BASE_URL/me" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

重点字段：

| 字段 | 含义 |
|---|---|
| `credits` | 当前可用积分 |
| `frozen_credits` | 进行中任务冻结积分 |
| `daily_used` | 今日请求数 |
| `daily_credits_used` | 今日消耗积分 |
| `rpm_limit` | 每分钟请求限制，`0` 表示不限 |
| `effective_pricing` | 当前 Key 的实时模型价格，以此为准 |

### 5.2 查询可用模型

```bash
curl -sS "$HOLO_BASE_URL/v1/models" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

### 5.3 查询任务列表

```bash
curl -sS "$HOLO_BASE_URL/v1/tasks?limit=20" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

按状态过滤：

```bash
curl -sS "$HOLO_BASE_URL/v1/tasks?status=completed&limit=20&offset=0" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

## 6. 统一生成流程

所有图片和视频生成都走同一个入口：

```http
POST /v1/generate
```

流程固定为：

1. `POST /v1/generate` 提交任务，成功通常返回 `202 Accepted` 和 `task_id`。
2. 每 5-10 秒 `GET /v1/tasks/{task_id}` 轮询。
3. `status=completed` 后下载结果。
4. `status=failed` 或 `cancelled` 时停止；失败任务通常自动退款。

任务状态：

| 状态 | 含义 | 客户端动作 |
|---|---|---|
| `queued` | 排队中 | 继续轮询，可显示队列位置 |
| `processing` | 生成中 | 继续轮询 |
| `completed` | 完成 | 下载文件 |
| `failed` | 失败 | 展示错误，检查是否退款 |
| `cancelled` | 已取消 | 停止轮询 |

## 7. curl 示例

### 7.1 文生图

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3.1-flash-image-square",
    "messages": [
      {
        "role": "user",
        "content": "A clean product photo of a matte black smart speaker on a white desk, soft studio light"
      }
    ]
  }'
```

### 7.2 参考图生图

带 `image_url` 时，图片模型会自动按 R2I 参考生图处理。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3.0-pro-image-landscape",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "image_url",
            "image_url": {
              "url": "https://upload.wikimedia.org/wikipedia/commons/3/3f/Fronalpstock_big.jpg"
            }
          },
          {
            "type": "text",
            "text": "Recreate this mountain scene as a cinematic winter travel poster, keep the composition, add soft sunrise light"
          }
        ]
      }
    ]
  }'
```

### 7.3 文生视频：Veo

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "veo_3_1_t2v_fast_landscape",
    "messages": [
      {
        "role": "user",
        "content": "A smooth drone shot flying over snowy mountains at golden hour, cinematic, realistic"
      }
    ]
  }'
```

### 7.4 图生视频：Veo I2V

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "veo_3_1_i2v_fast_landscape",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "image_url",
            "image_url": {
              "url": "https://upload.wikimedia.org/wikipedia/commons/3/3f/Fronalpstock_big.jpg"
            }
          },
          {
            "type": "text",
            "text": "Slow cinematic push-in, clouds drifting gently, natural mountain atmosphere"
          }
        ]
      }
    ]
  }'
```

### 7.5 首尾帧视频：Veo I2V 两张图

Veo I2V 模型传 2 张 `image_url` 时，会自动切到首尾帧模式：第一张是起始帧，第二张是结束帧。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "veo_3_1_i2v_fast_landscape",
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "image_url",
            "image_url": {
              "url": "https://upload.wikimedia.org/wikipedia/commons/3/3f/Fronalpstock_big.jpg"
            }
          },
          {
            "type": "image_url",
            "image_url": {
              "url": "https://upload.wikimedia.org/wikipedia/commons/3/3f/Fronalpstock_big.jpg"
            }
          },
          {
            "type": "text",
            "text": "Create a smooth day-to-sunset transition with gentle camera movement"
          }
        ]
      }
    ]
  }'
```

### 7.6 Sora-2 视频

Sora 模型名大小写敏感，`size` 必填，`seconds` 必须是字符串。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Sora-2-12",
    "size": "1280x720",
    "seconds": "12",
    "messages": [
      {
        "role": "user",
        "content": "A small bird flying over a green meadow, natural handheld camera, realistic"
      }
    ]
  }'
```

### 7.7 Grok 文生视频

Grok 文生视频可以使用顶层 `prompt`。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-imagine-video-10s-720p",
    "prompt": "A fast cinematic dolly shot through a futuristic city street at night"
  }'
```

Grok 图生视频只接受 base64 data URI，不接受远程图片 URL。远程图片需要你先下载并转 base64。

### 7.8 Omni Flash 文生视频

Omni Flash 的横竖屏由 `aspect_ratio` 控制，不写在模型名里。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_8s_1080p",
    "aspect_ratio": "9:16",
    "messages": [
      {
        "role": "user",
        "content": "Vertical cinematic shot of a calm river at sunset, golden light, slow camera movement"
      }
    ]
  }'
```

### 7.9 Omni Flash 视频编辑

视频编辑输入可以是公开 MP4 URL、base64 data URI，或预上传后的 `holo_upload_id`。

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/generate" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "omni_flash_edit_1080p",
    "input_video": "https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4",
    "aspect_ratio": "16:9",
    "seconds": "4",
    "messages": [
      {
        "role": "user",
        "content": "Turn the flower video into a cinematic macro shot with richer colors and gentle camera motion"
      }
    ]
  }'
```

## 8. 轮询和下载

假设提交返回：

```json
{
  "task_id": "abc123def456",
  "status": "queued",
  "position": 12,
  "cost": 12,
  "model": "gemini-3.1-flash-image-square",
  "created_at": "2026-03-26T12:00:00+00:00"
}
```

轮询任务：

```bash
TASK_ID="abc123def456"

curl -sS "$HOLO_BASE_URL/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

完成响应会包含：

```json
{
  "task_id": "abc123def456",
  "status": "completed",
  "result": {
    "file_url": "/v1/tasks/abc123def456/file",
    "file_ext": "png",
    "file_size": 1234567,
    "duration_ms": 45000,
    "type": "t2i"
  }
}
```

下载方式 1：鉴权下载，适合私密内容。

```bash
curl -L -o "result.png" "$HOLO_BASE_URL/v1/tasks/$TASK_ID/file" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

下载方式 2：公开 CDN 直链，适合展示或分享，但拿到 URL 的人都能访问。

```bash
FILE_EXT="png"
curl -L -o "result.$FILE_EXT" \
  "https://media.dealonhorizon.us/$TASK_ID.$FILE_EXT"
```

结果文件通常保留 24 小时。

## 9. Python 开箱即用脚本

保存为 `holo_generate.py` 后运行：

```bash
python3 holo_generate.py
```

脚本默认会先检查健康状态。如果服务不是可用状态，会直接退出，不提交扣费任务。

```python
import os
import sys
import time
from pathlib import Path

import requests

BASE = os.getenv("HOLO_BASE_URL", "https://api.dealonhorizon.us")
API_KEY = os.getenv("HOLO_API_KEY")

if not API_KEY:
    raise SystemExit("请先设置环境变量：export HOLO_API_KEY='你的 API Key'")

HEADERS = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json",
}


def check_health() -> None:
    resp = requests.get(f"{BASE}/health", timeout=20)
    resp.raise_for_status()
    data = resp.json()
    print("health:", data)
    capacity = str(data.get("capacity", "")).lower()
    status = str(data.get("status", "")).lower()
    if status not in {"ok", "available"} and "available" not in capacity:
        raise SystemExit("服务当前可能不可生成，已停止提交任务。")
    if "0/0" in capacity or "degraded" in status:
        raise SystemExit("服务当前容量不足，已停止提交任务。")


def submit_generation() -> str:
    payload = {
        "model": "gemini-3.1-flash-image-square",
        "messages": [
            {
                "role": "user",
                "content": "A clean product photo of a matte black smart speaker on a white desk, soft studio light",
            }
        ],
    }
    resp = requests.post(f"{BASE}/v1/generate", headers=HEADERS, json=payload, timeout=30)
    if resp.status_code not in (200, 202):
        raise RuntimeError(f"submit failed: {resp.status_code} {resp.text}")
    data = resp.json()
    print("submitted:", data)
    return data["task_id"]


def poll_task(task_id: str, interval: int = 8, timeout_seconds: int = 900) -> dict:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        resp = requests.get(f"{BASE}/v1/tasks/{task_id}", headers=HEADERS, timeout=30)
        resp.raise_for_status()
        data = resp.json()
        status = data.get("status")
        print("status:", status, "position:", data.get("position"))

        if status == "completed":
            return data
        if status in {"failed", "cancelled"}:
            raise RuntimeError(f"task ended: {data}")

        time.sleep(interval)

    raise TimeoutError(f"task timeout: {task_id}")


def download_result(task: dict) -> Path:
    task_id = task["task_id"]
    result = task["result"]
    ext = result.get("file_ext", "bin")
    output = Path(f"holo-result-{task_id}.{ext}")
    resp = requests.get(f"{BASE}/v1/tasks/{task_id}/file", headers=HEADERS, timeout=120)
    resp.raise_for_status()
    output.write_bytes(resp.content)
    print("saved:", output, "bytes:", output.stat().st_size)
    return output


if __name__ == "__main__":
    check_health()
    task_id = submit_generation()
    task = poll_task(task_id)
    download_result(task)
```

## 10. Node.js 开箱即用脚本

Node 18+ 原生支持 `fetch`。保存为 `holo-generate.mjs` 后运行：

```bash
node holo-generate.mjs
```

```js
import { writeFile } from "node:fs/promises";

const BASE = process.env.HOLO_BASE_URL ?? "https://api.dealonhorizon.us";
const API_KEY = process.env.HOLO_API_KEY;

if (!API_KEY) {
  throw new Error("请先设置环境变量：export HOLO_API_KEY='你的 API Key'");
}

const headers = {
  Authorization: `Bearer ${API_KEY}`,
  "Content-Type": "application/json",
};

async function requestJson(url, options = {}) {
  const response = await fetch(url, options);
  const text = await response.text();
  let data;
  try {
    data = text ? JSON.parse(text) : null;
  } catch {
    data = text;
  }
  if (!response.ok) {
    throw new Error(`${response.status} ${response.statusText}: ${text}`);
  }
  return data;
}

async function checkHealth() {
  const data = await requestJson(`${BASE}/health`);
  console.log("health:", data);
  const status = String(data.status ?? "").toLowerCase();
  const capacity = String(data.capacity ?? "").toLowerCase();
  if (status.includes("degraded") || capacity.includes("0/0")) {
    throw new Error("服务当前容量不足，已停止提交任务。");
  }
}

async function submit() {
  const body = {
    model: "gemini-3.1-flash-image-square",
    messages: [
      {
        role: "user",
        content:
          "A clean product photo of a matte black smart speaker on a white desk, soft studio light",
      },
    ],
  };

  const data = await requestJson(`${BASE}/v1/generate`, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });
  console.log("submitted:", data);
  return data.task_id;
}

async function poll(taskId) {
  const deadline = Date.now() + 15 * 60 * 1000;
  while (Date.now() < deadline) {
    const data = await requestJson(`${BASE}/v1/tasks/${taskId}`, { headers });
    console.log("status:", data.status, "position:", data.position);
    if (data.status === "completed") return data;
    if (["failed", "cancelled"].includes(data.status)) {
      throw new Error(`task ended: ${JSON.stringify(data)}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 8000));
  }
  throw new Error(`task timeout: ${taskId}`);
}

async function download(task) {
  const ext = task.result?.file_ext ?? "bin";
  const output = `holo-result-${task.task_id}.${ext}`;
  const response = await fetch(`${BASE}/v1/tasks/${task.task_id}/file`, {
    headers: { Authorization: `Bearer ${API_KEY}` },
  });
  if (!response.ok) {
    throw new Error(`download failed: ${response.status} ${await response.text()}`);
  }
  const bytes = new Uint8Array(await response.arrayBuffer());
  await writeFile(output, bytes);
  console.log("saved:", output, "bytes:", bytes.byteLength);
}

await checkHealth();
const taskId = await submit();
const task = await poll(taskId);
await download(task);
```

## 11. 模型选择速查

### 11.1 图片

| 场景 | 推荐模型 | 说明 |
|---|---|---|
| 快速通用文生图 | `gemini-3.1-flash-image-square` | 成本和速度相对均衡 |
| 横图海报 | `gemini-3.0-pro-image-landscape` | 横版 |
| 竖图海报 | `gemini-3.0-pro-image-portrait` | 竖版 |
| 方图头像/商品图 | `gemini-3.1-flash-image-square` | 方形 |
| 更高分辨率 | 加 `-2k` 或 `-4k` 后缀 | 更贵，耗时更长 |
| GPT Images | `GPT-images2` 或 `GPT-images2 16:9-4K` | 模型名包含空格，必须原样传 |

Gemini 方向后缀：

| 后缀 | 比例 |
|---|---|
| `landscape` | 横屏 |
| `portrait` | 竖屏 |
| `square` | 方形 |
| `four-three` | 4:3 |
| `three-four` | 3:4 |

### 11.2 视频：Veo

| 场景 | 推荐模型 |
|---|---|
| 快速文生视频横屏 | `veo_3_1_t2v_fast_landscape` |
| 快速文生视频竖屏 | `veo_3_1_t2v_fast_portrait` |
| 快速图生视频横屏 | `veo_3_1_i2v_fast_landscape` |
| 快速图生视频竖屏 | `veo_3_1_i2v_fast_portrait` |
| 低成本预览 | `veo_3_1_t2v_lite_landscape` 或 `veo_3_1_i2v_lite_landscape` |
| 高质量 | `veo_3_1_t2v_landscape` 或 `veo_3_1_i2v_s_landscape` |
| 4 秒短视频 | `*_4s_*` |
| 6 秒短视频 | `*_6s_*` |
| 1080p | 加 `_1080p` 后缀 |
| 4K | 加 `_4k` 后缀 |

### 11.3 Sora、Grok、Omni

| 模型族 | 关键点 |
|---|---|
| `Sora-2-12` | `size` 必填；`seconds` 写字符串 `"12"`；可带 1 张参考图 |
| `grok-imagine-video-*` | 文生视频可用 `prompt`；图生视频只接受 base64 data URI |
| `omni_flash_*` | 横竖屏使用 `aspect_ratio`，不是模型名后缀 |
| `omni_flash_edit_*` | 支持视频编辑，源视频可用公开 URL、base64 或 `holo_upload_id` |

## 12. 错误处理

| HTTP 状态码 | 现象 | 影响 | 建议 |
|---:|---|---|---|
| `400` | 请求参数错误 | 任务不会提交 | 检查模型名、JSON、图片 URL 是否可下载 |
| `401` | Key 缺失或无效 | 无法鉴权 | 检查请求头和 Key 是否正确 |
| `402` | 余额不足 | 无法提交任务 | 充值或换 Key |
| `429` | 频率限制 | 请求被限流 | 降低并发，按 `rpm_limit` 节流 |
| `503` | 服务忙或暂停 | 暂时无法生成 | 稍后重试，先看 `/health` |

任务级失败会在轮询响应里出现：

```json
{
  "task_id": "abc123",
  "status": "failed",
  "error": "Content policy violation",
  "refunded": true
}
```

处理建议：

1. `failed`：停止轮询，展示 `error`，记录 `task_id`。
2. `refunded=true`：说明积分已退还。
3. 内容安全类错误：调整提示词或输入素材，避免名人、色情、暴力、危险内容。
4. 图片下载失败：确认图片 URL 是公网可访问、返回正确 `Content-Type`，且不是登录态资源。

## 13. 取消任务

只支持取消 `queued` 状态任务。

```bash
TASK_ID="替换为任务 ID"

curl -sS -X DELETE "$HOLO_BASE_URL/v1/tasks/$TASK_ID" \
  -H "Authorization: Bearer $HOLO_API_KEY"
```

成功响应：

```json
{
  "ok": true,
  "task_id": "abc123",
  "refunded": 12
}
```

## 14. 视频大文件上传

当源视频较大，或不方便用公网 URL 时，先使用 multipart 上传，再把返回的 `upload_id` 传给 `input_video`。

### 14.1 初始化分段上传

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/uploads/presign-multipart" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "source.mp4",
    "mime_type": "video/mp4",
    "size_bytes": 134217728,
    "purpose": "reference",
    "part_count": 26,
    "client_nonce": "your-unique-nonce-001"
  }'
```

返回里会有 `upload_id` 和每段的 `part_urls`。

### 14.2 PUT 上传每一段

对每个预签名 URL 上传对应字节段，保留响应头里的 `ETag`。

```bash
curl -X PUT "<part_url>" \
  --data-binary "@chunk-001.bin"
```

### 14.3 完成上传

```bash
curl -sS -X POST "$HOLO_BASE_URL/v1/uploads/complete-multipart" \
  -H "Authorization: Bearer $HOLO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_id": "替换为 upload_id",
    "parts": [
      {"PartNumber": 1, "ETag": "替换为第一段 ETag"},
      {"PartNumber": 2, "ETag": "替换为第二段 ETag"}
    ]
  }'
```

然后在生成请求里使用：

```json
{
  "model": "omni_flash_edit_1080p",
  "input_video": {
    "holo_upload_id": "替换为 upload_id"
  },
  "aspect_ratio": "16:9",
  "messages": [
    {
      "role": "user",
      "content": "Add a slow zoom-in and cinematic color grade"
    }
  ]
}
```

## 15. 生产接入建议

1. Key 放服务端，不要放前端、Electron renderer 或移动端包里。
2. 提交任务前先查 `/health`，状态异常时暂停生成按钮或进入稍后重试。
3. 按 `rpm_limit` 做节流；当前实测 Key 为 `60` RPM。
4. 轮询间隔使用 5-10 秒，不要 1 秒内高频轮询。
5. 客户端记录 `task_id`，断线后可恢复查询。
6. 结果文件只保留 24 小时，完成后尽快下载到自己的存储。
7. 私密内容使用鉴权下载，不要把 `media.dealonhorizon.us/{task_id}.{ext}` 公链发给不可信对象。
8. 定价不要硬编码，以 `GET /me` 的 `effective_pricing` 为准。
