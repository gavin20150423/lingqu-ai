# Gavin2API 视频生成 API 接入文档

本文档面向调用 Gavin2API 视频生成能力的下游服务，说明鉴权、模型发现、素材上传、异步任务创建、状态查询、取消、成品下载、计费、幂等与错误重试。

> 文档协议版本：v1.1
> 推荐协议：统一异步任务 API（第三方视频 API 分组）
> Base URL：`https://api.gavinteam.online`

## 1. 接入前确认

平台方需要向下游提供：

- `Base URL`，不包含结尾的 `/v1`；
- API Key；
- API Key 所属的视频协议类型；
- 已开通模型、余额和并发限制。

Gavin2API 当前存在两种视频协议。二者共用创建路径，但任务响应和查询路径不同，不能混用。

| 协议 | 适用 API Key 分组 | 创建结果 | 查询路径 | 建议用途 |
|---|---|---|---|---|
| 统一异步任务 API | 第三方视频 API 分组 | `job_id`，前缀为 `vidjob_` | `/v1/videos/jobs/{job_id}` | 推荐给普通下游，支持幂等、任务列表和统一计费；素材上传与取消能力取决于上游协议 |
| Grok 原生兼容 API | Grok 分组 | 上游 `request_id` | `/v1/videos/{request_id}` | 已按 Grok/xAI 视频协议开发的客户端 |

除第 14 节外，本文档均描述推荐的“统一异步任务 API”。如果不确定 Key 属于哪种协议，请在联调前向平台方确认。

### 1.1 平台部署配置

统一异步视频 API 的总开关不在普通业务页面中，需要由部署管理员修改 `config.yaml`（示例文件为 `deploy/config.example.yaml`）或通过环境变量配置。配置完成后重启或按部署方式重新加载服务。

最小可用配置：

```yaml
video_api:
  enabled: true
  # 必须是下游能够访问的 Gavin2API 外部地址，不带结尾斜杠
  public_base_url: "https://api.gavinteam.online"
  request_timeout_seconds: 30
  reconcile_interval_seconds: 30
  reconcile_batch_size: 20
```

| 配置项 | 环境变量 | 默认值 | 说明 |
|---|---|---:|---|
| `video_api.enabled` | `VIDEO_API_ENABLED` | `false` | 视频 API 总开关；为 `false` 时不会执行视频任务 |
| `video_api.public_base_url` | `VIDEO_API_PUBLIC_BASE_URL` | 空 | 上传素材响应中的 URL 前缀；必须是 HTTP(S) URL，不能带 query 或 fragment |
| `video_api.request_timeout_seconds` | `VIDEO_API_REQUEST_TIMEOUT_SECONDS` | `30` | Gavin2API 请求上游的 HTTP 超时秒数 |
| `video_api.reconcile_interval_seconds` | `VIDEO_API_RECONCILE_INTERVAL_SECONDS` | `30` | 后台刷新在途任务的间隔；小于 5 秒时实际按 30 秒运行 |
| `video_api.reconcile_batch_size` | `VIDEO_API_RECONCILE_BATCH_SIZE` | `20` | 每轮后台最多刷新在途任务数 |

`public_base_url` 必须和下游实际访问的域名一致。本平台地址为 `https://api.gavinteam.online`，不要填写内网地址、容器地址或带 `/v1` 的地址。服务会自动拼接 `/v1/videos/...`。

### 1.2 管理后台配置顺序

完成部署配置后，管理员按以下顺序操作：

1. 在 **视频账号配置** 页面新增一个第三方视频上游账号；
2. 在 **分组管理** 页面创建视频平台分组，并保持分组为启用状态；
3. 将上游账号绑定到该分组；
4. 在账号配置中填写模型映射和视频销售价格；
5. 创建或选择一个 API Key，将它绑定到该视频分组；
6. 使用该 API Key 调用 `GET /v1/models`，确认模型列表和分辨率已经出现。

只完成账号配置而没有绑定分组，或只绑定分组而没有有效 API Key，均不能对下游提供视频能力。

### 1.3 视频账号配置页面

后台入口：**视频账号配置**（当前前端路由为 `/admin/xiao-video`）。页面用于管理第三方视频供应商账号，不是下游 API 的地址。

点击“添加视频上游”后，填写：

| 页面字段 | 必填 | 配置要求 |
|---|---|---|
| 上游名称 | 是 | 管理员自定义，例如 `供应商 A - 视频` |
| 上游协议 | 是 | 普通现有上游选择“原生 XiaoAPI 视频协议”；AIStartLab 选择“AIStartLab（OpenAI / Sora 兼容）” |
| 上游 Base URL | 是 | HTTP(S) 地址；不能带 query 或 fragment；可按供应商要求带 `/v1` |
| 上游 API Key | 是 | 供应商分配的凭证；编辑已有账号时留空表示保留旧值 |
| 状态 | 是 | `active` 才会参与调度；`inactive`/`error` 不参与 |
| 并发数 | 是 | 整数 `1-10000`；应不超过供应商给该 Key 的并发上限 |
| 备注 | 否 | 用于记录供应商、区域、用途等运维信息 |
| 绑定的分组 | 是 | 至少绑定一个启用的视频平台分组 |

保存后建议立即点击“测试连接”。测试成功只代表凭证和 Base URL 可访问，模型是否可售还取决于后面的模型映射和售价配置。

#### AIStartLab 配置

AIStartLab 使用飞书文档中的 OpenAI / Sora 兼容协议接入，后台填写：

| 页面字段 | 值 |
|---|---|
| 上游名称 | `AIStartLab`，或便于运维识别的名称 |
| 上游协议 | `AIStartLab（OpenAI / Sora 兼容）` |
| 上游 Base URL | `https://api.video.aistarslab.com/openai` |
| 上游 API Key | 在 AIStartLab 官网创建并妥善保存的新 Key |

AIStartLab 模型 ID 必须使用其 `GET /openai/v1/models` 返回的完整值，格式为 `{channelCode}:{model}`。建议把稳定的对外模型名映射到该完整 ID，例如：

```json
{
  "video-standard": "12:provider-video-model"
}
```

不要写死文档示例里的线路或模型 ID；AIStartLab 会动态调整模型、线路和价格。保存账号后点击“测试连接”，再通过 Gavin2API 的 `GET /v1/models` 确认已配置售价的模型已经出现。

AIStartLab 兼容协议的能力边界：

- 下游继续使用本文档的统一创建、查询、列表和下载接口；Gavin2API 会转换创建字段及 `queued` / `in_progress` 状态；
- AIStartLab 只接受公网可访问的 HTTP(S) 素材 URL，不支持 Gavin2API 将二进制素材上传到该上游；当分组内只有此类账号时，`POST /v1/videos/uploads` 返回 `422 VIDEO_UPLOAD_UNSUPPORTED`；
- AIStartLab 兼容接口没有取消端点，`DELETE /v1/videos/jobs/{job_id}` 返回 `409 VIDEO_JOB_NOT_CANCELABLE`；
- AIStartLab 兼容接口不返回上游积分消耗；下游扣费仍严格按本页面配置的销售价格计算。

### 1.4 模型映射配置

模型映射位于同一账号页面的“模型映射与销售价格”区域，用于把下游看到的公开模型名映射到供应商实际模型名：

```json
{
  "model_mapping": {
    "video-standard": "seedance-2.0",
    "video-fast-*": "seedance-2.0-fast"
  }
}
```

规则：

- 左侧“对外模型”是下游传入 `model` 的值；右侧“上游模型”是实际发送给供应商的值；
- 映射是可选的。没有映射时，平台会尝试按相同模型名转发；
- 对外模型允许通配符，但通配符只能在末尾出现一次，例如 `video-fast-*`；
- 上游模型不能包含通配符；
- 对外模型不能重复映射；
- 下游应使用 `/v1/models` 返回的对外模型名，而不是凭猜测使用供应商内部模型名。

### 1.5 视频销售价格配置

价格也位于账号页面的“模型映射与销售价格”区域，每条规则对应一个“公开模型 + 分辨率”。至少配置一条有效规则，否则该账号不会出现在可用视频模型中。

| 页面字段 | 类型/范围 | 计费含义 |
|---|---|---|
| 对外模型 | string，非空，最长 128 | 下游请求的 `model` |
| 分辨率 | string，非空，最长 64 | 例如 `480p`、`720p`、`1080p` |
| 每秒售价 | number，`>= 0` | 无音轨时每秒金额 |
| 音频每秒附加价 | number，`>= 0` | `audio: true` 时额外加到每秒单价 |
| 默认秒数 | 整数，页面要求 `1-3600` | 下游省略 `duration` 时使用 |
| 默认 | 每个模型最多一个 | 下游省略 `resolution` 时选择该规则 |

平台计算公式：

```text
任务金额 = duration × (price_per_second + (audio ? audio_price_per_second : 0))
```

同一模型配置多个分辨率时，必须且只能选择一个“默认”分辨率。同一“公开模型 + 分辨率”不能重复定价。价格可以配置为 `0`，表示该规格不向下游收费，但仍会执行供应商账号和余额相关流程。

页面保存的账号凭据结构等价于：

```json
{
  "base_url": "https://provider.example.com/v1",
  "api_key": "provider-secret",
  "model_mapping": {
    "video-standard": "seedance-2.0"
  },
  "video_pricing": [
    {
      "model": "video-standard",
      "resolution": "720p",
      "price_per_second": 0.10,
      "audio_price_per_second": 0.02,
      "default_resolution": true,
      "default_duration": 5
    }
  ]
}
```

不要把上面的 `api_key` 写入文档、代码仓库或前端。示例中的凭据仅用于说明字段名称。

### 1.6 分组配置页面

后台入口：**分组管理**（`/admin/groups`）。创建分组时：

| 字段 | 建议 |
|---|---|
| 平台 | 选择视频平台对应的“第三方 API”选项；不要选择 Grok 原生视频分组 |
| 状态 | `active` |
| 订阅类型 | 按你的套餐策略选择 `standard` 或 `subscription` |
| 费率倍率 | 通常从 `1.0` 开始；这是分组通用倍率，第三方异步视频的核心售价仍以账号价格规则为基础 |
| 专属分组 | 需要限制用户范围时开启，并给目标用户授权 |
| 日/周/月额度 | 按业务需要设置美元额度；留空表示不限制该维度 |

第三方异步视频的核心售价配置在“视频账号配置”页面，不依赖分组页面中仅针对 Grok 原生视频的 480p/720p/1080p 价格输入框。不要把两套计费配置混用。

### 1.7 API Key 和用户授权

API Key 必须绑定到上一步创建的启用分组：

- 用户自助场景：用户在用户端 **API Key** 页面创建 Key，并选择/使用管理员开放的视频分组；
- 管理员场景：进入 **用户管理**，在目标用户操作菜单中打开“API Keys”，将 Key 的分组切换为视频分组；
- 一个 Key 只能使用它所属分组允许的账号和模型；
- Key 未绑定分组、分组已禁用、用户没有专属分组授权时，创建请求不会进入视频调度。

余额必须大于任务预冻结金额。创建请求使用的 Key 也决定了任务、素材和扣费记录的归属。

### 1.8 配置完成后的验证清单

```text
[ ] video_api.enabled=true
[ ] video_api.public_base_url 是下游实际访问的 HTTPS 域名
[ ] 至少一个 active 的第三方视频上游账号
[ ] 上游账号有有效 base_url 和 api_key
[ ] 上游账号已绑定 active 视频分组
[ ] 每个公开模型至少一条售价规则
[ ] 多分辨率模型恰好一个默认分辨率
[ ] 下游 API Key 已绑定该视频分组且有余额
[ ] GET /v1/models 返回预期模型
[ ] POST /v1/videos/generations 返回 202
```

## 2. 基础约定

### 2.1 鉴权

所有接口都使用 Bearer Token：

```http
Authorization: Bearer YOUR_API_KEY
```

API Key 应只保存在服务端。不要把 Key 放入 URL、前端源码、公开日志或错误截图。

### 2.2 通用地址

```text
BASE_URL=https://api.gavinteam.online
```

本文档只使用带 `/v1` 的规范路径。调用时不要重复拼接 `/v1`。

### 2.3 资源隔离

上传素材和视频任务均按“创建资源的 API Key”隔离：

- 查询、取消和下载必须使用创建任务时的同一个 API Key；
- 同一用户的另一个 API Key 也不能读取该资源；
- 无权访问与资源不存在统一返回 `404`，下游不能据此判断资源是否属于其他 Key。

### 2.4 内容类型

- JSON 接口使用 `Content-Type: application/json`；允许附带 `charset=utf-8`；
- 素材上传使用带 boundary 的 `multipart/form-data`；使用 `curl -F` 或 SDK 的 multipart 能力即可自动生成；
- 成品和素材下载接口返回二进制流，并透传 `Content-Type`、`Content-Length`、`Content-Range`、`Accept-Ranges`、`Content-Disposition`、`Cache-Control`、`ETag` 和 `Last-Modified` 等安全响应头。

## 3. 推荐调用流程

1. 调用 `GET /v1/models` 获取当前 API Key 实际可用的视频模型和分辨率。
2. 文生视频可直接创建任务；XiaoAPI 的图生视频或参考媒体场景先调用上传接口，AIStartLab 则直接传公网 HTTP(S) 素材 URL。
3. 调用 `POST /v1/videos/generations`，同时携带 `Prefer: respond-async` 和唯一的 `Idempotency-Key`。
4. 保存响应中的 `job_id`，按照 `status_url` 轮询。
5. 状态为 `completed` 后，通过 `content_url` 下载成品。
6. 未启用平台 OSS 持久化时，将成品转存到自己的对象存储；管理员可以在系统设置中为指定用户开启私有 OSS 保存，开启后的新任务由平台保存并继续通过鉴权接口访问。

建议从 5 秒轮询间隔开始，长任务逐步退避到 30 秒。不要高频固定轮询。

## 4. 接口速查

| 方法 | 路径 | 用途 | 成功状态 |
|---|---|---|---|
| `GET` | `/v1/models` | 查询当前 Key 可用模型 | `200` |
| `POST` | `/v1/videos/uploads` | 上传图片、视频或音频素材 | `201` |
| `GET` | `/v1/videos/uploads/{media_id}/content` | 读取已上传素材 | `200` / `206` |
| `POST` | `/v1/videos/generations` | 创建异步视频任务 | `202` |
| `GET` | `/v1/videos/jobs?limit=20` | 查询最近任务 | `200` |
| `GET` | `/v1/videos/jobs/{job_id}` | 查询并刷新单个任务 | `200` |
| `DELETE` | `/v1/videos/jobs/{job_id}` | 取消任务 | `200` |
| `GET` | `/v1/videos/jobs/{job_id}/content` | 播放或下载成品 | `200` / `206` |

## 5. 查询可用模型

不同 API Key 可能被分配到不同分组、账号和定价规则。下游必须以接口实时返回为准，不要把模型列表写死。

### 请求

```bash
curl -sS "${BASE_URL}/v1/models" \
  -H "Authorization: Bearer ${API_KEY}"
```

### 响应示例

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
      "supports_guidances": true,
      "supports_audio": true
    }
  ]
}
```

模型对象中的可选字段可能不返回：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 创建任务时使用的公开模型名 |
| `object` | string | 固定为 `model` |
| `owned_by` | string | 视频模型通常为 `video` |
| `capability_source` | string | 能力规则来源：`native` 表示 XiaoAPI，`openai_sora` 表示 AIStartLab，`mixed` 表示同名模型来自多种上游 |
| `resolutions` | string[] | 当前 Key 已定价且上游支持的分辨率交集 |
| `default_resolution` | string | 省略 `resolution` 时使用的默认值 |
| `default_duration` | integer | 省略 `duration` 时使用的默认秒数 |
| `default_aspect_ratio` | string | 省略 `aspect_ratio` 时使用的默认比例 |
| `durations` | integer[] | 当前模型支持的可选时长；管理员价格配置仍决定可结算的默认时长 |
| `aspect_ratios` | string[] | 当前模型支持的画面比例 |
| `supports_guidances` | boolean | 是否支持 `guidances` 参考媒体结构 |
| `max_references` | object | 图片、视频、音频参考素材的上限；以 `/v1/models` 返回为准 |
| `supports_audio` | boolean | 是否支持生成音轨 |

模型接口不返回最终任务价格。任务价格在任务创建后通过 `amount` 和 `currency` 返回。

## 6. 上传素材

### 6.1 请求

```bash
curl -sS -X POST "${BASE_URL}/v1/videos/uploads" \
  -H "Authorization: Bearer ${API_KEY}" \
  -F "file=@opening.webp"
```

不要手工设置 multipart boundary。

### 6.2 响应

```json
{
  "media_id": "vidmedia_0123456789abcdef0123456789abcdef",
      "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_0123456789abcdef0123456789abcdef/content",
  "type": "UPLOADED",
  "expires_at": "2026-08-08T12:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `media_id` | string | 平台素材 ID |
| `url` | string | 可放入生成请求的绝对 URL；读取时仍需 Bearer 鉴权 |
| `type` | string | 媒体引用类型，通常为 `UPLOADED` |
| `expires_at` | string | RFC 3339 格式的素材过期时间 |

### 6.3 素材参考限制

实际校验由当前模型和上游共同决定。当前接入页面展示的参考限制如下，仍应以联调结果和 `/v1/models` 为准：

| 类型 | 格式 | 大小/时长参考 |
|---|---|---|
| 图片 | PNG、JPEG、WebP | 最大 10 MiB |
| 视频 | MP4、MOV（ISO BMFF） | 最大 100 MiB |
| 音频 | MP3、WAV（16/24-bit PCM） | 2-30 秒，最大 15 MiB |

注意：

- 上传成功不代表某个模型一定支持该素材类型；最终以创建任务结果为准；
- `expires_at` 到期前应完成任务创建；
- 多个由平台上传的素材必须能绑定到同一个上游视频账号，否则创建时返回 `422 VIDEO_MEDIA_INVALID`；
- 公网 HTTP(S) 素材 URL 可以直接传入，但必须能被上游访问，且不应依赖 Cookie、内网地址或短时单次链接；
- 不接受 `data:` URL，也不要把 Base64 或原始媒体字节放进创建任务 JSON。

## 7. 创建视频任务

### 7.1 必需请求头

```http
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
Prefer: respond-async
Idempotency-Key: YOUR_UNIQUE_OPERATION_KEY
```

`Idempotency-Key` 是强制项，规则如下：

- 长度为 1-128 个字符；
- 只能包含可打印 ASCII 字符，即字符范围 `0x20` 至 `0x7E`；
- 推荐使用下游业务订单号或 UUID，例如 `video-order-20260807-0001`；
- 同一次业务请求的超时、断线或 5xx 重试必须复用原 Key 和完全相同的 JSON 语义；
- 同一个 API Key 下，用相同幂等键提交不同请求会返回 `409 IDEMPOTENCY_KEY_CONFLICT`；
- 幂等键按 API Key 隔离，不同 API Key 可以使用相同字符串。

### 7.2 请求字段

创建接口严格校验顶层 JSON，未知顶层字段会返回 `400 VIDEO_REQUEST_INVALID`。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `model` | string | 是 | `/v1/models` 返回的模型 `id` |
| `prompt` | string | 是 | 视频内容、动作、场景、镜头和风格描述；不能为空 |
| `resolution` | string | 否 | 例如 `480p`、`720p`；省略时使用平台定价配置的默认值 |
| `duration` | integer | 否 | 正整数秒；省略时使用平台默认值 |
| `aspect_ratio` | string | 否 | 例如 `16:9`、`9:16`、`1:1` |
| `audio` | boolean | 否 | 是否让模型生成音轨，默认 `false`；不能传字符串或 `null` |
| `prompt_enhance` | string | 否 | 常见值为 `AUTO`、`ON`、`OFF`，是否支持取决于模型 |
| `start_frame_url` | string | 否 | 首帧绝对 HTTP(S) URL，推荐使用上传接口返回的 `url` |
| `end_frame_url` | string | 否 | 尾帧绝对 HTTP(S) URL |
| `image_url` | string | 否 | `start_frame_url` 的旧兼容别名；新接入不要使用 |
| `guidances` | object | 否 | 图片、视频、音频参考媒体，结构见第 8 节 |

`image_url` 与 `start_frame_url` 不能同时传。可选字段不使用时应直接省略，不要传空字符串或 `null`。

### 7.3 文生视频示例

```bash
curl -i -X POST "${BASE_URL}/v1/videos/generations" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Prefer: respond-async" \
  -H "Idempotency-Key: video-order-20260807-0001" \
  -d '{
    "model": "seedance-2.0",
    "prompt": "云海之上的仙侠山门，清晨薄雾，镜头缓慢向前推进，电影级光影",
    "resolution": "480p",
    "duration": 5,
    "aspect_ratio": "16:9",
    "audio": true
  }'
```

### 7.4 图生视频示例

先按第 6 节上传图片，再使用响应中的 `url`：

```bash
curl -i -X POST "${BASE_URL}/v1/videos/generations" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Content-Type: application/json" \
  -H "Prefer: respond-async" \
  -H "Idempotency-Key: video-order-20260807-0002" \
  -d '{
    "model": "seedance-2.0",
    "prompt": "保持人物与服装一致，衣袂随风，镜头平稳环绕",
    "resolution": "720p",
    "duration": 8,
    "aspect_ratio": "16:9",
      "start_frame_url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_xxx/content"
  }'
```

### 7.5 创建成功响应

```http
HTTP/1.1 202 Accepted
Preference-Applied: respond-async
Location: /v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef
Content-Type: application/json
```

```json
{
  "job_id": "vidjob_0123456789abcdef0123456789abcdef",
  "status": "pending",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef"
}
```

`status_url` 和 `Location` 是相对路径，下游应以本次请求使用的 `Base URL` 拼接，不要写死其他域名。

## 8. 参考媒体与生成音轨

`audio` 和 `guidances.audio_reference` 含义不同：

- `audio: true`：要求模型为生成的视频创建音轨；
- `guidances.audio_reference`：提供一段已有音频作为节奏、动作或风格参考。

### 8.1 图片与音频参考示例

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
        "strength": "MID",
        "order": 1
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

### 8.2 视频参考结构

```json
{
  "guidances": {
    "video_reference_base": [
      {
        "video": {
          "url": "https://api.gavinteam.online/v1/videos/uploads/vidmedia_video/content",
          "type": "UPLOADED"
        }
      }
    ]
  }
}
```

### 8.3 XiaoAPI 模型能力参考

下表仅适用于 `capability_source=native` 的 XiaoAPI 模型，不适用于 AIStartLab。它是代码内接入页面的能力参考，不是静态授权清单。模型别名、可用分辨率和默认参数可能由平台管理员调整，生产调用必须先查询 `/v1/models`。

| 模型 | 分辨率参考 | 时长参考 | 比例参考 | 素材/音轨参考 |
|---|---|---|---|---|
| `seedance-2.0` | 480p / 720p / 1080p | 4-15 秒；1080p 最长 12 秒 | 16:9、9:16、1:1、4:3、3:4、21:9、9:21；720p 不支持 9:21 | 生成音轨、首尾帧、图片/视频/音频参考 |
| `seedance-2.0-fast` | 480p / 720p | 4-15 秒 | Seedance 常用 7 种比例 | 生成音轨、首尾帧、图片/视频/音频参考 |
| `seedance-2.0-mini` | 480p / 720p | 4-15 秒 | 16:9、1:1、9:16 | 生成音轨、首尾帧、图片/视频/音频参考 |
| `happy-horse-1.1` | 720p / 1080p | 3-15 秒 | 16:9、4:3、1:1、3:4、9:16 | 音轨、首帧、图片参考、提示增强 |
| `grok-imagine-1.5` | 400p / 544p / 720p / 960p | 3-15 秒 | 16:9、9:16、1:1；544p/960p 仅 1:1 | 必须且只能提供一张首帧；不支持尾帧和 `guidances` |
| `ltx-2.3-pro` | 1080p / 1440p / 2160p | 6 / 8 / 10 秒 | 仅 16:9 | 音轨、首尾帧、提示增强；不支持 `guidances` |
| `ltx-2.3-fast` | 1080p / 1440p / 2160p | 6-20 秒偶数 | 仅 16:9 | 音轨、首尾帧、提示增强；不支持 `guidances` |

其他参考规则：

- Seedance 最多参考图片 4 张、参考视频 3 个、参考音频 1 个；
- Happy Horse 最多参考图片 9 张，支持首帧，不支持尾帧；
- 首帧或尾帧不能与 `guidances.image_reference` 同时使用；
- `strength`、`order` 及更深层的 `guidances` 字段由具体模型校验；不支持时通常返回 `422 VIDEO_OPTION_UNSUPPORTED` 或 `VIDEO_MEDIA_INVALID`。

AIStartLab（`capability_source=openai_sora`）使用其配置接口返回的分辨率、时长、比例、模式和参考素材上限，并叠加管理员价格配置；它不继承本节 XiaoAPI 的静态文件大小、时长和组合限制。AIStartLab 的素材必须是上游可访问的公网 HTTP(S) URL，工作台会提供 URL 输入，不会走 `/v1/videos/uploads`。

## 9. 查询任务

### 9.1 查询单个任务

```bash
curl -sS "${BASE_URL}/v1/videos/jobs/${JOB_ID}" \
  -H "Authorization: Bearer ${API_KEY}"
```

该接口会在任务未结束时向上游刷新一次状态。

### 9.2 任务响应示例

处理中：

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
  "created_at": "2026-08-07T12:00:00Z",
  "updated_at": "2026-08-07T12:00:10Z",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef"
}
```

已完成：

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
  "created_at": "2026-08-07T12:00:00Z",
  "updated_at": "2026-08-07T12:02:30Z",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef",
  "content_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef/content"
}
```

失败：

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
  "created_at": "2026-08-07T12:00:00Z",
  "updated_at": "2026-08-07T12:01:00Z",
  "finished_at": "2026-08-07T12:01:00Z",
  "settlement_status": "released",
  "status_url": "/v1/videos/jobs/vidjob_0123456789abcdef0123456789abcdef",
  "error": {
    "code": "VIDEO_GENERATION_FAILED",
    "message": "resolution is not supported by this model",
    "stage": "processing",
    "task_id": "vidjob_0123456789abcdef0123456789abcdef",
    "failed_at": "2026-08-07T12:01:00Z",
    "upstream_code": "VIDEO_RESOLUTION_INVALID",
    "request_id": "req_0123456789abcdef"
  }
}
```

错误诊断只公开白名单化的错误码、失败阶段和请求追踪号；任务响应不会暴露上游账号、上游任务 ID、Key、私有地址或原始上游错误正文。

### 9.3 字段说明

| 字段 | 类型 | 说明 |
|---|---|---|
| `job_id` | string | 平台任务 ID |
| `status` | string | 当前任务状态 |
| `model` | string | 下游请求使用的公开模型名 |
| `resolution` | string | 平台最终解析的分辨率 |
| `duration` | integer | 平台最终解析的时长，单位秒 |
| `aspect_ratio` | string | 最终比例 |
| `amount` | string | 对下游计费金额，固定 8 位小数；请按十进制定点数处理，不要使用二进制浮点累计 |
| `currency` | string | 币种，常见为 `USD` |
| `created_at` | string | RFC 3339 创建时间 |
| `updated_at` | string | RFC 3339 最近更新时间 |
| `finished_at` | string | 终态时间；任务尚未结束时不返回 |
| `settlement_status` | string | 费用状态：`held`、`captured` 或 `released` |
| `status_url` | string | 相对查询路径 |
| `content_url` | string | 仅 `completed` 时返回的相对下载路径 |
| `error` | object | 仅 `failed` 时返回的脱敏错误 |

失败任务的 `error` 可包含：

| 字段 | 类型 | 说明 |
|---|---|---|
| `code` | string | 平台稳定错误码，当前为 `VIDEO_GENERATION_FAILED` |
| `message` | string | 可公开的脱敏错误说明 |
| `stage` | string | 失败阶段，例如 `validation`、`processing`、`content`、`settlement` |
| `task_id` | string | 与顶层 `job_id` 相同，便于直接复制诊断信息 |
| `failed_at` | string | RFC 3339 失败时间 |
| `upstream_code` | string | 上游返回且通过公开白名单的错误码；无法安全识别时不返回 |
| `request_id` | string | 可公开的上游请求追踪号；上游未返回时不提供 |

### 9.4 状态机

| 状态 | 是否终态 | 说明 | 下游动作 |
|---|---|---|---|
| `pending` | 否 | 已接受，等待执行 | 继续轮询；允许取消 |
| `running` | 否 | 正在生成 | 继续轮询；允许取消 |
| `settling` | 否 | 上游任务进入结算阶段 | 继续轮询；不可取消 |
| `completed` | 是 | 生成成功 | 立即下载并转存成品 |
| `failed` | 是 | 生成失败 | 读取通用错误；如需重做，使用新的幂等键创建新任务 |
| `canceled` | 是 | 已取消 | 不再轮询；冻结金额会释放 |

### 9.5 查询最近任务

```bash
curl -sS "${BASE_URL}/v1/videos/jobs?limit=20" \
  -H "Authorization: Bearer ${API_KEY}"
```

响应按创建时间倒序：

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
      "created_at": "2026-08-07T12:00:00Z",
      "updated_at": "2026-08-07T12:02:30Z",
      "status_url": "/v1/videos/jobs/vidjob_xxx",
      "content_url": "/v1/videos/jobs/vidjob_xxx/content"
    }
  ]
}
```

`limit` 默认值为 `20`，最大值为 `100`。当前接口不提供 cursor 或 offset 分页。

## 10. 取消任务

仅 `pending` 和 `running` 状态允许请求取消：

```bash
curl -sS -X DELETE "${BASE_URL}/v1/videos/jobs/${JOB_ID}" \
  -H "Authorization: Bearer ${API_KEY}"
```

成功时返回最新任务对象。取消是一个请求动作；如果上游没有立即返回明确状态，响应仍可能暂时显示原状态，随后应继续低频查询，直到进入终态。

对 `settling` 或任何终态任务取消会返回：

```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "VIDEO_JOB_NOT_CANCELABLE",
    "message": "video job is not cancelable"
  }
}
```

## 11. 下载与播放

### 11.1 下载成品

```bash
curl -fL "${BASE_URL}/v1/videos/jobs/${JOB_ID}/content?download=1" \
  -H "Authorization: Bearer ${API_KEY}" \
  -o result.mp4
```

只有 `completed` 状态可以下载。其他状态返回 `404 VIDEO_RESOURCE_NOT_FOUND`。

接口支持 `Range`，可以用于断点续传和播放器拖动：

```bash
curl -i "${BASE_URL}/v1/videos/jobs/${JOB_ID}/content" \
  -H "Authorization: Bearer ${API_KEY}" \
  -H "Range: bytes=0-1048575"
```

上游支持时返回 `206 Partial Content`。

### 11.2 浏览器播放

原生 `<video src="...">` 无法添加 `Authorization` 请求头。浏览器端应由可信后端代理，或使用 `fetch` 获取 Blob 后创建临时对象 URL：

```ts
const response = await fetch(`${baseUrl}${contentUrl}`, {
  headers: { Authorization: `Bearer ${apiKey}` },
})

if (!response.ok) throw new Error(`download failed: ${response.status}`)

const blob = await response.blob()
const objectUrl = URL.createObjectURL(blob)
videoElement.src = objectUrl

// 页面销毁或切换视频时释放
URL.revokeObjectURL(objectUrl)
```

不要把 API Key 拼接到 query string。面向终端用户的生产系统建议由自己的后端下载并转存到对象存储，再返回受控的临时访问链接。

## 12. 计费与幂等语义

统一异步任务 API 的计费流程：

1. 创建任务前，平台根据最终解析的模型、分辨率、时长、音轨开关和分组定价计算金额；
2. 余额不足时返回 `402 INSUFFICIENT_BALANCE`，不会向上游创建任务；
3. 创建时预冻结任务金额；
4. `completed` 时捕获金额；
5. `failed` 或 `canceled` 时释放冻结金额；
6. 任务响应的 `amount`、`currency` 是对下游的计费结果，不是上游供应商成本。

关键行为：

- 相同 API Key、相同幂等键、相同请求会返回原任务，不重复创建和扣费；
- 相同幂等键但请求内容不同会返回 `409`；
- 首次请求在“已发送上游但响应丢失”的不确定状态下，平台保留幂等记录；下游必须用相同请求体和幂等键重试；
- 并发提交同一幂等请求时，可能短暂返回 `503 VIDEO_REQUEST_IN_PROGRESS`，应稍后用相同请求重试；
- 不要因为客户端超时就改用新幂等键，否则可能生成两个视频并产生两笔费用。

## 13. 错误处理与重试

### 13.1 错误结构

```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "VIDEO_RESOLUTION_INVALID",
    "message": "resolution is not supported by this model"
  }
}
```

下游应先判断 HTTP 状态，再读取 `error.code`。`message` 用于展示或排查，不应作为程序分支条件。

### 13.2 常见错误码

| HTTP | 错误码 | 含义 | 是否自动重试 |
|---|---|---|---|
| `400` | `ASYNC_REQUIRED` | 缺少 `Prefer: respond-async` | 否，补请求头 |
| `400` | `VIDEO_REQUEST_INVALID` | JSON、字段类型、未知字段、媒体 URL 或 Content-Type 不合法 | 否，修正请求 |
| `400` | `IDEMPOTENCY_KEY_INVALID` | 幂等键缺失、过长或包含非可打印 ASCII | 否，修正请求 |
| `401` | `API_KEY_REQUIRED` 或鉴权错误 | Key 缺失、无效或已失效 | 否，检查 Key |
| `402` | `INSUFFICIENT_BALANCE` | 余额不足 | 否，充值或降低规格后用新幂等键创建 |
| `403` | `VIDEO_GENERATION_DISABLED` | 当前 Key 未开通统一视频分组 | 否，联系平台方 |
| `404` | `VIDEO_RESOURCE_NOT_FOUND` | 资源不存在、Key 不匹配或成品尚不可下载 | 否，先核对任务状态和 Key |
| `409` | `IDEMPOTENCY_KEY_CONFLICT` | 同一幂等键对应了不同请求 | 否，更正业务逻辑 |
| `409` | `VIDEO_JOB_NOT_CANCELABLE` | 当前状态不允许取消 | 否，刷新任务状态 |
| `422` | `VIDEO_MODEL_INVALID` | 模型不支持 | 否，刷新模型列表 |
| `422` | `VIDEO_PROMPT_INVALID` | 提示词不符合模型要求 | 否，修正提示词 |
| `422` | `VIDEO_RESOLUTION_INVALID` | 分辨率不支持 | 否，使用模型返回的分辨率 |
| `422` | `VIDEO_DURATION_INVALID` | 时长不支持 | 否，修正时长 |
| `422` | `VIDEO_ASPECT_RATIO_INVALID` | 比例不支持 | 否，修正比例 |
| `422` | `VIDEO_MEDIA_INVALID` | 素材无效、过期、Key 不匹配或上游账号不一致 | 否，检查或重新上传素材 |
| `422` | `VIDEO_UPLOAD_UNSUPPORTED` | 所选上游只接受公网素材 URL | 否，改用公网 HTTP(S) URL |
| `422` | `VIDEO_OPTION_UNSUPPORTED` | 所选上游不支持请求中的选项或素材组合 | 否，按模型能力调整请求 |
| `422` | `VIDEO_OPTION_UNSUPPORTED` | 当前模型不支持某个选项 | 否，移除该选项 |
| `429` | `VIDEO_CAPACITY_EXHAUSTED` | 暂无可用视频容量 | 是，指数退避；优先遵守 `Retry-After` |
| `503` | `VIDEO_REQUEST_IN_PROGRESS` | 相同幂等请求仍在创建 | 是，保留原请求和幂等键 |
| `503` | `VIDEO_EXECUTION_DISABLED` | 平台视频执行总开关关闭 | 是，联系平台方确认后重试 |
| `503` | `VIDEO_PRICING_UNAVAILABLE` | 当前模型/分辨率没有可用定价 | 否，联系平台方或换规格 |
| `503` | `VIDEO_UPSTREAM_UNAVAILABLE` | 上游网络、凭证或服务不可用 | 是，保留原请求和幂等键 |
| `4xx/5xx` | `VIDEO_UPSTREAM_ERROR` | 已脱敏的其他上游错误 | 仅 5xx 或平台明确建议时重试 |

上游返回的 `401`、`402`、`403` 和 `5xx` 会被平台统一转换为 `503`，避免泄漏上游凭证和账号状态。

### 13.3 推荐重试策略

- 建连失败、读取超时、连接重置、HTTP `429` 或 `5xx`：使用相同 JSON 和相同 `Idempotency-Key` 重试；
- 首次等待 2 秒，随后使用 4、8、16、30 秒的指数退避，并加入随机抖动；
- `Retry-After` 存在时优先遵守；不存在时使用本地退避；
- `400`、`401`、`402`、`403`、`422` 和幂等冲突不要原样自动重试；
- 创建请求得到 `202` 后，不再重复创建，后续只轮询 `status_url`；
- 保存平台返回的请求追踪头（如 `X-Request-Id`）、HTTP 状态、`error.code` 和发生时间；不要记录完整 Key。

## 14. Grok 原生兼容协议

仅当平台方明确提供的是 Grok 分组 API Key 时使用本节。该协议透传 Grok/xAI 风格的视频响应，不使用统一的 `vidjob_` 任务结构。

### 14.1 路径

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/v1/videos/generations` | 创建 Grok 视频 |
| `POST` | `/v1/videos/edits` | 编辑视频 |
| `POST` | `/v1/videos/extensions` | 延长视频 |
| `GET` | `/v1/videos/{request_id}` | 查询状态 |
| `GET` | `/v1/videos/{request_id}/content` | 下载内容，支持 Range |

### 14.2 与统一异步任务 API 的差异

- 创建响应由所配置的 Grok 上游返回，下游应从 `request_id`、`id`、`data.request_id`、`data.id`、`video.request_id` 或 `video.id` 中读取任务标识；具体结构以联调响应为准；
- 查询路径是 `/v1/videos/{request_id}`，不是 `/v1/videos/jobs/{job_id}`；
- 不提供 `/v1/videos/uploads`、任务列表和统一取消路径；
- Gavin2API 会把状态响应中的 Grok 成品地址改写为本站 `/v1/videos/{request_id}/content` 代理地址；
- 查询和下载必须继续使用创建任务时的同一用户与同一 API Key；
- 请求体按当前 Grok 上游协议透传，创建类请求至少必须包含非空 `model`；
- `grok-imagine-video-1.5` 在纯文生视频时可能被网关规范化到其上游文本生成别名，带输入图片时保留图生视频模型；
- 统一异步任务 API 的 `Prefer`、`Idempotency-Key`、`job_id`、任务列表和冻结结算语义不能套用到 Grok 原生兼容协议。

新下游如果不依赖 Grok 原生请求/响应格式，建议使用统一异步任务 API。

## 15. Python 完整示例

依赖：`pip install requests`

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


def api_error(response: requests.Response) -> RuntimeError:
    try:
        payload = response.json()
        error = payload.get("error", {})
        code = error.get("code", "UNKNOWN_ERROR")
        message = error.get("message", response.text)
    except ValueError:
        code, message = "NON_JSON_ERROR", response.text[:500]
    request_id = response.headers.get("X-Request-Id", "-")
    return RuntimeError(
        f"HTTP {response.status_code} {code}: {message}; request_id={request_id}"
    )


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
                f"{BASE_URL}/v1/videos/generations",
                headers=headers,
                json=body,
                timeout=(10, 45),
            )
        except (requests.Timeout, requests.ConnectionError):
            if attempt == 5:
                raise
        else:
            if response.status_code == 202:
                return response.json()
            if response.status_code != 429 and response.status_code < 500:
                raise api_error(response)
            if attempt == 5:
                raise api_error(response)
            retry_after = response.headers.get("Retry-After")
            if retry_after and retry_after.isdigit():
                time.sleep(int(retry_after))
                continue

        delay = min(2 ** (attempt + 1), 30) + random.random()
        time.sleep(delay)

    raise RuntimeError("unreachable")


def wait_for_video(job_id: str) -> dict:
    delay = 5
    while True:
        response = requests.get(
            f"{BASE_URL}/v1/videos/jobs/{job_id}",
            headers=AUTH,
            timeout=(10, 45),
        )
        if response.status_code != 200:
            raise api_error(response)

        job = response.json()
        status = job["status"]
        print(f"job={job_id} status={status}")

        if status == "completed":
            print("charged:", Decimal(job["amount"]), job["currency"])
            return job
        if status in {"failed", "canceled"}:
            raise RuntimeError(f"video ended with status={status}: {job.get('error')}")

        time.sleep(delay)
        delay = min(delay + 5, 30)


def download_video(content_url: str, output: Path) -> None:
    with requests.get(
        f"{BASE_URL}{content_url}",
        headers=AUTH,
        stream=True,
        timeout=(10, 300),
    ) as response:
        if response.status_code != 200:
            raise api_error(response)
        with output.open("wb") as file:
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if chunk:
                    file.write(chunk)


created = create_video()
job = wait_for_video(created["job_id"])
download_video(job["content_url"], Path("result.mp4"))
```

生产代码还应增加整体任务超时、进程重启后的任务恢复，以及 `206` 分段下载处理。

## 16. TypeScript 服务端示例

以下示例适用于 Node.js 18+：

```ts
import { randomUUID } from 'node:crypto'
import { writeFile } from 'node:fs/promises'

const baseUrl = process.env.GAVIN2API_BASE_URL!.replace(/\/$/, '')
const apiKey = process.env.GAVIN2API_KEY!
const auth = { Authorization: `Bearer ${apiKey}` }

type VideoJob = {
  job_id: string
  status: 'pending' | 'running' | 'settling' | 'completed' | 'failed' | 'canceled'
  amount: string
  currency: string
  status_url: string
  content_url?: string
  error?: { code: string; message: string }
}

async function readError(response: Response): Promise<Error> {
  const text = await response.text()
  let detail = text
  try {
    const body = JSON.parse(text)
    detail = `${body.error?.code ?? 'UNKNOWN_ERROR'}: ${body.error?.message ?? text}`
  } catch {
    // Keep the bounded raw response for diagnostics.
    detail = text.slice(0, 500)
  }
  return new Error(`HTTP ${response.status} ${detail}`)
}

async function createVideo(): Promise<{ job_id: string; status_url: string }> {
  const response = await fetch(`${baseUrl}/v1/videos/generations`, {
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
  if (response.status !== 202) throw await readError(response)
  return response.json()
}

async function waitForVideo(jobId: string): Promise<VideoJob> {
  let delayMs = 5_000
  for (;;) {
    const response = await fetch(`${baseUrl}/v1/videos/jobs/${jobId}`, { headers: auth })
    if (!response.ok) throw await readError(response)
    const job = (await response.json()) as VideoJob
    if (job.status === 'completed') return job
    if (job.status === 'failed' || job.status === 'canceled') {
      throw new Error(`video ${job.status}: ${JSON.stringify(job.error)}`)
    }
    await new Promise(resolve => setTimeout(resolve, delayMs))
    delayMs = Math.min(delayMs + 5_000, 30_000)
  }
}

async function downloadVideo(contentUrl: string): Promise<void> {
  const response = await fetch(`${baseUrl}${contentUrl}`, { headers: auth })
  if (!response.ok) throw await readError(response)
  await writeFile('result.mp4', Buffer.from(await response.arrayBuffer()))
}

const created = await createVideo()
const job = await waitForVideo(created.job_id)
await downloadVideo(job.content_url!)
```

大文件生产下载建议使用流式管道写入文件或对象存储，避免把整个视频读入内存。

## 17. 下游上线检查表

- [ ] Base URL 未重复拼接 `/v1`，生产环境使用 HTTPS；
- [ ] 已确认 API Key 属于统一异步视频分组，而不是 Grok 原生分组；
- [ ] 启动或定时刷新 `/v1/models`，没有写死平台模型授权；
- [ ] API Key 仅保存在服务端或密钥管理系统；
- [ ] 每个业务创建操作生成稳定且唯一的 `Idempotency-Key`；
- [ ] 网络超时和 5xx 重试会复用完全相同的请求体与幂等键；
- [ ] `202` 后只轮询任务，不重复创建；
- [ ] 轮询从约 5 秒开始并逐步退避，设置整体任务超时；
- [ ] 金额使用 Decimal/定点数处理；
- [ ] 使用创建任务时的同一 API Key 查询、取消和下载；
- [ ] 已处理 `pending`、`running`、`settling`、`completed`、`failed`、`canceled` 全部状态；
- [ ] 成品完成后及时转存，不依赖平台长期保存；
- [ ] 日志记录 `X-Request-Id`、HTTP 状态和错误码，但不会记录完整 API Key；
- [ ] 已验证内容审核拒绝、余额不足、容量不足、任务失败和下载中断场景。

## 18. 联调时建议提供的信息

出现问题时，下游应向平台方提供：

- 请求时间和时区；
- 请求路径和 HTTP 方法；
- HTTP 状态码；
- `error.code` 与脱敏后的 `error.message`；
- 响应头中的 `X-Request-Id`（如有）；
- `job_id` 或 Grok `request_id`；
- API Key 的末 4 位，不要提供完整 Key；
- 使用的模型、分辨率、时长和比例；
- 是否使用上传素材以及素材的 `media_id`。

不要发送完整 API Key、上游凭证、原始私密素材或包含敏感信息的完整请求日志。
