# Gavin2API 统一视频 API 对接文档

本文档面向需要接入 Gavin2API 视频能力的客户。平台会根据 API Key 所属分组和后台配置，自动选择 Seedance、MiniMax H3、Stable Video Diffusion 或其他已配置的上游；客户不需要感知上游的 URL、协议和字段差异。

## 1. 地址与认证

```text
Base URL: https://api.gavinteam.online
```

请求统一使用 Bearer Token：

```http
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
Idempotency-Key: your-stable-request-id
```

API Key 只放在服务端，不要写入浏览器源码、URL、日志或错误截图。

## 2. 模型发现与能力

先调用模型列表，按照返回的能力字段构造表单或请求：

```bash
curl -sS "$BASE_URL/v1/models" \
  -H "Authorization: Bearer $API_KEY"
```

重点字段：

| 字段 | 含义 |
| --- | --- |
| `id` | 对外模型名，创建任务时传入 |
| `resolutions` | 可用分辨率，例如 `768p`、`2K`、`4K` |
| `durations` | 模型支持的秒数；没有该字段时使用 `default_duration` |
| `aspect_ratios` | 模型支持的比例 |
| `supports_guidances` | 是否支持全能参考 |
| `supports_start_frame` / `supports_end_frame` | 是否支持首帧/尾帧 |
| `requires_start_frame` | 是否必须提供首帧 |
| `max_references` | 图片、视频、音频参考数量上限 |
| `generation_modes` | 支持的生成方式，如 `text_to_video`、`omni_reference` |

客户端应以这些字段动态显示选项。某个模型不支持全能参考时，不要发送 `guidances`，并将“全能参考”按钮置灰或隐藏。

能力字段以当前 Key 调用 `GET /v1/models` 的实时返回为准，不要用模型名称推断统一限制。例如当前 CTMOAI 返回的 `minimax-h3-quantized-768p` 只支持 4–10 秒，最多 4 张参考图片，参考视频和参考音频上限均为 0；普通 H3 模型可能返回 4–15 秒并支持三类参考素材。工作台会按这些字段动态隐藏或限制选项。

## 3. 创建视频任务

推荐使用 OpenAI 风格入口：

```http
POST /v1/videos
```

现有工作台入口仍然兼容：

```http
POST /v1/videos/generations
```

两条入口创建的是同一种异步任务。请求示例：

```json
{
  "model": "minimax-h3-original-768p",
  "prompt": "A slow cinematic shot of waves at sunset",
  "resolution": "768p",
  "duration": 6,
  "aspect_ratio": "16:9"
}
```

`duration` 是统一字段。为兼容 CTMOAI 等上游文档，`seconds` 也可以使用；如果同时提供两个字段，数值必须一致。

参考图任务可使用统一媒体字段：

```json
{
  "model": "minimax-h3-original-cf-2k",
  "prompt": "Animate the subject naturally",
  "duration": 6,
  "aspect_ratio": "16:9",
  "start_frame_url": "https://your-domain.example/reference.jpg"
}
```

也可以使用 `guidances.image_reference`、`guidances.video_reference_base` 和 `guidances.audio_reference`。CF（参考图）模型必须有图片参考；普通 H3 768p/1080p 模型可以纯文生。量化版 H3 的参考视频和音频不受支持，不能发送对应数组。

成功返回 `202 Accepted`：

```json
{
  "job_id": "vidjob_xxx",
  "status": "pending",
  "status_url": "/v1/videos/vidjob_xxx"
}
```

建议每 5 秒轮询一次，长任务逐步放宽到 30 秒。必须保存 `Idempotency-Key`，网络重试时复用同一个值，避免重复扣费。

## 4. 查询、下载和取消

### 查询任务

```http
GET /v1/videos/{job_id}
```

工作台和旧版客户端也可以使用：

```http
GET /v1/videos/jobs/{job_id}
```

状态包括 `pending`、`running`、`completed`、`failed`、`canceled`。查询接口遇到 408、425、429、500、502、503、504 时应继续重试，不要直接把任务标为失败。

### 下载成品

```http
GET /v1/videos/{job_id}/content
Range: bytes=0-
```

也兼容 `/v1/videos/jobs/{job_id}/content`。接口返回 `video/mp4` 或上游实际媒体类型，并支持 Range 断点读取。

### 取消任务

```http
DELETE /v1/videos/jobs/{job_id}
```

是否能取消取决于上游协议；不支持时返回 `VIDEO_JOB_NOT_CANCELABLE`。

## 5. 错误处理

错误统一返回：

```json
{
  "error": {
    "code": "VIDEO_OPTION_UNSUPPORTED",
    "message": "the selected video upstream does not support this video option"
  }
}
```

常见处理：

| code | 处理 |
| --- | --- |
| `VIDEO_REQUEST_INVALID` | 修正字段类型或必填字段 |
| `VIDEO_OPTION_UNSUPPORTED` | 根据 `/v1/models` 能力字段隐藏不支持的选项 |
| `VIDEO_CAPACITY_EXHAUSTED` | 延迟后重试或切换模型 |
| `VIDEO_UPSTREAM_UNAVAILABLE` | 按退避策略重试，不要立即判定任务失败 |
| `INSUFFICIENT_BALANCE` | 提示充值，不要重试 |
| `VIDEO_GENERATION_FAILED` | 读取安全的 `upstream_code`，修正参数后重新创建 |

## 6. 上游扩展约定

新增上游只需要在后台新增视频账号，选择对应协议适配器，配置模型映射、能力和售价。对外 API 不变：模型发现仍走 `/v1/models`，创建仍走 `/v1/videos`，状态和成品仍走统一任务接口。上游的 `seconds`、任务状态名、参考图字段、鉴权头和结果下载 URL 均由服务端适配器转换。

当前 CTMOAI 文档可参考：[MiniMax H3](https://video.ctmoai.com/docs/minimax-h3)、[Special Stable](https://video.ctmoai.com/docs/special-stable)。
