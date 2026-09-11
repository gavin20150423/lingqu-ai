# Gavin2API 视频 API 下游接入文档

> 面向下游服务端开发者。本文档描述 Gavin2API 当前实际对外暴露的视频接口、字段、状态、计费和错误处理约定。
>
> 文档范围：Gavin2API 自有统一异步视频 API。上游供应商的 URL、凭证、任务 ID 和字段由服务端适配，下游不需要直接访问上游。

## 0. 接入范围

本文档只描述 Gavin2API 自有的视频任务接口。平台会根据 API Key 所属视频分组和后台配置选择可用模型与上游账号；下游只需要调用统一接口。

## 1. 快速开始

### 1.1 平台方需要提供

- `Base URL`，例如 `https://api.gavinteam.online`。调用时不要重复拼接 `/v1`。
- 一个已启用视频分组的 API Key。
- 该 Key 可用的模型、余额和并发配额。

所有视频资源都按创建资源的 API Key 隔离。创建、查询、取消、播放和下载必须使用同一个 Key。

### 1.2 推荐调用流程

1. `GET /v1/models`，读取当前 Key 真正可用的模型和能力字段。
2. 文生视频直接创建；需要平台托管素材时先 `POST /v1/videos/uploads`。
3. `POST /v1/videos` 或 `POST /v1/videos/generations` 创建异步任务，并保存 `job_id` 和 `status_url`。
4. 从约 5 秒开始轮询任务；长任务逐步退避到 30 秒。
5. 状态为 `completed` 后使用 `content_url` 下载，建议立即转存到自己的对象存储。
6. 网络超时或 5xx 重试创建时，必须复用完全相同的请求体和 `Idempotency-Key`。

### 1.3 最小请求头

```http
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
Accept: application/json
Idempotency-Key: video-order-20260909-0001
```

`Prefer: respond-async` 推荐携带。统一视频接口始终返回异步任务；外部 `/v1` 网关不要求该头，携带后响应会返回 `Preference-Applied: respond-async`。浏览器工作台的内部接口另有强制校验，不属于下游 API。

`Idempotency-Key` 对创建任务是必填项：长度 1-128，字符必须是可打印 ASCII（`0x20`-`0x7E`）。推荐使用业务订单号或 UUID，不要使用会被多个业务请求复用的固定值。

## 2. 通用约定

### 2.1 地址、鉴权和请求追踪

```text
BASE_URL=https://api.gavinteam.online
```

请求：

```http
Authorization: Bearer YOUR_API_KEY
```

不要把 API Key 放到 URL、前端源码、日志、截图或回调参数中。网关会自动返回 `X-Client-Request-ID` 作为客户端请求关联号；如需自定义入站关联号，可提供 `X-Request-ID`（长度不超过 64 字节）。上游错误如果带有安全的追踪号，网关可能在错误响应中返回 `X-Request-Id`。

### 2.2 内容类型和大小

- JSON 接口必须使用 `Content-Type: application/json`，请求体必须是一个完整 JSON 对象。
- 创建任务的 JSON 读取上限约为 2 MiB；超限或截断后通常返回 `VIDEO_REQUEST_INVALID`。
- 上传接口必须使用带 boundary 的 `multipart/form-data`。使用 `curl -F`、浏览器 `FormData` 或 SDK 的 multipart 实现，不要手工填写 boundary。
- 上传请求体还受部署项 `gateway.max_body_size` 限制，实际媒体大小以平台配置和上游模型限制为准。
- 下载接口返回二进制流，可能透传 `Content-Type`、`Content-Length`、`Content-Range`、`Accept-Ranges`、`Content-Disposition`、`ETag` 等安全响应头。

### 2.3 URL 和资源隔离

- 外部媒体 URL 必须是上游可访问的绝对 `http://` 或 `https://` URL；不要依赖 Cookie、内网地址或一次性不可复用链接。
- 不接受 `data:` URL，也不要把 Base64 或媒体字节放进创建任务 JSON。
- 上传接口返回的 `url` 可以直接作为 `start_frame_url` 或 `guidances.*` 中的媒体 URL。网关会把本地上传 URL 映射为对应上游 URL。
- 上传素材和任务均按 API Key 隔离。Key 不一致、素材已过期或资源不存在时统一按资源不存在处理，不能据此探测其他 Key 的资源。

### 2.4 错误结构

HTTP 错误通常为：

```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "VIDEO_RESOLUTION_INVALID",
    "message": "resolution is not supported by this model"
  }
}
```

程序分支应使用 HTTP 状态和 `error.code`，不要匹配 `message` 文本。异步任务生成失败不会以创建请求错误返回，而是在任务查询结果的 `error` 字段中返回。

## 3. 接口速查（统一异步视频 API）

| 方法 | 路径 | 说明 | 成功状态 |
| --- | --- | --- | --- |
| `GET` | `/v1/models` | 查询当前 Key 可用模型、能力和价格变体 | `200` |
| `POST` | `/v1/videos/uploads` | 上传图片、视频或音频素材 | `201` |
| `GET` | `/v1/videos/uploads/{media_id}/content` | 读取已上传素材 | `200` / `206` |
| `POST` | `/v1/videos` | OpenAI 风格创建别名 | `202` |
| `POST` | `/v1/videos/generations` | 工作台兼容创建路径 | `202` |
| `GET` | `/v1/videos/jobs?limit=20` | 查询最近任务 | `200` |
| `GET` | `/v1/videos/jobs/{job_id}` | 查询并刷新单个任务 | `200` |
| `GET` | `/v1/videos/{job_id}` | 单任务查询别名 | `200` |
| `DELETE` | `/v1/videos/jobs/{job_id}` | 取消任务 | `200` |
| `GET` | `/v1/videos/jobs/{job_id}/content` | 播放或下载成品 | `200` / `206` |
| `GET` | `/v1/videos/{job_id}/content` | 成品下载别名 | `200` / `206` |

## 4. 查询可用模型

### 4.1 请求

```bash
curl -sS "$BASE_URL/v1/models" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Accept: application/json"
```

### 4.2 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "seedance-2.0",
      "object": "model",
      "owned_by": "video",
      "capability_source": "native",
      "resolutions": ["480p", "720p", "1080p"],
      "default_resolution": "720p",
      "default_duration": 8,
      "default_aspect_ratio": "16:9",
      "durations": [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15],
      "aspect_ratios": ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "9:21"],
      "supports_audio": true,
      "supports_guidances": true,
      "supports_start_frame": true,
      "requires_start_frame": false,
      "supports_end_frame": true,
      "max_references": {"image": 4, "video": 3, "audio": 1},
      "pricing_variants": [
        {
          "billing_unit": "per_second",
          "resolution": "720p",
          "unit_price": "0.1",
          "audio_unit_price": "0.02",
          "currency": "USD"
        }
      ]
    }
  ]
}
```

### 4.3 模型字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 创建任务时传入的公开模型名；可能是管理员映射后的名称 |
| `object` | string | 固定为 `model` |
| `owned_by` | string | 视频模型通常为 `video` |
| `capability_source` | string | `native`、`ctmoai`、`openai_sora`、`custom_json` 或同名模型合并时的 `mixed` |
| `resolutions` | string[] | 当前 Key 已配置价格且上游支持的分辨率交集 |
| `default_resolution` | string | 省略 `resolution` 时使用的默认分辨率 |
| `default_duration` | integer | 省略 `duration` 时使用的默认秒数 |
| `default_aspect_ratio` | string | 省略 `aspect_ratio` 时使用的默认比例；未提供时客户端可使用 `16:9` 作为 UI 回退 |
| `durations` | integer[] | 上游当前支持的时长集合；部分供应商不提供 |
| `aspect_ratios` | string[] | 上游当前支持的比例集合；部分供应商不提供 |
| `generation_modes` | string[] | 例如 `text_to_video`、`image_to_video`、`omni_reference` |
| `qualities` / `default_quality` | string[] / string | 上游支持的质量选项（如果有） |
| `supports_audio` | boolean | 是否支持生成音轨；不返回或为 `false` 时不要传 `audio: true` |
| `supports_guidances` | boolean | 是否支持 `guidances` 参考媒体 |
| `supports_start_frame` | boolean | 是否支持 `start_frame_url` |
| `requires_start_frame` | boolean | 是否必须提供首帧 |
| `supports_end_frame` | boolean | 是否支持 `end_frame_url` |
| `max_references` | object | `image`、`video`、`audio` 三类参考数组的最大数量；缺少某类或为 0 表示不支持 |
| `supports_watermark` | boolean | 是否支持 `watermark`（如果上游提供） |
| `reference_video_multiplier` | number | 参考视频可能带来的价格倍率，仅在配置了倍率时返回 |
| `pricing_variants` | object[] | 价格展示信息，不是最终扣费承诺；最终以任务 `amount` 和 `currency` 为准 |

模型列表会随 Key、分组、账号状态、上游实时目录和管理员定价变化。客户端必须动态读取，不能根据模型名称硬编码能力矩阵。

`pricing_variants.billing_unit` 常见为 `per_second` 或 `per_request`。`unit_price`、`audio_unit_price` 是字符串，避免金额在 JSON 浮点数中的精度损失。

## 5. 上传素材

### 5.1 请求

```bash
curl -i -X POST "$BASE_URL/v1/videos/uploads" \
  -H "Authorization: Bearer $API_KEY" \
  -F "file=@opening.webp" \
  -F "type=images"
```

表单字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `file` | binary | 要上传的媒体文件；必填 |
| `type` | string | 推荐传 `images`、`videos` 或 `audios`。原生适配器可忽略此字段，CTMOAI 等上游可能需要它 |

服务端会把 multipart 请求整体转发给所选上游适配器，不会把媒体内容转成 Base64。

### 5.2 响应

```json
{
  "media_id": "vidmedia_0123456789abcdef0123456789abcdef",
  "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_0123456789abcdef0123456789abcdef/content",
  "type": "UPLOADED",
  "expires_at": "2026-09-10T12:00:00Z"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `media_id` | string | Gavin2API 素材 ID，通常以 `vidmedia_` 开头 |
| `url` | string | 带鉴权的素材读取地址，可放入创建请求；`public_base_url` 未配置时可能是相对路径 |
| `type` | string | 上游归一化后的类型，通常为 `UPLOADED` |
| `expires_at` | RFC 3339 string | 素材过期时间；必须在过期前创建任务 |
| `media_type` | string | 部分上游返回的媒体集合类型，可能出现 |
| `mime_type` | string | 部分上游返回的 MIME 类型，可能出现 |
| `container` | string | 部分上游返回的容器格式，可能出现 |
| `duration_us` | integer | 部分上游返回的媒体时长，单位微秒，可能出现 |

图片、视频和音频的格式、像素、时长和大小限制由当前上游模型决定。工作台当前的建议值（不是所有上游的固定公共契约）是：图片 PNG/JPEG/WebP，视频 MP4/MOV，音频 MP3/WAV。上传成功只代表素材被上游接收，不代表所选模型支持该素材。

### 5.3 AIStartLab 限制

当当前模型只路由到 `capability_source=openai_sora` 的 AIStartLab 账号时，`POST /v1/videos/uploads` 返回 `422 VIDEO_UPLOAD_UNSUPPORTED`。此时应直接在创建请求中传上游可访问的公网素材 URL；不要把 Gavin2API 上传 URL 当成 AIStartLab 的二进制上传能力。

### 5.4 读取上传素材

```bash
curl -fL "$BASE_URL/v1/videos/uploads/$MEDIA_ID/content" \
  -H "Authorization: Bearer $API_KEY" \
  -o reference.webp
```

支持 `Range: bytes=start-end`。素材过期、Key 不匹配或资源不存在时返回 `404 VIDEO_RESOURCE_NOT_FOUND`。

## 6. 创建视频任务

### 6.1 两个等价入口

- `POST /v1/videos`：OpenAI 风格别名。
- `POST /v1/videos/generations`：历史工作台和旧客户端路径。

两条路径创建同一种 Gavin2API 异步任务。建议新客户端使用 `/v1/videos`，但必须始终使用响应里的 `status_url`，不要自行拼接查询路径。

### 6.2 请求头

```http
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
Accept: application/json
Prefer: respond-async
Idempotency-Key: video-order-20260909-0001
```

### 6.3 顶层请求字段

未知顶层字段会返回 `400 VIDEO_REQUEST_INVALID`。可选字段不使用时建议省略，不要传空字符串或 `null`。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 必须是当前 `/v1/models` 返回的 `id` |
| `prompt` | string | 是 | 场景、主体、动作、镜头和风格描述；不能为空 |
| `generation_mode` | string | 否 | 例如 `text_to_video`；由上游模型决定是否支持 |
| `quality` | string | 否 | 由模型能力决定 |
| `resolution` | string | 否 | 例如 `480p`、`720p`；省略时使用定价规则默认值 |
| `duration` | integer | 否 | 正整数秒；省略时使用定价规则默认值 |
| `seconds` | integer | 否 | `duration` 的兼容别名；与 `duration` 同时传时数值必须完全一致 |
| `aspect_ratio` | string | 否 | 例如 `16:9`、`9:16`、`1:1` |
| `audio` | boolean | 否 | 是否生成音轨，默认 `false`；不能传字符串 |
| `watermark` | boolean | 否 | 是否添加水印；仅部分上游支持 |
| `prompt_enhance` | string | 否 | 常见值 `AUTO`、`ON`、`OFF`；仅部分模型支持 |
| `image_url` | string | 否 | 旧兼容首帧/参考图字段；新接入使用 `start_frame_url` |
| `start_frame_url` | string | 否 | 首帧绝对 HTTP(S) URL |
| `end_frame_url` | string | 否 | 尾帧绝对 HTTP(S) URL；多数适配器要求同时提供首帧 |
| `guidances` | object | 否 | 参考图片、视频和音频，见第 7 章 |

`image_url` 和 `start_frame_url` 不能同时传。首尾帧与 `guidances.image_reference` 不能组合；具体模型的支持情况仍以 `/v1/models` 能力字段为准。

### 6.4 文生视频示例

```bash
curl -i -X POST "$BASE_URL/v1/videos" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "Prefer: respond-async" \
  -H "Idempotency-Key: video-order-20260909-0001" \
  -d '{
    "model": "seedance-2.0",
    "prompt": "云海之上的仙侠山门，清晨薄雾，镜头缓慢向前推进，电影级光影",
    "resolution": "480p",
    "duration": 5,
    "aspect_ratio": "16:9",
    "audio": true
  }'
```

### 6.5 首帧/尾帧示例

```json
{
  "model": "seedance-2.0",
  "prompt": "保持人物与服装一致，人物缓慢转身，镜头平稳向前推进",
  "resolution": "720p",
  "duration": 8,
  "aspect_ratio": "16:9",
  "start_frame_url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_image/content",
  "end_frame_url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_end/content"
}
```

### 6.6 创建成功响应

```http
HTTP/1.1 202 Accepted
Preference-Applied: respond-async
Location: /v1/videos/vidjob_0123456789abcdef0123456789abcdef
Content-Type: application/json
```

```json
{
  "job_id": "vidjob_0123456789abcdef0123456789abcdef",
  "status": "pending",
  "status_url": "/v1/videos/vidjob_0123456789abcdef0123456789abcdef"
}
```

`status_url` 和 `Location` 可能是相对路径。使用当前请求的 `BASE_URL` 拼接，不要固定写入其他域名。响应中的任务 ID 只代表 Gavin2API 任务，不是供应商的内部任务 ID。

### 6.7 幂等行为

- 同一 API Key、同一 `Idempotency-Key`、语义相同的请求会返回原任务，不重复创建和扣费。
- 同一 API Key 下同一个幂等键对应不同 JSON 语义时返回 `409 IDEMPOTENCY_KEY_CONFLICT`。
- 相同请求正在被其他请求创建时，可能短暂返回 `503 VIDEO_REQUEST_IN_PROGRESS`；稍后使用相同请求体和幂等键重试。
- 首次请求如果在上游已接收但响应丢失的窗口中超时，不能换新幂等键；必须用原键恢复。

## 7. 参考媒体和生成音轨

### 7.1 `audio` 与音频参考的区别

- `audio: true`：让模型为新生成视频创建音轨；默认 `false`。
- `guidances.audio_reference`：把已有音频作为节奏、动作或风格参考；不等于生成音轨。

### 7.2 `guidances` 结构

```json
{
  "model": "seedance-2.0",
  "prompt": "根据参考人物和鼓点生成连贯舞蹈动作，人物外观保持一致",
  "resolution": "720p",
  "duration": 8,
  "aspect_ratio": "16:9",
  "audio": true,
  "guidances": {
    "image_reference": [
      {
        "image": {
          "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_image/content",
          "type": "UPLOADED"
        },
        "strength": "MID"
      }
    ],
    "video_reference_base": [
      {
        "video": {
          "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_video/content",
          "type": "UPLOADED"
        }
      }
    ],
    "audio_reference": [
      {
        "audio": {
          "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_audio/content",
          "type": "UPLOADED"
        }
      }
    ]
  }
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `image_reference[].image.url` | 图片 URL；推荐使用上传响应的 `url` |
| `image_reference[].image.type` | 推荐固定为 `UPLOADED`；公网 URL 仍需满足上游可访问要求 |
| `image_reference[].strength` | `LOW`、`MID`、`HIGH`；省略或 `AUTO` 会归一化为 `MID` |
| `image_reference[].order` | 可省略；服务端会按数组顺序重新编号，客户端传入值不会作为最终顺序依据 |
| `video_reference_base[].video.url` | 参考视频 URL |
| `audio_reference[].audio.url` | 参考音频 URL |

后端会校验图片 `strength`；其他深层字段由具体适配器和模型校验。数组数量必须不超过模型 `max_references`，不支持的类型不要发送。

### 7.3 组合约束

- 首帧或尾帧不能与 `guidances.image_reference` 同时使用。
- 使用 `end_frame_url` 时，AIStartLab 和 CTMOAI 适配器要求同时有 `start_frame_url`。
- 参考视频或参考音频是否必须同时提供图片，取决于上游；AIStartLab 的能力配置会对此进行校验。
- 不能把同一请求中的平台上传素材混用为不同上游账号的素材。混用时返回 `422 VIDEO_MEDIA_INVALID`。
- 模型不支持音轨时传 `audio: true`，或不支持参考素材却发送 `guidances`，通常返回 `422 VIDEO_OPTION_UNSUPPORTED`。

## 8. 查询任务

### 8.1 查询单个任务

```bash
curl -sS "$BASE_URL/v1/videos/jobs/$JOB_ID" \
  -H "Authorization: Bearer $API_KEY"
```

未进入终态时，查询接口会向上游刷新一次状态；已经是终态时直接返回本地记录。

### 8.2 处理中响应

```json
{
  "job_id": "vidjob_0123456789abcdef0123456789abcdef",
  "status": "running",
  "model": "seedance-2.0",
  "resolution": "720p",
  "duration": 8,
  "aspect_ratio": "16:9",
  "amount": "0.80000000",
  "currency": "USD",
  "created_at": "2026-09-09T12:00:00Z",
  "updated_at": "2026-09-09T12:00:10Z",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef"
}
```

### 8.3 完成响应

```json
{
  "job_id": "vidjob_0123456789abcdef0123456789abcdef",
  "status": "completed",
  "model": "seedance-2.0",
  "resolution": "720p",
  "duration": 8,
  "aspect_ratio": "16:9",
  "amount": "0.80000000",
  "currency": "USD",
  "created_at": "2026-09-09T12:00:00Z",
  "updated_at": "2026-09-09T12:02:30Z",
  "finished_at": "2026-09-09T12:02:30Z",
  "settlement_status": "captured",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef",
  "content_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef/content"
}
```

### 8.4 失败响应

```json
{
  "job_id": "vidjob_0123456789abcdef0123456789abcdef",
  "status": "failed",
  "model": "seedance-2.0",
  "resolution": "720p",
  "duration": 8,
  "aspect_ratio": "16:9",
  "amount": "0.80000000",
  "currency": "USD",
  "created_at": "2026-09-09T12:00:00Z",
  "updated_at": "2026-09-09T12:01:00Z",
  "finished_at": "2026-09-09T12:01:00Z",
  "settlement_status": "released",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef",
  "error": {
    "code": "VIDEO_GENERATION_FAILED",
    "message": "resolution is not supported by this model",
    "stage": "processing",
    "task_id": "vidjob_0123456789abcdef0123456789abcdef",
    "failed_at": "2026-09-09T12:01:00Z",
    "upstream_code": "VIDEO_RESOLUTION_INVALID",
    "request_id": "req_0123456789abcdef"
  }
}
```

失败诊断字段经过白名单和脱敏处理，不会返回上游账号、上游凭证、内部 URL、上游任务 ID 或原始错误正文。

### 8.5 任务字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `job_id` | string | Gavin2API 任务 ID |
| `status` | string | `pending`、`running`、`settling`、`completed`、`failed`、`canceled` |
| `model` | string | 下游请求使用的公开模型名 |
| `resolution` | string | 平台最终解析的分辨率 |
| `duration` | integer | 平台最终解析的时长，单位秒 |
| `aspect_ratio` | string | 平台最终解析的比例 |
| `amount` | string | 对下游计费金额，固定 8 位小数；按 Decimal/定点数处理 |
| `currency` | string | 币种，通常为 `USD` |
| `created_at` / `updated_at` | RFC 3339 string | 创建和最近更新时间 |
| `finished_at` | RFC 3339 string | 进入终态的时间，未结束时不返回 |
| `settlement_status` | string | `held`、`captured` 或 `released` |
| `status_url` | string | 相对查询路径 |
| `content_url` | string | 仅 `completed` 时返回 |
| `error` | object | 仅 `failed` 时返回，见上例 |

## 9. 任务列表、取消和下载

### 9.1 查询最近任务

```bash
curl -sS "$BASE_URL/v1/videos/jobs?limit=20" \
  -H "Authorization: Bearer $API_KEY"
```

响应为：

```json
{
  "object": "list",
  "data": [
    {
      "job_id": "vidjob_xxx",
      "status": "completed",
      "model": "seedance-2.0",
      "resolution": "720p",
      "duration": 8,
      "aspect_ratio": "16:9",
      "amount": "0.80000000",
      "currency": "USD",
      "created_at": "2026-09-09T12:00:00Z",
      "updated_at": "2026-09-09T12:02:30Z",
      "status_url": "/v1/videos/jobs/vidjob_xxx",
      "content_url": "/v1/videos/jobs/vidjob_xxx/content"
    }
  ]
}
```

`limit` 省略或小于等于 0 时为 20，最大为 100。当前没有 cursor 或 offset 分页，结果按创建时间倒序。

### 9.2 取消任务

```bash
curl -i -X DELETE "$BASE_URL/v1/videos/jobs/$JOB_ID" \
  -H "Authorization: Bearer $API_KEY"
```

只有 `pending` 和 `running` 且当前上游适配器提供取消端点时可以取消。成功返回最新任务对象，取消请求没有立即携带明确终态时，继续低频查询。

以下情况返回 `409 VIDEO_JOB_NOT_CANCELABLE`：任务已进入 `settling`、已经是 `completed`/`failed`/`canceled`，或当前上游适配器不支持取消。

### 9.3 下载成品

```bash
curl -fL "$BASE_URL/v1/videos/jobs/$JOB_ID/content?download=1" \
  -H "Authorization: Bearer $API_KEY" \
  -o result.mp4
```

只有 `completed` 任务可以下载，其他状态返回 `404 VIDEO_RESOURCE_NOT_FOUND`。支持断点读取：

```bash
curl -i "$BASE_URL/v1/videos/jobs/$JOB_ID/content" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Range: bytes=0-1048575"
```

上游支持范围读取时返回 `206 Partial Content`。成品是否长期保留取决于平台上游和可选 OSS 配置，接口不承诺固定保存天数；完成后请及时转存。

浏览器原生 `<video src>` 无法附加 Bearer 头。浏览器端应由可信后端代理，或先使用 `fetch` 携带 Key 读取 Blob，再创建临时对象 URL；不要把 Key 拼到 query string。

## 10. 计费和结算

统一异步 API 的计费生命周期：

1. 创建前按照最终模型、分辨率、时长、音轨开关和价格规则计算预冻结金额。
2. 余额不足返回 `402 INSUFFICIENT_BALANCE`，不会调用上游创建。
3. 任务创建时余额从可用余额转入冻结余额，任务状态为 `held`。
4. `completed` 捕获冻结金额并将结算状态置为 `captured`。
5. `failed` 或 `canceled` 释放冻结金额并将结算状态置为 `released`。

常见价格形式：

```text
per_second: 任务金额 = duration × (unit_price + audio_unit_price 当 audio=true)
per_request: 任务金额 = unit_price（如平台规则配置了音频附加价，再按平台规则叠加）
```

参考视频可能应用 `reference_video_multiplier`。由于模型、账号和定价可动态变化，下游不要自行预估或重复扣费，最终金额始终以任务返回的 `amount` 和 `currency` 为准。`amount` 是下游售价，不是上游供应商成本。

## 11. 状态机和轮询策略

| 状态 | 终态 | 下游动作 |
| --- | --- | --- |
| `pending` | 否 | 继续轮询；通常可取消 |
| `running` | 否 | 继续轮询；通常可取消 |
| `settling` | 否 | 继续轮询；不可取消 |
| `completed` | 是 | 下载 `content_url`，记录金额 |
| `failed` | 是 | 读取 `error`；如需重做，生成新的幂等键 |
| `canceled` | 是 | 停止轮询；冻结金额已释放 |

建议：首次查询间隔约 5 秒，随后逐步使用 10、15、30 秒；设置业务整体超时。查询接口遇到网络错误、429 或 5xx 时，不要立即把任务标成失败，继续退避重试。任务进入 `failed` 后不能用原任务 ID 重新生成。

## 12. 错误码和处理建议

| HTTP | 错误码 | 含义 | 建议 |
| ---: | --- | --- | --- |
| 400 | `VIDEO_REQUEST_INVALID` | JSON 无效、未知字段、字段类型错误、URL 无效或 Content-Type 错误 | 修正请求，不要原样重试 |
| 400 | `IDEMPOTENCY_KEY_INVALID` | 创建缺少幂等键、长度超限或含不可打印字符 | 修正幂等键 |
| 401 | `API_KEY_REQUIRED` 或鉴权错误 | 缺少、无效或失效的 Key | 检查服务端密钥配置 |
| 402 | `INSUFFICIENT_BALANCE` | 余额不足以预冻结任务金额 | 充值或降低规格后使用新幂等键 |
| 403 | `VIDEO_GENERATION_DISABLED` | 当前 Key 未绑定启用的视频分组 | 联系平台方开通或更换 Key |
| 404 | `VIDEO_RESOURCE_NOT_FOUND` | 资源不存在、已过期、Key 不匹配或成品尚未可读 | 核对 ID、Key 和任务状态 |
| 409 | `IDEMPOTENCY_KEY_CONFLICT` | 同一 Key 下幂等键对应不同请求 | 修正业务幂等逻辑 |
| 409 | `VIDEO_JOB_NOT_CANCELABLE` | 任务已进入不可取消状态或上游不支持取消 | 刷新状态，不要重复取消 |
| 422 | `VIDEO_MODEL_INVALID` | 模型不存在或当前 Key 不可用 | 刷新 `/v1/models` |
| 422 | `VIDEO_PROMPT_INVALID` | 提示词不符合模型要求 | 修正提示词 |
| 422 | `VIDEO_RESOLUTION_INVALID` | 分辨率不支持 | 使用模型返回的 `resolutions` |
| 422 | `VIDEO_DURATION_INVALID` | 时长不支持或非正整数 | 使用模型返回的 `durations`/默认时长 |
| 422 | `VIDEO_ASPECT_RATIO_INVALID` | 比例不支持 | 使用模型返回的 `aspect_ratios` |
| 422 | `VIDEO_MEDIA_INVALID` | 素材无效、过期、跨 Key、跨上游账号或组合不合法 | 重新上传并检查模型限制 |
| 422 | `VIDEO_UPLOAD_UNSUPPORTED` | 所选上游只接受公网 URL，不支持平台二进制上传 | 改用公网素材 URL |
| 422 | `VIDEO_REFERENCE_IMAGE_STRENGTH_INVALID` | `strength` 不是 `LOW`、`MID` 或 `HIGH` | 修正字段 |
| 422 | `VIDEO_OPTION_UNSUPPORTED` | 模型不支持某个选项、参考类型或组合 | 按模型能力移除/调整字段 |
| 422 | `VIDEO_PROMPT_ASPECT_RATIO_CONFLICT` | 提示词中声明的比例和 `aspect_ratio` 冲突 | 统一提示词和结构化参数 |
| 422 | `VIDEO_PROMPT_DURATION_CONFLICT` | 提示词中声明的时长和 `duration` 冲突 | 统一提示词和结构化参数 |
| 429 | `VIDEO_CAPACITY_EXHAUSTED` | 当前账号并发或上游容量暂时耗尽 | 按 `Retry-After` 或指数退避重试 |
| 503 | `VIDEO_UPSTREAM_FORBIDDEN` | 上游拒绝当前模型或凭证权限 | 联系平台方检查账号和模型授权 |
| 503 | `VIDEO_REQUEST_IN_PROGRESS` | 相同幂等请求仍在创建 | 保留原请求和幂等键稍后重试 |
| 503 | `VIDEO_EXECUTION_DISABLED` | 平台视频执行总开关关闭 | 联系平台方确认后再重试 |
| 503 | `VIDEO_PRICING_UNAVAILABLE` | 当前模型/分辨率没有有效价格 | 联系平台方或换规格 |
| 503 | `VIDEO_UPSTREAM_UNAVAILABLE` | 上游网络、凭证或服务不可用 | 保留原请求和幂等键退避重试 |
| 503 | `VIDEO_UPSTREAM_ERROR` | 已脱敏的其他上游错误 | 仅在 5xx 或平台建议时重试 |

上游的 401、402、403 和大多数 5xx 会被网关转换为 503，避免泄漏上游凭证和账号状态。任务进入 `failed` 后，查询响应使用稳定的 `error.code=VIDEO_GENERATION_FAILED`，并可能在 `error.upstream_code` 中提供经过白名单过滤的上游代码。

## 13. 重试、安全和上线检查

### 13.1 创建请求重试

- 建连失败、读取超时、连接重置、429 或 5xx：使用相同 JSON 和相同 `Idempotency-Key`。
- 推荐退避：2、4、8、16、30 秒并加入随机抖动；存在 `Retry-After` 时优先遵守。
- 400、401、402、403、422 和幂等冲突不要自动原样重试。
- 获得 `202` 后不要再次创建，只轮询 `status_url`。
- 记录 HTTP 状态、`error.code`、`X-Client-Request-ID`、`X-Request-Id`（如有）和发生时间，不记录完整 Key。

### 13.2 上线前检查

- [ ] Base URL 使用 HTTPS，且没有重复拼接 `/v1`。
- [ ] API Key 已绑定启用的视频分组。
- [ ] 启动时或定时刷新 `/v1/models`，没有写死模型授权。
- [ ] API Key 只保存在服务端密钥系统。
- [ ] 每次业务创建使用唯一、稳定的 `Idempotency-Key`。
- [ ] 已实现所有任务状态和 `failed.error` 解析。
- [ ] 轮询有退避和整体超时，完成后及时转存媒体。
- [ ] 金额使用 Decimal/定点数，不用二进制浮点累计。
- [ ] 查询、取消、播放、下载使用创建任务的同一个 API Key。
- [ ] 已验证余额不足、容量不足、内容审核拒绝、素材过期、任务失败和 Range 下载。

### 13.3 联调故障信息

请提供：请求时间和时区、方法和路径、HTTP 状态、`error.code`、请求追踪头、`job_id`/`media_id`、模型、分辨率、时长、比例，以及 API Key 末 4 位。不要提供完整 Key、上游凭证、原始私密素材或包含敏感信息的完整请求日志。

## 14. 服务端调用示例

以下示例只展示关键流程，API Key 应从服务端密钥管理系统读取。

### 14.1 Python

```python
import os
import random
import time
import uuid
from decimal import Decimal
from pathlib import Path

import requests


BASE_URL = os.environ["GAVIN2API_BASE_URL"].rstrip("/")
API_KEY = os.environ["GAVIN2API_KEY"]
AUTH = {"Authorization": f"Bearer {API_KEY}"}


def raise_api_error(response: requests.Response) -> None:
    try:
        payload = response.json()
        error = payload.get("error", {})
        code = error.get("code", "UNKNOWN_ERROR")
        message = error.get("message", "request failed")
    except ValueError:
        code, message = "NON_JSON_ERROR", response.text[:500]
    request_id = response.headers.get("X-Request-Id") or response.headers.get("X-Client-Request-ID")
    raise RuntimeError(f"HTTP {response.status_code} {code}: {message}; request_id={request_id}")


def create_video() -> dict:
    body = {
        "model": "seedance-2.0",
        "prompt": "云海之上的仙侠山门，镜头缓慢向前推进",
        "resolution": "480p",
        "duration": 5,
        "aspect_ratio": "16:9",
        "audio": True,
    }
    idempotency_key = f"video-{uuid.uuid4()}"
    headers = {
        **AUTH,
        "Content-Type": "application/json",
        "Prefer": "respond-async",
        "Idempotency-Key": idempotency_key,
    }
    for attempt in range(6):
        try:
            response = requests.post(
                f"{BASE_URL}/v1/videos",
                headers=headers,
                json=body,
                timeout=(10, 45),
            )
        except (requests.Timeout, requests.ConnectionError):
            response = None
        if response is not None and response.status_code == 202:
            return response.json()
        if response is not None and response.status_code < 500 and response.status_code != 429:
            raise_api_error(response)
        if attempt == 5:
            if response is not None:
                raise_api_error(response)
            raise RuntimeError("video creation timed out")
        retry_after = response.headers.get("Retry-After") if response is not None else None
        delay = int(retry_after) if retry_after and retry_after.isdigit() else min(2 ** (attempt + 1), 30)
        time.sleep(delay + random.random())
    raise RuntimeError("unreachable")


def wait_for_video(job_id: str) -> dict:
    delay = 5
    while True:
        response = requests.get(f"{BASE_URL}/v1/videos/jobs/{job_id}", headers=AUTH, timeout=(10, 45))
        if response.status_code != 200:
            raise_api_error(response)
        job = response.json()
        if job["status"] == "completed":
            print("charged:", Decimal(job["amount"]), job["currency"])
            return job
        if job["status"] in {"failed", "canceled"}:
            raise RuntimeError(f"video ended: {job.get('error')}")
        time.sleep(delay)
        delay = min(delay + 5, 30)


def download_video(content_url: str, output: Path) -> None:
    with requests.get(f"{BASE_URL}{content_url}", headers=AUTH, stream=True, timeout=(10, 300)) as response:
        if response.status_code not in (200, 206):
            raise_api_error(response)
        with output.open("wb") as file:
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if chunk:
                    file.write(chunk)


created = create_video()
job = wait_for_video(created["job_id"])
download_video(job["content_url"], Path("result.mp4"))
```

### 14.2 TypeScript / Node.js 18+

```ts
import { randomUUID } from 'node:crypto'
import { writeFile } from 'node:fs/promises'

const baseUrl = process.env.GAVIN2API_BASE_URL!.replace(/\/$/, '')
const apiKey = process.env.GAVIN2API_KEY!
const auth = { Authorization: `Bearer ${apiKey}` }

async function readError(response: Response): Promise<Error> {
  const text = await response.text()
  try {
    const body = JSON.parse(text)
    return new Error(`HTTP ${response.status} ${body.error?.code ?? 'UNKNOWN_ERROR'}: ${body.error?.message ?? ''}`)
  } catch {
    return new Error(`HTTP ${response.status}: ${text.slice(0, 500)}`)
  }
}

const create = await fetch(`${baseUrl}/v1/videos`, {
  method: 'POST',
  headers: {
    ...auth,
    'Content-Type': 'application/json',
    Prefer: 'respond-async',
    'Idempotency-Key': `video-${randomUUID()}`,
  },
  body: JSON.stringify({
    model: 'seedance-2.0',
    prompt: '云海之上的仙侠山门，镜头缓慢向前推进',
    resolution: '480p',
    duration: 5,
    aspect_ratio: '16:9',
    audio: true,
  }),
})
if (create.status !== 202) throw await readError(create)
const accepted = await create.json() as { job_id: string }

let delayMs = 5_000
let job: any
for (;;) {
  const response = await fetch(`${baseUrl}/v1/videos/jobs/${encodeURIComponent(accepted.job_id)}`, { headers: auth })
  if (!response.ok) throw await readError(response)
  job = await response.json()
  if (job.status === 'completed') break
  if (job.status === 'failed' || job.status === 'canceled') throw new Error(JSON.stringify(job.error))
  await new Promise(resolve => setTimeout(resolve, delayMs))
  delayMs = Math.min(delayMs + 5_000, 30_000)
}

const content = await fetch(`${baseUrl}${job.content_url}`, { headers: auth })
if (!content.ok) throw await readError(content)
await writeFile('result.mp4', Buffer.from(await content.arrayBuffer()))
```
