# Gavin2API 发布日志

每次正式发布都必须新增版本条目，并分别写清楚“修复了什么”、“增加了什么”和“当前已有功能”。没有新增功能时也必须明确记录。

## v0.2.14-lingqu.2 - 2026-10-08

单点修复版：堵上「操作员禁用账号仍被调用」的单账号分组绕过口。不含新功能。

### 修复了什么

- **🔴 单账号分组的强制路由会无视操作员的显式禁用**（2026-10-06 由 804/kiro-95 事故定案，
  至此才真正修掉）。
  `tryForceSingleAccountGroup`（`backend/internal/service/gateway_scheduling.go`）在"分组恰好绑定 1 个账号"时
  直接把该账号推去调度，取到账号后**从不检查 `status` / `schedulable`**
  （`GetSingleAccountIDByGroupID` 的 SQL 也没 join accounts）。于是 `status=disabled`、
  `schedulable=false` 的账号仍会被选中并真实发出流量。

  修法：取到账号后校验两项**操作员显式控制**——`!account.IsActive() || !account.Schedulable` 时
  **不走强制路径**（返回 `forced=false`），交回正常调度，由它给出"无可用账号"或改选其它账号；
  日志记 `single_account_group_skip_disabled_account`。
  刻意**不**套 `isAccountSchedulableForSelection`：那个口径还包含限流/过载/冷却等瞬态窗口，
  而这条路径本来就要靠等待计划兜住瞬态不可用，一并拦掉会改变既有行为。

  保留原意：该路径仍会绕过**运行时**状态（限流窗口、冷却、SubPilot 不参与）——
  这是 2026-07-26 `cec350f3a` 引入时的明确设计（"fix: force routing for single-account groups"），
  当时只是**误把"运行时状态"与"操作员禁用"混为一谈**：既有测试 `bypasses runtime status and exclusions`
  直接拿一个 `disabled + schedulable=false` 的账号当例子。本版把测试改成"active 但在限流窗口内"
  来保住原意，并新增两条用例锁住"禁用必须生效"。

- **顺带修掉 `unit` 构建标签测试套件编译不过**：`account_test_service_openai_test.go` 有 17 处
  `testOpenAIAccountConnection` 调用少传 `maxTokens` / `reasoningEffort` 两个参数（函数签名早已变更），
  导致 `go test -tags unit ./internal/service/` **整包编译失败**——这套测试等于一直是死代码。
  已按生产调用点（`testOpts.MaxTokens, testOpts.ReasoningEffort`）补上 `0, ""`（走默认值）。

### 影响面核对（发布前）

- 生产 `accounts.status` 分布：active 700 / error 321 / disabled 1，**没有空状态**账号 →
  严格判定不会误伤（测试 fixture 里的零值账号已单独补齐 `Status`/`Schedulable`）。
- 全库共 6 个"只绑 1 个账号"的分组：36 `jp-gemini-image`、73 `gemini-image`、142 `国模订阅`、
  145 `nano banana订阅`、146 `openai-官key`、148 `正价kiro`。
  其中 36/73/145 绑定的账号 884（`nicolessss-gemini`）当前是 `active + schedulable=false`
  （今天 20:17 被关掉），修复后这 3 个分组将不再派发到它。
  **实测近 3 小时这 3 个分组与账号 884 的请求量均为 0**，故当前无客户影响；若有流量需先确认 884 的去留。
- 未修的同类绕过口：DB 函数 `sync_dynamic_rate_group` 的 INSERT 不校验 `status/schedulable`
  （动态速率分组自动绑定账号），属另一条路径，仍待处理。

### 验证结果

- `go build ./...`、`go vet ./...` 全绿。
- `TestGatewaySingleAccountGroupForce` 5 个子用例全过（含新增 2 条禁用用例，
  与改写后的"绕过运行时状态"用例）。
- ⚠️ `go test -tags unit ./internal/service/` **有 4 个既有失败**（与本次改动无关，
  因上面那个编译问题此前从未真正跑起来）：`TestAdminService_CNProviderModelsListCandidatesUseProviderDefaults`、
  `TestBatchImagePublicService_Submit`（2 个子用例）、`TestOllamaProbeCallback_StaleLongDoesNotOverrideNewShort`。
  单独排期修复，未在本版处理。
- 前端未改动（沿用 v0.2.14-lingqu.1 的产物）。

### 当前已有功能

与 `0.2.14-lingqu.1` 完全一致（EasyPay 伪造回调安全修复、setup 加固、远程 Codex 目录发现、
前端依赖审计修复等全部保留），本版只收窄了单账号分组的强制路由条件。

## v0.2.14-lingqu.1 - 2026-10-07

本版为**合并上游 Sub2API v0.2.14 后的首个发版**，核心是修复 EasyPay 伪造回调安全漏洞（#7881）。
发布提交 `47d46777d` 之后的合并提交 `fc03998fd`（双父 `47d46777d` + `0363b8cdb`），合并基线即 v0.2.13（`b8dece900`），**零冲突**。

### 修复了什么

- **🔴 EasyPay 下单签名可被重放为支付成功回调（#7881，安全漏洞）**。
  攻击者下单后可在 submit.php 弹窗 URL 里看到下单签名，把它原样重放成支付成功通知即可到账，**全程不需要商户密钥**。
  两个叠加缺陷：① `CanonicalizeReturnURL` 保留客户端查询参数，攻击者让 `return_url` 以 `&trade_status=TRADE_SUCCESS`
  结尾并进入签名串；② EasyPay 签名基串不做转义直接拼接，回调把 return_url 解码到一半、被夹带的键值对升级为顶层参数后，
  重排序的签名基串与下单时逐字节一致，验签通过。修复（上游 `d1aac6b98`）：
  - `CanonicalizeReturnURL` 丢弃客户端查询参数（`parsed.RawQuery = ""`）——服务端自己构造签名集，客户端查询无合法用途；
  - `VerifyNotification` 只接受真正的异步通知参数集
    （pid/trade_no/out_trade_no/type/name/money/trade_status/param/sign/sign_type），其余参数一律拒绝（fail closed）。
  - 上游附带回归测试（`easypay_notify_security_test.go`，覆盖 PoC 载荷/整单 URL 重放/真实回调/未知参数拒绝/return_url 夹带剥离）。
  被上游判定误拒的真实支付仍可走既有 QueryOrder 对账路径恢复。
  **发布前审计结论：该洞在我方代码中存在但未被利用** —— 生产 93 笔 COMPLETED 订单 100% 携带 28 位真支付宝流水号，
  0 笔有伪造特征；webhook 仅被低强度扫描（7 次 400，无一过验签）。
- **前端依赖审计修复**（`0f9d460dc`）：vue、source-map-js、xlsx 豁免调整（`.github/audit-exceptions.yml`），`pnpm-lock.yaml` 同步。
- **测试对齐（本地提交 `f6293250f`）**：上游 `bbba01dae` 改了 `UseKeyModal.vue`（codex 目录发现）但没同步更新测试，
  导致 v0.2.14 自带测试与组件不一致（上游带病发布）。本版对齐测试断言（`[features]` 新增 `api_key_model_discovery`）。

### 增加了什么

- **setup 加固（#7850 / PR #7893，上游 `d97ccc952`）**：全新安装不再出厂可猜测的管理员凭据
  （`backend/internal/setup/` cli/handler/setup 重构，+236 行测试）。**对已安装实例无行为变化**。
- **远程 Codex 目录 API-key 发现（#7851 相关，上游 `bbba01dae`）**：codex CLI 配置在 `[features]` 中新增
  `api_key_model_discovery = true`、`[model_providers.OpenAI]` 中新增 `model_catalog_url`（远程目录发现）。
- 无数据库迁移（本版 0 条新迁移，回滚无 DB 顾虑）。

### 当前已有功能

- 与 `0.2.13-lingqu.2` 一致并全部保留：EasyPay 渠道（zpayz.cn）、充值赠送、线下提现幂等、TypeSafe 平台、
  xiaoapi 平台配额白名单（12 平台）、内容审核 risk-control 仅记录、简易模式限流、在途余额预留、备份归档、
  Composite 路由、Claude Code 版本同步、opencode-go 用量、Grok CLI 门槛、Antigravity 脱敏、Cyber policy allowlist、
  小视频工作台、SubPilot 租约释放/内池耗尽 503/池模式同账号重试、`gavin2api` 稳定别名部署等。

### 验证重点

- **EasyPay 安全修复生效**：向 `/api/v1/payment/webhook/easypay` 发送带非标准参数（如 `return_url`、`notify_url`、`device`）
  的通知必须被拒（日志出现 `unexpected notify param`）；标准参数 + 正确签名的真实回调仍通过。
- **回归**：EasyPay 真实订单（zpayz.cn 恢复后）下单→支付→回调→到账全链路不受影响。
- **setup**：已安装实例无变化（不触发 bootstrap）。
- **回滚提示**：本版无迁移，回滚到 `0.2.13-lingqu.2`（容器 `gavin2api-release-0.2.13-lingqu.2-47d46777d`，只 stop 不删）零 DB 顾虑。
- ⚠️ **门禁基线说明**：前端 vitest 存在 **16 个合并前既有红**（PaymentView 19、RedeemView 11 等，已在合并前提交
  `47d46777d` 上实测复现为基线，与本合并无关，单独排期修复）；本合并引入的唯一测试回归（UseKeyModal，上游测试未同步）
  已在本版修平（30/30 通过）。Go 门禁全绿（52 包，含新增安全回归测试）。

## v0.2.13-lingqu.2 - 2026-10-06

发布提交 `f61dd5d65`（上一个发版提交 `9876782d5`）。本版为**单点修复版**，只包含 xiaoapi 平台白名单漂移的修复，不含新功能。

### 修复了什么

- **平台配额白名单三处副本漂移，导致 xiaoapi 平台配额「能选不能存」**。
  平台白名单存在三处副本：① `service.AllowedQuotaPlatforms`（单一权威源）② `ent/schema/user_platform_quota.go`
  的构建期 `Validate` ③ 数据库 `CHECK` 约束（由迁移维护）。本地视频平台 `xiaoapi` **只在 ① 里**：
  237/238/241 三次平台迁移各自照抄了当时的列表，从未包含 xiaoapi。后果是管理后台的用户平台配额弹窗会
  正常渲染出 xiaoapi 一行并允许填写，但保存时先被 ent 构建期校验拒绝；即使绕过，也会被
  `user_platform_quotas_platform_check` 拒绝。
  - 修法一：`ent/schema/user_platform_quota.go` 的 `Validate` switch 补上 `"xiaoapi"`。
    （该处的校验闭包在运行时由 schema 描述符提供，生成代码只引用 `validators[N]`，因此**不需要重新生成 ent 代码**。）
  - 修法二：新增迁移 **`250_add_xiaoapi_quota_platform.sql`** —— `DROP CONSTRAINT IF EXISTS` 后重建
    `user_platform_quotas.platform` 与 `composite_model_routes.target_platform` 两个 CHECK 约束，
    列表 = 241 的 11 平台 + `xiaoapi` = `service.AllowedQuotaPlatforms` 的 **12 平台**。
    与 241 同型，是**严格超集**，存量行瞬时校验通过；`user_platform_quotas` 本库 0 行、
    `composite_model_routes` 7 行且全为 `openai`，不存在因存量行失败的路径。
- **防回归（关键）**：此漂移此前**不会让任何既有测试变红**，这正是它能潜伏的原因。本版补两条测试：
  - `internal/service/platform_quota_sync_test.go` —— **同步守卫**：断言 `AllowedQuotaPlatforms` 中每个平台
    都出现在 ent schema 的 `Validate` 里。已**反向验证非空转**（临时移除 `"xiaoapi"` 该测试立即失败）。
  - `migrations/xiaoapi_platform_migration_test.go` —— 锁定 250 号迁移的 SQL 文本与 12 平台列表。

### 增加了什么

- 无新增业务功能。
- 新增 1 条数据库迁移 `250_add_xiaoapi_quota_platform.sql`（只放宽 CHECK 约束，不改表结构、不动计费数据）。

### 当前已有功能

- 与 `0.2.13-lingqu.1` 完全一致（本版未改任何业务逻辑）：充值赠送（recharge bonus）、线下提现幂等、
  TypeSafe 平台接入、内容审核 risk-control 仅记录模式、简易模式 API Key 限流计费、在途余额预留、
  备份归档与保留策略、Composite 路由解析器接入 OpenAI 网关、Claude Code 版本同步、Claude reset credits、
  opencode-go 用量服务、Grok CLI 版本门槛对齐、Antigravity 上游错误脱敏、Cyber policy allowlist。
- 本地扩展全部保留：`xiaoapi` 视频平台与只读探针、线下结算（`offline_settlement`）、账号共享模式结算、
  OpenAI 网关在途余额预留接入、SubPilot 内部探针路由、grok 视频 pending 计费快照、SubPilot 租约释放、
  内池耗尽 503 `upstream_pool_exhausted`、池模式 401/403 同账号重试、`gavin2api` 稳定网络别名部署。

### 验证重点

- **迁移核实（本版核心）**：候选启动后 `schema_migrations` 应出现 `250_add_xiaoapi_quota_platform.sql`；
  `pg_get_constraintdef` 查 `user_platform_quotas_platform_check` 与 `composite_model_routes_target_platform_check`
  都必须包含 `xiaoapi`，且两个约束列表长度一致（12）。
- **回归验证**：管理后台为某用户保存一条 `xiaoapi` 平台配额应成功（此前必然失败）。
- ⚠️ **回滚提示**：本版只新增 1 条**放宽约束**的迁移，回滚到 `0.2.13-lingqu.1`（容器
  `gavin2api-release-0.2.13-lingqu.1-9876782d5`，只 stop 不删）**不会**因该迁移失败 ——
  回滚后唯一影响是 `xiaoapi` 平台配额又变回不可保存。
- 切流手法：稳定别名金丝雀；退役旧容器用优雅 `docker stop`。

### 发布批次内的运维动作（非代码变更，随本版一并记录）

- 账号 **804（kiro-95）** 已于本次发布前停止被调用：`status=disabled`/`schedulable=false` 无法阻止它被选中
  （gavin2api 的「单账号分组强制路径」与「动态费率分组自动绑定」都不检查账号状态，且它是 `pool_mode`
  账号导致错误不标记本地状态、永不冷却）。已通过 `subpilot_channel_configs.dynamic_group_enabled=false`
  把它从动态费率分组自动分配中排除，验证后调用量归零。
  ⚠️ 该账号原所属分组 `claude-kiro-vip`(id=8) 已由管理员软删除。
- 陈旧回滚容器 `0.2.7-lingqu.1-150be00d2`、`0.2.5-lingqu.4` 已清理，只保留 1 个有效回滚点。

## v0.2.13-lingqu.1 - 2026-10-06

本版为**合并上游 Sub2API v0.2.13（`b8dece900`）后的首个发版**，合并提交 `a63f3cde4`（本地上一版 `08cc24322`，合并基线 `20a94fbb5`）。
规模：上游新增 246 提交，双边改动文件交集 159，冲突文件 36 / 冲突块 60。**本次合并本身没有引入新功能**，功能增量全部来自上游。

### 修复了什么

本版修掉的都是「git 自动合并静默吃掉本地改动 / 并集语义未对齐」，其中数条会导致**线上功能静默失效或漏计费**：

- **`payment_orders.bonus_amount` 列在本地生成代码里不存在**。上游新增该列，本地新增 `provider_instance_id`/`provider_key`/`provider_snapshot`，两侧列索引不同（本地 42 / 上游 41 / 并集 43）。手工改索引必错，本版用 ent v0.14.5 重新生成 `ent/` 代码后拿到一致结果（`PaymentOrderMutation.Fields()` 容量 41 → 42，`PaymentOrdersColumns[42]`）。**不重新生成则上游充值赠送金额会静默不落库。**
- **`affiliate_repo.go`**：本地线下结算 `ClearAvailableQuotaOfflineSettlement` 与上游线下提现 `WithdrawQuota` 被 diff3 **交错进同一段函数体**，两个能力都残缺；提取记录 SQL 的 action 白名单只剩上游的 `('transfer','withdraw')` → **本地 `offline_settlement` 流水会从后台列表整体消失**。已整块重写为两个完整函数，白名单补成三值并集。
- **`affiliate_service.go`**：`AffiliateTransferRecord.Action` 字段被上游字段块**整段复制写入两遍** → 后端编译失败。
- **`content_moderation_repo.go`**：与 `affiliate_repo.go` 出现**同名不同签名的 `nullableInt64Ptr`** → 重复声明；本地版改名 `nullableInt64Any`。
- **`wire_gen.go` 注入链三处**（`ProvideAdminHandlers` / `ProvideOpenAIGatewayHandler` / `provideCleanup`）两侧都加参数 → 按并集补齐（`openCodeGoUsage` / `claudeResetCredit` / `claudeCodeVersionSync` / `compositeRouteResolver` / `imageAsyncTaskStore` / `xiaoVideoRuntime` / `communityBilling`），并去掉重复的 `idempotencyCoordinator`。
- **`payment_order.go`**：上游把订单金额计算后移到 `methodCurrency` 解析之后并引入 `quoteRechargeBonus`，本地有 `planCurrency` + 优惠码折扣 → plan 分支保留本地促销折扣，balance 分支采用上游赠金报价。
- **`content_moderation.go`**：保留本地账号共享/会员归属字段（`ScopeType`/`AccountShareListingID`/`AccountID`/`OwnerUserID`/`ConsumerUserID`/`MembershipID`），同时接纳上游 `riskControlLogOnly` 的 mode 覆盖。
- **`gemini_v1beta_handler.go`**：保留本地 SubPilot 失败回报，同时接纳上游 `failoverClientGone` 的 499 标记。
- **前端 `PaymentView.vue`**：`marked` / `DOMPurify` / `announcement-markdown.css` 的 import 被合并吃掉（组件仍在用），同时存在与本地重复的 `subscriptionEnabled` 和未引用的 `AppLayout`。已保留本地 `UserWorkspaceLayout` 布局，补回上述 import 与 `renderedHelpText` / `renderedBonusNotice`。
- **前端 `AdminAffiliateRecordsTable.vue`**：合并后**出现两个 `<template #cell-action>`**（Vue 会静默取一个）→ 合并为一处，三类流水文案统一。
- **前端 `DateRangePicker.vue`**：保留本地 Teleport + fixed 定位版式；`today` 统一为函数式（模板与预设调用的是 `today()`）。
- **前端 `affiliates.ts`**：`AffiliateTransferRecord.action` 重复声明去重，并集为 `transfer | offline_settlement | withdraw`。
- **前端平台清单**统一为 **12 平台并集**（本地 `xiaoapi` + 上游 `typesafe`，与后端 `service.AllowedQuotaPlatforms` 一致）。

### 增加了什么

以下均为**上游 v0.2.5 → v0.2.13 引入**，本版随合并带入：

- **充值赠送（recharge bonus）**：赠送阶梯配置与文案，`payment_orders.bonus_amount` 落列，前端 `utils/rechargeBonus` 与金额卡展示。新增迁移 `241_add_payment_order_bonus_amount.sql`。
- **线下提现登记（affiliate withdraw）幂等**：管理员登记站外打款，同 `Idempotency-Key` 只扣减一次。新增迁移 `240_affiliate_ledger_operation_id.sql`（加列 + 部分唯一索引）。
- **TypeSafe（Jev System One）平台接入**：平台常量、账号测试服务（`account_test_service_typesafe.go`）、前端平台清单。新增迁移 `241_add_typesafe_platform.sql`。
- **内容审核 risk-control 仅记录模式**（`riskControlLogOnly`）：命中后只记日志不改判定。
- **简易模式 API Key 限流计费路径**（`SimpleModeKeyRateLimitOnly`）：只累计 5h/1d/7d 窗口用量，不触发余额/订阅/账号/平台/终身额度效果。
- **在途余额预留**（inflight balance reservation）：`billing_inflight_reservation.go` / `gateway_inflight_reservation.go`，余额模式下按估算预占、计费后释放。
- **备份归档与保留策略**：`backup_restore_state.go` / `backup_retention.go` 与前端 `BackupArchiveSettings.vue`。
- **Composite 路由解析器接入 OpenAI 网关**（`compositeRouteResolver`）；**Claude Code 版本同步服务**与 Claude reset credits；**opencode-go 用量服务**。
- **Grok CLI 版本门槛与官方无界面请求头对齐**；**Antigravity 上游错误脱敏**；**Cyber policy allowlist**（`cyber_policy_allowlist.go` / `openai_cyber_allowlist.go`）。
- 前端 **Axios 升级至 1.20.0**（修复前端安全审计）、平台配额/看板/支付页若干改进。

### 当前已有功能

- 与 `0.2.7-lingqu.2` 一致并全部保留：内容审核 `engine_meta` 审计溯源、渠道 reasoning effort 分档计费倍率、订阅目录与配额、插件、channel monitor v2、opencode-go / minimax 平台接入、Grok 视频 pending 计费快照、SubPilot 租约释放与内池耗尽 503 `upstream_pool_exhausted` 分支、池模式 401/403 同账号重试、`gavin2api` 稳定网络别名部署、账号共享/商城/积分/社区、小视频工作台与异步图片任务等。
- 本版额外保留的本地扩展：`xiaoapi` 视频平台与只读探针、线下结算（`offline_settlement`）、账号共享模式结算、OpenAI 网关在途余额预留接入、SubPilot 内部探针路由。

### 验证重点

- **充值赠送**：余额充值命中赠送阶梯时，`payment_orders.amount` 为到账总额、`bonus_amount` 为赠送额，返利基数剔除正确。
- **线下提现幂等**：同 `Idempotency-Key` 重复登记不重复扣减；提取记录列表同时列出 `transfer` / `offline_settlement` / `withdraw` 三类流水。
- **TypeSafe 平台**：`user_platform_quotas` / `composite_model_routes` 的 CHECK 约束重建后包含 `typesafe`；存量行瞬时校验通过。
- **原有本地能力不回退**：grok 视频 pending 计费快照、SubPilot 租约释放（3 处 `RecordUsage`）、内池耗尽 503、池模式同账号重试仍在位（已用符号计数审计前后对照）。
- ⚠️ **迁移风险已实测**：本库 `user_platform_quotas` 0 行、`composite_model_routes` 7 行且全部为 `openai`、`payment_orders` 107 行、`user_affiliate_ledger` 12 行 → 三条新迁移均为安全操作（近乎空操作），`ADD CONSTRAINT` 不会因存量行失败。
- ⚠️ **回滚提示**：本版**新增 3 条迁移**（240_affiliate_ledger_operation_id、241_add_payment_order_bonus_amount、241_add_typesafe_platform）。回滚到 `0.2.7-lingqu.2`（容器 `gavin2api-release-0.2.7-lingqu.2-dc3d8e3f4`，只 stop 不删）**不会**因这三条迁移失败——它们只加列/加索引/放宽 CHECK，属向后兼容；旧版本忽略新列即可，唯一影响是旧版本不再校验 `typesafe` 平台。
- 切流手法：稳定别名金丝雀；**退役旧容器用优雅 `docker stop`（SIGTERM → FIN），不使用 `docker network disconnect`**。别名交接后必须显式验证候选的 redis/postgres 名称解析与一次业务请求。
- ⚠️ 遗留（**合并前既有，非本版引入**）：`service.AllowedQuotaPlatforms` 含 `xiaoapi`，但 ent 构建期校验（`ent/schema/user_platform_quota.go`）与数据库 CHECK 约束都**不含** `xiaoapi` → 目前为用户配置 xiaoapi 平台配额会失败。已用三方对照确认 `08cc24322` 同样如此。本版按最小改动未处理，待单独修。

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
