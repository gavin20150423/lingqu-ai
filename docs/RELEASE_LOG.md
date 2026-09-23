# Gavin2API 发布日志

每次正式发布都必须新增版本条目，并分别写清楚“修复了什么”、“增加了什么”和“当前已有功能”。没有新增功能时也必须明确记录。

## v0.2.7-lingqu.2 - 2026-09-24

### 修复了什么

- **池模式（`pool_mode`）显式配置的 401/403 同账号重试被 handler 层拦截压掉的回归**。`sameAccountRetryAllowed` 对 401/403 无条件拒绝，而池模式默认重试状态码列表 `defaultPoolModeRetryableStatusCodes` 本就含 401/403，两者语义冲突：池模式账号持的是共享上游凭证，其中的 401/403 属于池内凭证轮换/临时拒绝，**不是针对本次请求的确定性拒绝**。现新增 `UpstreamFailoverError.PoolModeSameAccountRetry` 标记及该状态下的拦截例外（账号侧判定入口 `Account.PoolModeSameAccountRetryFor`）。标记在 `newOpenAIAccountFailoverErrorWithClassificationHeaders` 统一补写，覆盖 passthrough / forward / images / embeddings / alpha_search / cc_pipeline 等全部漏斗调用点，并在不走该漏斗的字面量构造点（`gateway_forward`、`gemini_*`、`anthropic_passthrough`、`bedrock`、`grok` 家族、`forward_as_*`）显式补写，避免不同端点行为不一致。**非池模式账号的 401/403 确定性拒绝拦截行为保持不变。**
- **OAuth 非流式出图把「写下游」耗时算进了上游耗时**。`handleOpenAIImagesOAuthNonStreamingResponse` 与 `handleCodexDirectImagesNonStreamingResponse` 原先在 `c.Data` 之后才取 `time.Since(startTime)`，慢客户端会放大上游耗时与计费口径；现改为读完上游 body、写下游之前取样并作为返回值传出，与 api_key 路径 `handleOpenAIImagesNonStreamingResponse` 同口径。流式路径沿用墙上时间，不变。

### 增加了什么

- 无新增业务功能；本版只修实现侧回归与耗时口径。
- 测试同步：`TestOpenAIImagesNonStreamingDurationExcludesDownstreamWrite/oauth` 的桩由旧 Responses SSE 更新为 Codex 直连 Images 端点的 `application/json`（`gpt-image-2` 已由 `c0d511937` 改走该端点）；三个测试文件按新签名补 duration 返回值入参。
- **本版无新增数据库迁移**，不触碰任何表结构或计费数据。

### 当前已有功能

- 与 `0.2.7-lingqu.1` 一致：内容审核 `engine_meta` 审计溯源、渠道 reasoning effort 分档计费倍率、订阅目录与配额、插件、channel monitor v2、opencode-go / minimax 平台接入、Grok 视频 pending 计费快照、SubPilot 租约释放与内池耗尽 503 `upstream_pool_exhausted` 分支、`gavin2api` 稳定网络别名部署、账号共享/商城/积分/社区、Composite 多供应商分组、异步图片任务与视频工作台等全部保留。

### 验证重点

- 池模式账号遇到 401/403 时应**在同账号内重试** `pool_mode_retry_count` 次，而不是立刻换号；非池模式账号的 401/403 行为必须与发布前一致（未被放宽）。
- OAuth 非流式出图的 usage 耗时口径应只反映上游耗时，慢客户端不再放大该值。
- ⚠️ 回滚提示：本版无新增迁移，回滚到 `0.2.7-lingqu.1`（容器 `gavin2api-release-0.2.7-lingqu.1-150be00d2`，已退役保留的上一版生产容器，只 stop 不删）无 DB 侧遗留；按 `docker network connect --alias gavin2api …` → `docker start` 顺序恢复别名后即可。回滚后将**重新丢失池模式 401/403 同账号重试**。
- 切流手法：稳定别名金丝雀；**退役旧容器用优雅 `docker stop`（SIGTERM → FIN），不再用 `docker network disconnect`**（黑洞式断链会挂死 Caddy 按主机名建的上游连接池）。详见 `docs/release-reports/20260924-v0.2.7-lingqu.2.md` §6。

## v0.2.7-lingqu.1 - 2026-09-23

### 修复了什么

- 修复合并上游 Sub2API v0.2.7（`20a94fbb5`）过程中被 git 自动合并**静默吃掉**的三处本地改动：
  - `grok_media.go` 视频 create 路径丢失 `StoreGrokVideoPendingBilling` 落快照块，导致 status 侧 `prepareGrokVideoCompletionBilling` 因缺快照 fail-closed —— **视频用量会彻底漏计费**。已补回落快照双写重试块。
  - `openai_gateway_handler.go` 的 SubPilot 租约释放字段：3 处 `RecordUsage` 缺 `SubPilotLeaseID` / `SubPilotSessionKey`（由上一次 v0.2.5 合并丢失）。缺失会导致成功请求不归还 sidecar 租约，账号在 SubPilot 侧被持续误判为并发已满。
  - `openai_gateway_handler.go` 的内池耗尽 503 / `upstream_pool_exhausted` 分支（同样由上一次 v0.2.5 合并丢失）。缺失会把内池最后一个账号的 429 原样抛给下游，使 newapi 等下游把本服务整条链路当作**单个被限流账号整体冷却**。
- 修复 `content_moderation.go` 合并产生的字段重复声明（导致后端编译失败）。
- 修复合并引入的若干断言过时：内容审核日志列数、账号快照 denylist 分类、gemini 白名单断言、前端平台数与分页签名等。

### 增加了什么

- 合并上游 0.2.5 → 0.2.7 的全部上游功能（内容审核 `engine_meta` 审计溯源、渠道 reasoning effort 分档计费倍率、订阅目录与配额、插件、channel monitor v2、opencode-go / minimax 平台接入等，明细见上游版本记录）。
- 新增数据库迁移（均为增量、向后兼容）：
  - `238b_content_moderation_engine_meta.sql`：`content_moderation_logs` 增加**可空** `engine_meta JSONB`（老行与老版本写入保持 NULL）。
  - `239_channel_reasoning_effort_multipliers.sql`：`channel_model_pricing` 与 `channel_account_stats_model_pricing` 增加 `reasoning_effort_multipliers JSONB`，并把旧的 `max_reasoning_effort_multiplier` 回填/迁移；`groups.model_pricing` 中的旧键一并迁移并移除。
- 本仓专有业务功能**无新增**。

### 当前已有功能

- 与 `0.2.5-lingqu.4` 一致：自定义提示词输出回传、`client_type` 调度上报、`gavin2api` 稳定网络别名部署、账号共享/商城/积分/社区、Composite 多供应商分组、异步图片任务与视频工作台等全部保留。

### 验证重点

- 视频生成（Grok 媒体）完成后 usage 应正常入账，不再漏计费。
- SubPilot 侧观察：OpenAI 系（`/v1/responses`、messages、WebSocket）请求结束后租约应立即释放，账号并发同步下降。
- 内池整体被限流时，客户端应收到 503 `upstream_pool_exhausted`，而不是最后一个账号的 429。
- 迁移校验：`content_moderation_logs.engine_meta`、`channel_model_pricing.reasoning_effort_multipliers`、`channel_account_stats_model_pricing.reasoning_effort_multipliers` 三列存在；`groups.model_pricing` 中原先带 `max_reasoning_effort_multiplier` 的条目已迁移为 `reasoning_effort_multipliers`。
- ⚠️ 回滚提示：`239` 会从 `groups.model_pricing` 移除旧键 `max_reasoning_effort_multiplier`。若需回滚到 `0.2.5-lingqu.4`，这些分组的分档倍率将不再被旧代码读取，需手工按 `reasoning_effort_multipliers` 回填为旧键。


## v0.2.5-lingqu.4 - 2026-09-21

### 修复了什么

- **内部测试端点（IQ 测试）在 OpenAI 系账号上提示词丢失**：`Responses` 主路径与 CN 自适应路径（Responses/Anthropic）调用测试 payload 构造函数时未透传调用方提示词，始终发送默认 "hi" —— 表现为糖果题/鹈鹕题发出后模型只回一句招呼语。现全部路径透传 prompt。

### 增加了什么

- 测试端点支持思考强度：请求新增 `reasoning_effort`（low/medium/high），在 OpenAI Responses 路径写入 `reasoning.effort`；请求新增 `max_output_tokens` 支持（跟随既有 `max_tokens` 语义）。
- 供 SubPilot v4.36 智商测试弹窗的"模型 / 思考强度"选择使用。

### 当前已有功能

- 与 `0.2.5-lingqu.3` 一致（自定义提示词输出回传、client_type 调度上报等全部保留）。

### 验证重点

- 对 OpenAI apikey 账号（如 1034 aigateway-pro）发糖果题：应返回题目相关回答而非 "Hi"。
- `reasoning_effort=high` 时 gpt-5 系响应应体现更长思考（延迟/内容变化）。
- 常规探测（空 prompt）行为不变（默认 "hi"、1024 上限）。

## v0.2.5-lingqu.3 - 2026-09-21

### 修复了什么

- 内部测试端点（`/api/v1/internal/subpilot/probe/:id`）的 prompt 参数此前对 Anthropic 账号被忽略（固定发送 "hi"），且响应不携带模型输出文本，无法用于"智商测试"类内容检测。

### 增加了什么

- 内部测试端点支持自定义提示词的完整输出回传：请求新增 `max_tokens`（0=默认 1024），响应新增 `response_text`；Anthropic 路径透传调用方提示词并支持放宽输出上限（SubPilot 智商测试使用 8192，可供糖果题/鹈鹕骑车动画等长输出场景）。
- 供 SubPilot v4.35"智商测试"功能使用：渠道卡片一键运行糖果测试（21/29/其他自动判定）与鹈鹕骑车（生成 2D 动画 HTML 并渲染）。

### 当前已有功能

- 与 `0.2.5-lingqu.2` 一致（含 client_type 调度上报、稳定别名部署等全部保留）。

### 验证重点

- 内部 probe 端点带 prompt + max_tokens 的返回文本完整（糖果题回答可读）。
- 常规探测（无自定义 prompt）行为与之前完全一致（默认 "hi"、1024 上限）。

## v0.2.5-lingqu.2 - 2026-09-18

> 说明：上一版 `0.2.5-lingqu-4904ba07a`（2026-09-17 前后上线，未写入本日志）等效于
> `0.2.5-lingqu.1`；本版按 Runbook 命名规范顺延为 `0.2.5-lingqu.2`。

### 修复了什么

- 无（本版无修复项）。

### 增加了什么

- 调度请求（SubPilot select）新增 `client_type` 字段：网关按既有 User-Agent 机制识别
  Claude Code 客户端后在调度请求中声明 `claude_code`，供 SubPilot 按"账号客户端类型
  标签"分流（仅 CC / 仅普通 / 不限）。调用方为旧版本 SubPilot 时字段被忽略，行为不变。
- 发布工具 `deploy/safe-blue-green-cutover.sh` 新增 `--upstream` 参数，支持按稳定网络
  别名（`gavin2api:8080`）切流并校验别名解析到新容器；此后 Caddyfile 与 SubPilot 的
  upstream 固定为别名，发布不再需要改任何引用（Runbook"生产实例稳定别名"落地）。

### 当前已有功能

- 与 `v0.2.4-lingqu.7` 一致（订阅系列、OAuth/API Key 多账号、SubPilot 调度、精确计费、
  充值订阅、视频工作台、无限画布、管理后台等全部保留）。

### 验证重点

- 候选容器 `/health` 200、日志无 panic/数据库/Redis 连接错误。
- 切流后公网 `api/cdn /health` 连续 200，`/v1/models` 未授权 401。
- SubPilot 委托探测（`SUB2API_BASE_URL=http://gavin2api:8080`）对带
  `probe_as_claude_code=true` 的账号成功。
- SubPilot 调度决策中 CC 请求不再选中"仅普通"标签账号、普通请求不再选中"仅 CC"标签账号。

## v0.2.4-lingqu.7 - 2026-09-14

### 修复了什么

- 修复管理员订阅分配只能选择订阅分组、无法选择具体套餐的问题。
- 修复管理员分配时套餐有效期、日 / 周 / 月额度和权益没有按所选档位保存的问题。
- 修复已有同分组订阅在管理员改选具体档位后仍保留旧套餐快照的问题；改选会在原订阅记录上安全迁移，不改变用户和历史用量归属。
- 修复套餐列表加载失败时仍允许提交模糊分组分配的问题，避免误配。

### 增加了什么

- 管理员分配订阅增加具体套餐选择器、档位价格 / 有效期 / 日周月额度预览，以及套餐名称回显。
- 支持批量分配订阅时传递并校验具体套餐；套餐始终以所属分组为边界，防止跨分组误配。

### 当前已有功能

- 保留 CCMax Claude、Kiro Claude、GPT、Gemini、国模、Grok、GPT Image 和 nano banana 8 个订阅系列及每系列 5 个按月档位。
- 保留人民币月卡售价、美元模型额度、日 / 周 / 月独立限额、订阅与充值分离、图片异步任务和对应模型资源包。
- 保留 OAuth/API Key 多账号管理、SubPilot 调度、精确计费、充值、订阅、视频工作台、无限画布及管理员账号 / 分组 / 套餐管理。

### 验证重点

- 已完成后端套餐快照与跨分组校验、管理员处理器、前端类型检查、前端生产构建和浏览器分配弹窗回归。
- 正式上线继续使用本地 Docker 镜像和安全蓝绿脚本，数据库、Redis、Caddy、SubPilot 不重启。
- 发布后复核并补发嵌入式前端资源，公网管理员订阅分包已确认包含具体套餐选择逻辑；生产应用容器已设置为 `unless-stopped`。

## v0.2.4-lingqu.6 - 2026-09-14

### 修复了什么

- 修复历史订阅套餐币种为空时回退显示美元、并可能按美元计算的问题；订阅售价默认使用人民币，套餐日 / 周 / 月额度仍按 USD 展示和扣减。
- 修复套餐编辑、用户订阅卡片和管理员套餐列表的币种默认不一致问题，避免刷新或重新保存后再次显示 `$`。

### 增加了什么

- 为 GPT Image 和 nano banana 两个生图订阅系列补充每张图片的额度消耗说明，并按档位展示 `$0.08 / $0.06 / $0.05` 与 `$0.15 / $0.13 / $0.12 / $0.10` 的单张额度。
- 将两个生图系列统一为 `GPT Image订阅` 和 `nano banana订阅`，改名后仍能显示正确的生图单张额度。
- 修复用户结算接口遗漏 `for_sale` 状态的问题，避免已上架套餐被错误显示为“暂不可用”。

### 当前已有功能

- 保留 CCMax Claude、Kiro Claude、GPT、Gemini、国模、Grok、GPT Image 和 nano banana 8 个订阅系列及每系列 5 个按月档位。
- 保留人民币月卡售价、美元模型额度、日 / 周 / 月独立限额、订阅与充值分离、图片异步任务和对应模型资源包。
- 保留 OAuth/API Key 多账号管理、SubPilot 调度、精确计费、充值、订阅、视频工作台、无限画布及管理员账号 / 分组管理。

### 验证重点

- 前端订阅卡片、管理员套餐列表和币种工具测试通过；覆盖两个生图系列十个档位及重命名后的系列名称。
- 后端订阅支付金额、空币种 CNY 默认和计划币种校验测试通过。
- 正式上线继续使用本地 Docker 镜像和安全蓝绿脚本，数据库、Redis、Caddy、SubPilot 不重启。

## v0.2.4-lingqu.5 - 2026-09-13

### 修复了什么

- 修复订阅卡片下方重复展示“套餐权益”的问题；通用额度信息不再与上方权益摘要重复出现。
- 修复套餐售价币种与额度币种混用的展示/支付兼容问题；售价按套餐币种显示，额度继续按 USD 展示，人民币套餐在人民币支付通道按原价扣款。

### 增加了什么

- 增加按产品系列和档位区分的套餐权益摘要文案，让 Basic、Plus、Standard、Pro、Ultra 的适用场景和使用价值清晰可辨。
- 补齐订阅目录商业化文案迁移，移除价格/市场价式描述，并从正式国模订阅权益中移除 Qwen。

### 当前已有功能

- 保留 CCMax Claude、Kiro Claude、GPT、Gemini、国模、Grok、GPT Image 2.5 和香蕉生图 8 个订阅系列及每系列 5 个按月档位。
- 保留日限额、周限额、月限额独立控制，订阅与充值分离，售价使用人民币、模型额度使用美元的展示约定。
- 保留 OAuth/API Key 多账号管理、SubPilot 调度、精确计费、充值、订阅、异步图片任务、视频工作台、无限画布及管理员账号/分组管理。

### 验证重点

- 前端订阅卡片、用户支付页、管理员套餐编辑测试通过（38 tests）；国际化完整性测试通过（3 tests）。
- 后端订阅支付金额测试通过；TypeScript 类型检查、生产前端构建和 ESLint 检查通过（0 errors，3 个既有 warning）。
- 正式上线继续使用本地 Docker 镜像和安全蓝绿脚本，数据库、Redis、Caddy、SubPilot 不重启。

## v0.2.4-lingqu.4 - 2026-09-13

### 修复了什么

- 修复订阅套餐把资源分组与用户商品混用的问题；用户侧现在只展示正式套餐，不再把每个档位当成普通分组。
- 修复套餐日、周、月额度展示与扣减链路，套餐额度按购买时快照独立限制，避免共享分组额度互相影响。
- 下架此前没有用户、Key 或用量记录的临时测试订阅配置；保留已有普通分组和账号数据。

### 增加了什么

- 增加 8 个订阅系列、40 个按月档位：CCMax Claude、Kiro Claude、GPT、Gemini、国模、Grok、GPT Image 2.5 和香蕉生图。
- 增加 GPT Image 2.5 与香蕉生图的按张数阶梯价格，以及所有系列的日限额、周限额、月限额。
- 增加订阅目录幂等迁移和误配测试分组的安全软删除迁移。

### 当前已有功能

- 保留 OAuth/API Key 多账号管理、SubPilot 调度、精确计费、充值、订阅、异步图片任务、视频工作台、无限画布及管理员账号/分组管理。
- 保留 GPT Image 2.5、Claude、GPT、Gemini、Grok、国产模型及 Composite 等现有接入能力。

### 验证重点

- 订阅目录迁移在生产数据库结构上以事务演练通过：8 个资源组、40 个套餐，失败自动回滚。
- 后端服务、处理器、仓储、路由定向测试通过；订阅卡片和支付页面测试通过。
- 正式上线使用本地 Docker 镜像和安全蓝绿脚本，数据库、Redis、Caddy、SubPilot 不重启。

## v0.2.4-lingqu.2 - 2026-09-12

### 修复了什么

- 修复超级管理员侧边栏遗漏“账号管理”入口的问题；恢复后可直接进入 `/admin/accounts` 管理上游 AI 账号。
- 补齐同样已注册路由但未展示在侧边栏的“视频账号配置”入口。

### 增加了什么

- 无新增功能，本次为管理员导航修复版本。

### 当前已有功能

- 保留 v0.2.4-lingqu.1 的模型接入、账号管理、分组管理、订阅/充值、图片与视频能力。
- 管理员菜单现在与已注册的管理员页面路由保持对应，账号管理和视频账号配置均可从左侧进入。

## v0.2.4-lingqu.1 - 2026-09-12

### 修复了什么

- 修复登录/注册链路的本地回归问题，补强邀请码并发注册的一次性占用，避免同一邀请码创建多个账号。
- 修复管理员分组创建请求无响应、账号配置弹窗国产协议字段异常和国产供应商模型候选列表错误回落到 Claude 列表的问题。
- 修复插件安装在 Windows 下重复关闭 ZIP 句柄导致安装失败的问题，以及 OpenAI Responses 压缩回退重复重试和模型不存在错误分类问题。
- 修复 OpenAI 运行时模型映射缓存不会随运行时设置版本失效的问题，并统一计费倍率应用路径。

### 增加了什么

- 增加 GPT、Claude、Gemini、Kimi、智谱、DeepSeek、Qwen 等模型品牌素材与新的首页/登录注册视觉布局。
- 增加 GPT CCMax/Kiro 订阅目录、订阅权益与优惠码相关数据结构、迁移和用户侧订阅/充值页面能力。
- 增加 GPT Image 2.5、视频工作台下游接口说明和无限画布图像接口兼容调整。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制、健康状态管理和 SubPilot 智能调度。
- OpenAI（含 GPT Image 2.5）、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、MiniMax、Antigravity 等平台接入，以及 Composite 多供应商分组。
- API Key 分发、精确计费、速率限制、用户/账号并发控制、管理后台、渠道监控、用量统计和支付充值。
- 用户订阅与充值分离页面、订阅权益、优惠码、异步图片任务、视频工作台、无限画布和 CDN/API 双域名访问。

### 验证重点

- `go test -tags=unit ./internal/service -count=1` 通过。
- 前端 ESLint、TypeScript 类型检查和生产构建通过；lint 仅保留 3 个 warning，无 error。
- 远端 `lingqu` SSH 认证未通过，本版本按本地镜像蓝绿发布流程处理，远端同步需在认证恢复后补做。
- 正式上线仍以本地 Docker 镜像构建、服务器候选容器健康、公网检查、Caddy 平滑切换和依赖容器未变化全部通过为准。

## v0.2.1-lingqu.1 - 2026-09-06

### 修复了什么

- 综合合并官方 Sub2API v0.2.1/main 的网关、计费、账号调度、图片回填和 OpenAI Responses 兼容性修复，同时保留 Lingqu 的 SubPilot 重试预算、利润控制、渠道池和 XiaoAPI 定制。
- 修复前端账号编辑时上游请求 ID 写回与本地提交锁、CN API 默认地址兼容逻辑的合并回归，保留 Grok 等非 CN 平台的原有默认配置。
- 补齐上游数据库迁移、Ent 运行时索引和前端渠道/分组配置序列化，避免新字段与现有本地字段冲突。

### 增加了什么

- 支持 GPT-6 Astra（`gpt-6`、`gpt-6-astra`）和 GPT-5.6 Sol/Terra/Luna 模型别名及能力识别。
- 增加 Codex 模型清单与账号投影、reasoning effort 乘数、渠道最大 reasoning effort、上游请求 ID、加密内容 lineage、OpenCode 会话和图片 base64 回填能力。
- 增加官方 v0.2.1 对应的数据库迁移、后台配置界面、请求追踪字段及相关回归测试。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制、健康状态管理和 SubPilot 智能调度。
- OpenAI（含 GPT-6 Astra/GPT-5.6）、Anthropic（含 Fable 5.1）、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- API Key 分发、精确计费、速率限制、用户/账号并发控制、管理后台、渠道监控、用量统计和支付充值。
- 异步图片任务、视频工作台、视频上游价格同步与阿里云 OSS 持久化、无限画布和 CDN/API 双域名访问。

### 验证重点

- 后端 `go test ./...` 通过；主前端 TypeScript 类型检查通过；受影响的账号、渠道、分组和上游请求 ID 测试通过。
- 主前端生产构建通过。全量前端套件仍有 14 个文件、24 个既有环境/断言失败，集中在未改动的首页、支付、注册和本地化测试，未作为本版本通过项隐瞒。
- 本条目仅记录本地合并和发布准备；正式上线仍必须按 Runbook 完成本地 Docker 镜像摘要、服务器上传、候选容器健康、公网检查和回滚点核对。

## v0.2.0-lingqu.2 - 2026-09-03

### 修复了什么

- 修复 SubPilot 切号重试会让每个新账号重新获得完整超时的问题；现在一次请求的总重试预算在切号、并发槽等待和 OpenAI 首 token 阶段共享且只能收紧。
- 修复 WebSocket 首轮凭证成功后仍携带旧连接级预算的问题；首轮接入完成后清除连接级预算，避免影响后续独立的 `response.create` 请求。
- 兼容旧版 SubPilot 未返回剩余预算的响应，未提供预算时保持原有超时行为。

### 增加了什么

- SubPilot 选择响应增加剩余重试预算元数据，并在 Gavin2API 内统一传播到请求上下文。
- 增加预算上限、预算耗尽停止切号、槽位等待上限、首 token 上限和 WebSocket 预算生命周期的回归测试。
- 无其他独立业务功能新增；本版本专注于慢请求故障转移和超时一致性。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制、健康状态管理和 SubPilot 智能调度。
- OpenAI、Anthropic（含 Fable 5.1）、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- API Key 分发、精确计费、速率限制、用户/账号并发控制、管理后台、渠道监控、用量统计和支付充值。
- 异步图片任务、视频工作台、视频上游价格同步与阿里云 OSS 持久化、无限画布和 CDN/API 双域名访问。

### 验证重点

- Gavin2API `go test ./internal/service`、`go test ./internal/handler` 及 SubPilot `go test ./...` 通过。
- 正式发布必须由本标签触发 GitHub Actions，使用成功 Actions 生成的 GHCR 镜像；部署后再执行容器、内部 `/health`、公网健康和关键 API 冒烟检查。
- 发布范围仅替换 Gavin2API 应用容器；PostgreSQL、Redis、Caddy、SubPilot、Docker 卷、网络和凭据保持不变。

## v0.2.0-lingqu.1 - 2026-09-02

### 修复了什么

- 合并官方 Sub2API v0.2.0 的上游修复和稳定性改进，并保留现有 Lingqu AI、SubPilot、视频与无限画布定制能力。
- 修复 Claude Fable 5.1 账号协议适配，确保 Fable 5.1 模型可以沿用现有账号调度、计费和健康检查链路。

### 增加了什么

- 支持 Claude Fable 5.1 的模型映射、请求转发和能力识别。
- 同步官方 v0.2.0 的模型目录、账号限制、用量字段及 OpenAI Responses/Compact/WebSocket 兼容性更新。
- 无其他独立业务功能新增；本版本重点是官方版本合并和 Fable 5.1 兼容。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制、健康状态管理和 SubPilot 智能调度。
- OpenAI、Anthropic（含 Fable 5.1）、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- API Key 分发、精确计费、速率限制、用户/账号并发控制、管理后台、渠道监控、用量统计和支付充值。
- 异步图片任务、视频工作台、视频上游价格同步与阿里云 OSS 持久化、无限画布和 CDN/API 双域名访问。

### 验证重点

- 后端 `go test ./...`、主前端 TypeScript 类型检查和 Docker 生产镜像构建通过。
- 正式部署使用本地构建镜像进行蓝绿预热；确认候选健康、内部接口和公网 API/CDN 检查通过后再切换 Caddy，数据库、Redis、Caddy 和 SubPilot 保持原容器不变。

## v0.1.186-lingqu.3 - 2026-09-01

### 修复了什么

- 修复管理员批量删除账号时并发执行多个删除事务，导致出现“仅 1 个成功、其余失败”的问题；现在逐项执行，单个账号失败不会阻断后续账号。
- 修复阿里云 OSS 视频持久化配置误用数据库备份 S3 配置的风险，视频存储改为独立的阿里云 OSS 配置和原生 SDK。
- 批量删除失败现在记录账号 ID、根账号 ID 和具体错误，管理界面保留失败账号并显示首个失败原因。

### 增加了什么

- 系统设置支持独立配置阿里云 OSS 的 Endpoint、Region、Bucket、AccessKey ID、AccessKey Secret、对象前缀及启用用户。
- 视频任务在创建时记录 OSS 使用策略；已选择 OSS 的视频不参与本地定期清理，并继续通过鉴权接口访问。
- 无新增其他独立业务功能。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制、健康状态管理和安全的批量操作。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务、视频工作台、阿里云 OSS 视频持久化和无限画布。

### 验证重点

- 后端账号批量删除定向测试通过，覆盖串行执行、单项失败后继续、影子账号联动和请求取消场景。
- 后端测试、主前端类型检查、Lint 和生产构建通过。
- 正式部署依照 Runbook 使用本地构建镜像进行蓝绿预热，确认健康后平滑切换 Caddy；不重启数据库、Redis、Caddy 或 SubPilot。
- 生产实录：镜像 `local/gavin2api:0.1.186-lingqu.3-3953d796`（`sha256:ed6c5ae6...668b710`）已切换，API/CDN `/health` 连续验证为 `200`，未授权 `/v1/models` 为 `401`。
- 事故注记：首次 Caddy reload 读到既有单文件 bind mount 的旧 inode，短暂产生 `502`；已立即回滚并确认公网恢复后，改用 Caddy 管理 API 加载已校验配置完成切换。详见 `docs/RELEASE_INCIDENT_20260901_CN.md`。

## v0.1.186-lingqu.2 - 2026-09-01

### 修复了什么

- 修复生成视频只能依赖上游临时文件保存，历史任务可能因定期清理或上游回收而失效的问题。
- 优化无限画布文本生成配置，渠道和模型选择器改为同排显示，较长模型名自动截断。
- 合并官方 Sub2API `v0.1.184` 的上游修复，并与现有 SubPilot 租约、会话和计费用量字段兼容。
- 修复原生 OpenAI Responses v2 请求的账号能力选择，以及 WebSocket 内容安全命中后会话屏蔽标记不能及时写入的问题。

### 增加了什么

- 系统设置新增独立的阿里云 OSS 视频持久化配置，使用阿里云 OSS 原生 SDK，不复用数据库备份的 S3 配置。
- 支持按用户勾选视频 OSS 使用范围；新任务会记录当时的存储策略，启用 OSS 的视频不再进入本地定期删除流程。
- 视频历史和下载接口继续通过平台鉴权访问已持久化的成品文件，并提供 OSS 连通性测试。
- 接入官方新增的用量字段、账户限制、上游模型目录同步、OpenAI Responses/Compact 与 WebSocket 兼容性改进。
- 无新增独立业务功能；本版本主要是官方合并与稳定性更新。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制和健康状态管理。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务、视频工作台、视频 OSS 持久化和无限画布。

### 验证重点

- 视频 OSS 配置、用户选择、任务持久化和下载链路测试通过；嵌入式前端类型检查和生产构建通过。
- 后端 `go test ./...`、主前端类型检查、Lint 和管理后台定向测试通过。
- 正式部署依照 Runbook 使用本地构建镜像进行蓝绿预热，确认健康后平滑切换 Caddy；不重启数据库、Redis、Caddy 或 SubPilot。

## v0.1.186-lingqu.1 - 2026-09-01

### 修复了什么

- 修复无限画布文本节点在共享模型字段残留生图模型时可能选错渠道的问题。
- 修复同一 OpenAI 分组存在多个 Key 时重复显示分组的问题。
- 修复系统托管模式下无有效文本模型时静默回退到默认模型的问题。

### 增加了什么

- 文本创作桥接现在按 OpenAI 分组读取并展示各分组实际返回的 `/v1/models` 模型列表。
- 视频账号支持独立售价加价倍率，可按账号保存并用于视频模型售价同步。
- 发布 Runbook 增加本地构建、镜像上传、蓝绿预热、Caddy 平滑切换和数据库/依赖保护规则。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制和健康状态管理。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务、视频工作台和无限画布。

### 验证重点

- 文本 Key 访问控制测试、主前端类型检查、画布类型检查和嵌入式前端生产构建通过。
- 发布流程明确禁止数据库、Redis、Caddy 和 SubPilot 的无必要重启，失败时优先回切旧应用容器。

## v0.1.186 - 2026-08-31

### 修复了什么

- 修复无限画布反推提示词可能沿用共享生图模型字段，导致文本请求错误使用生图渠道或 Key 的问题。
- 修复系统托管模式下缺少文本能力模型时静默回退到默认模型的问题，避免请求被路由到错误渠道。

### 增加了什么

- 增加当前用户可用 OpenAI 文本 Key 的独立桥接与渠道展示，普通文本分组优先于生图分组。
- 更新无限画布生产资源，文本生成会明确使用系统注入的文本渠道，用户仍不能填写接口地址或 API Key。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制和健康状态管理。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务、视频工作台和无限画布。

### 验证重点

- 文本 Key 访问控制测试、宿主前端类型检查、画布类型检查全部通过。
- 画布和主站生产构建通过；生产部署只替换网关应用容器，不触碰数据库、Redis 或持久化数据卷。

## v0.1.185 - 2026-08-31

### 修复了什么

- 修复 pnpm 9 执行独立 `embedded/infinite-canvas` 项目时因 workspace 配置缺少 `packages` 字段而导致正式发布前端构建失败的问题。
- 延续并包含 v0.1.184 中已完成的 SubPilot 租约释放修复，确保 OpenAI 成功请求和内容策略拒绝都不会长期占用调度租约。

### 增加了什么

- 无新增业务功能。
- 为独立前端子项目补充明确的 workspace 根包声明，并以 pnpm 9 完成安装和生产构建回归验证。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制和健康状态管理。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务和视频工作台。

### 验证重点

- GitHub Actions 的后端测试、前端测试、类型检查、Lint 和前端生产构建全部通过后才部署。
- 生产部署仅替换网关应用容器，保留数据库、Redis、Caddy、SubPilot 网络和数据卷不变。

> v0.1.184 发布尝试因前端 CI 构建配置错误失败，未生成可发布镜像，也未部署生产；本条目记录该问题在 v0.1.185 中的修复。

## v0.1.184 - 2026-08-30

### 修复了什么

- 修复 OpenAI 请求成功后只释放 Sub2API Redis 并发槽、未及时释放 SubPilot 调度租约的问题。此前租约会继续占用最长 60 秒，导致 OAuth 账号在 SubPilot 中被误判为并发已满，成本优先模式因而绕过更便宜且健康的 OAuth 账号。
- 修复内容策略拒绝不计入账号健康时遗漏释放 SubPilot 租约的问题。现在该类请求仍不会污染账号健康分，但会立即归还并发租约。
- 修复正式发布工作流的 Go 版本门禁与 `backend/go.mod` 不一致的问题，后端测试和发布构建统一校验 Go 1.27.0。
- 修复账号创建弹窗在分组长上下文阶梯定价全部开启时仍显示冗余账号开关、且 WebSocket 模式测试标识丢失的问题。
- 修复主前端 ESLint 误扫描独立的 `embedded/infinite-canvas` 子项目导致发布 CI 被无关规则错误阻断的问题。

### 增加了什么

- 无新增业务功能。本版本专注于并发租约一致性和成本优先调度恢复。
- 增加 OpenAI 成功回报释放租约、内容策略失败释放租约的集成回归测试。
- 增加发布日志强制规范，后续每次发布都必须记录修复、新增和当前已有功能。

### 当前已有功能

- OAuth 与 API Key 多账号管理，支持账号级并发限制和健康状态管理。
- SubPilot 智能调度集成，支持成本优先、稳定优先和综合调度，并支持粘性会话与故障转移。
- API Key 分发、精确计费、速率限制和用户/账号并发控制。
- OpenAI、Anthropic、Gemini、Grok、Kimi、智谱、DeepSeek、Antigravity 等平台接入，以及 Composite 多供应商分组。
- 管理后台、渠道监控、用量统计、支付充值、异步图片任务和视频工作台。

### 验证重点

- OpenAI 请求结束后，Sub2API 显示的账号并发与 SubPilot 内部租约应同步下降。
- 成本优先模式在健康门槛内应优先选择成本更低的可用账号，不应因过期租约持续调度高成本账号。
- 内容策略拒绝不降低账号健康度，但必须即时释放对应租约。
