# 生产发布事故复盘：v0.1.186-lingqu.2 短暂 502

日期：2026-09-01（Asia/Shanghai）

项目：Gavin2API 生产环境

发布版本：`0.1.186-lingqu.2`
发布提交：`130ce12225514f5ae8439a26c325f40d004e5a8d`

## 1. 事故结论

本次发布最终完成，但切换过程中发生了可避免的线上 502。候选应用已经启动并健康，然而在 Caddy 的实际运行配置确认生效前，旧线上应用容器被停止。Caddy 仍尝试访问已停止的旧 upstream，于是 API 和 CDN 短暂返回 HTTP 502。

这违反了项目的核心发布要求：平滑发布必须保证旧版本持续承载流量，直到新版本已经完成切流并通过公网验证。此次事故没有重启 PostgreSQL、Redis、Caddy 或 SubPilot，也没有发现数据库数据丢失。

## 2. 时间线

以下时间以服务器 UTC 日志和发布会话记录为准：

1. 本地 `main` 完成官方 Sub2API `v0.1.184` 合并及本地功能，构建 `0.1.186-lingqu.2` 的 Linux amd64 镜像。
2. 候选应用使用独立容器名启动，健康检查和内部 `/health` 通过。
3. 首次候选启动发现生产数据库连接数紧张，且生产环境跳过自动迁移；候选被停止，释放历史候选容器占用的连接。
4. 通过一次性迁移容器执行 4 个 `231_*` 增量迁移，确认 OSS 视频存储、用量字段和用户限制字段存在。迁移容器随后停止并移除。
5. 候选应用重新启动并通过健康检查，启动日志无致命、数据库或 Redis 连接错误。
6. 尝试切换时仅修改宿主机 Caddy 文件，容器内实际 bind mount 文件仍是旧 inode；Caddy 运行时 upstream 没有可靠确认已切换。
7. 在旧 upstream 尚未从 Caddy 运行时配置移除时停止旧应用容器，公网开始出现 502。
8. 发现 Caddy reload 参数错误：将 `http://127.0.0.1:2019` 传给自动补全协议的 `--address` 参数，形成 `http:///127.0.0.1:2019`，reload 实际失败。
9. 改用容器内合法管理地址 `127.0.0.1:2019` reload 候选配置；Caddy 管理 API 确认 upstream 为候选，API/CDN `/health` 恢复 200。
10. 延时观察、公网 `/health`、`/v1/models`、候选容器健康、迁移记录和依赖容器状态均通过，发布完成。

## 3. 影响

- `https://api.gavinteam.online/health` 和 `https://cdn.gavinteam.online/health` 在错误切换窗口内返回 502。
- 依赖服务没有被重启：PostgreSQL、Redis、Caddy、SubPilot 的容器 ID、启动时间和重启次数保持不变。
- 旧应用容器被停止后保留为回滚容器，没有删除旧镜像。
- 数据库仅执行了经过检查的、幂等的新增迁移；未执行删除、重建或清空操作。

## 4. 根因分析

### 直接原因

旧应用容器停止时，Caddy 实际仍指向旧容器。Docker DNS 无法解析已停止的旧 upstream，Caddy 返回 502。

### 流程原因

发布顺序没有严格执行平滑发布的必要状态机：

```text
错误：启动候选 -> 尝试 reload（未确认成功） -> 停止旧 -> 公网失败
正确：启动候选 -> 候选验证 -> validate/reload -> 运行时 upstream 验证
     -> 公网连续验证 -> 停止旧
```

### 技术原因

- Caddy reload 参数格式错误，且失败结果没有阻止后续停止旧容器。
- 单文件 bind mount 的宿主机文件更新没有改变运行中容器已打开的旧 inode；宿主机文件摘要与容器内文件摘要不一致。
- 发布只检查了配置文件和部分健康结果，没有把 Caddy 管理 API 的运行时 upstream 作为硬门禁。
- 缺少发布锁和统一的失败即回滚脚本，导致故障窗口扩大。

## 5. 已完成的恢复措施

- 候选应用保持运行，没有重启数据库、Redis、Caddy 或 SubPilot。
- 在 Caddy 容器内重新加载候选配置，并用管理 API 确认实际 upstream。
- 连续验证 API/CDN 公网 `/health` 返回 200。
- 验证未授权 `/v1/models` 返回 401，说明请求已到达应用路由而不是错误页。
- 验证候选容器为 `running healthy`，启动日志无 panic、fatal、数据库/Redis 连接失败或缺失表字段。
- 验证四个 `231_*` 数据库迁移记录及新增字段存在。
- 保留旧应用镜像和容器作为回滚点，保留发布备份目录。

## 6. 永久改进措施

### 发布顺序

1. 创建生产发布锁，确认没有其他发布会话。
2. 记录旧容器状态，旧容器全程保持运行。
3. 启动独立候选容器，不占用旧应用端口。
4. 完成候选健康、内部接口、关键 API 和日志检查。
5. 校验 Caddy 配置并执行 reload，检查命令返回值。
6. 从 Caddy 运行时管理 API 确认新 upstream。
7. 连续执行公网健康和关键 API 检查。
8. 观察窗口通过后才停止旧容器。
9. 任何失败先回到旧 upstream 并验证公网，再处理候选。

### 强制检查

- Caddy 管理地址使用 `127.0.0.1:2019` 这类合法地址，不把 `http://` 重复传给自动补协议参数。
- 必须同时检查：宿主机配置摘要、容器内配置摘要、Caddy 运行时 `/config/` upstream。
- reload 非零退出、管理 API 错误、运行时 upstream 未变化、公网 5xx，均视为切换失败。
- 发布报告必须记录旧容器 ID、候选容器 ID、Caddy 切换时间和实际验证结果。
- 不得以“候选健康”替代“公网已切换且验证通过”。

## 7. 防复发标准

以后任何生产发布必须满足以下条件才可报告成功：

- 旧容器在公网验证前没有停止。
- 新容器已经通过健康、内部接口、日志和关键 API 检查。
- Caddy reload 命令成功，且运行时 upstream 明确指向新容器。
- 公网 API/CDN 连续健康检查通过，没有 5xx 或旧 upstream DNS 错误。
- PostgreSQL、Redis、Caddy、SubPilot 未被无必要重启。
- 失败时已按顺序完成旧容器恢复、Caddy 回切和公网验证。

本复盘与 `docs/RELEASE_RUNBOOK_CN.md` 同步维护。Runbook 与本记录冲突时，以更严格的平滑发布门禁为准。

## 8. 复发记录：v0.1.186-lingqu.3 首次切换尝试短暂 502（2026-09-01）

本节记录在修复上述流程后，本次 `.3` 发布中仍发生的一次切换失误。最终版本已经完成生产切换，但本次发布不能被视为无事故发布。

### 事实与影响

- 发布目标：`0.1.186-lingqu.3`，提交 `3953d796b`，镜像 `sha256:ed6c5ae61b9b0d0f8b6aa5b9afc74c28fda4867eeebcc2bb8b50eda1c668b710`。
- 候选容器 `gavin2api-release-0.1.186-lingqu.3` 已经 `running/healthy`，旧容器在整个首次切换尝试期间仍保持运行，没有先停止旧应用。
- 第一次 reload 错误地使用了容器内 `/etc/caddy/Caddyfile`。该路径是旧的单文件 bind mount inode，实际内容仍指向不存在的 `gavin2api-canvas-text-key-0186-lingqu-1-default-pool`，因此 API/CDN 的本机入口短暂返回 `502`。
- 发现后立即使用已备份的旧配置在容器内通过 `127.0.0.1:2019` 回滚，API/CDN `/health` 恢复 `200`；随后使用容器内临时配置通过同一管理 API 加载候选 upstream。
- 候选 upstream 在运行时 `/config/` 中确认出现 2 处、旧 upstream 为 0 处；API/CDN 外部域名连续 3 次为 `200`，未授权 `/v1/models` 为 `401`，之后才优雅停止旧容器。

### 直接原因

此前已经记录了单文件 bind mount 的 inode 漂移风险，但本次操作仍执行了从 `/etc/caddy/Caddyfile` reload 的命令，没有在 reload 前阻断并强制使用已校验的容器内临时配置。宿主机文件已修改不代表容器内 bind mount 文件已更新。

### 新增永久措施

- 发布前必须比较宿主机 Caddyfile、容器内 `/etc/caddy/Caddyfile` 的 inode、大小和摘要；不一致时禁止从 `/etc/caddy/Caddyfile` reload。
- Caddy 使用单文件 bind mount 时，候选配置必须先复制到 Caddy 容器内临时路径，执行 `caddy validate`，再使用 `127.0.0.1:2019` reload；切换成功必须以管理 API `/config/` 的 upstream 为准。
- 第一次 reload 出现任何公网 `5xx` 时，发布进入回滚状态：先恢复旧 upstream 并验证公网，再决定是否重新开始一个新的发布窗口；不得把“回滚后继续”当作无事故成功发布。
- 本次服务器报告保存在 `/opt/gavin2api/deployment-backups/20260901T065522Z-0.1.186-lingqu.3-3953d796/release-report.txt`，其中保留了镜像、候选、旧容器和验证结果。
