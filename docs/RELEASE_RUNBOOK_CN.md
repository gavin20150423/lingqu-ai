# Gavin2API 正式发布 Runbook

本文档规定 Gavin2API 的正式发布路径，适用于 `gavin20150423/lingqu-ai` 仓库和生产环境。发布人员与自动化代理都必须遵守。

## 1. 唯一正式发布路径

正式发布必须依次经过以下链路：

1. 本地完成代码修改、测试和提交。
2. 将目标提交推送到 `gavin20150423/lingqu-ai` 的默认分支。
3. 创建并推送版本标签，或由维护者手动触发 GitHub Actions 的 `.github/workflows/release.yml`。
4. 等待 GitHub Actions 的测试、前端构建和 GoReleaser 任务全部成功。
5. 确认 GitHub Release 已创建，且 GHCR 中存在对应版本镜像。
6. 生产服务器只拉取 GitHub Actions 生成的 GHCR 镜像。
7. 备份 Compose 配置，只重建 `gavin2api` 服务，并执行健康检查。
8. 健康检查失败时立即恢复 Compose 备份并回滚到上一 GHCR 镜像。

正式生产镜像必须采用以下命名：

```text
ghcr.io/gavin20150423/sub2api:<version>
```

生产 Compose 中不得把 `local/gavin2api:*`、人工 `docker commit` 生成的镜像，或单独上传的本地二进制当作正式发布物。

## 2. 发布前检查

在创建版本标签前完成以下检查：

```bash
git status --short --branch
git diff --check
git remote -v
git log -1 --oneline
```

必须确认：

- 工作区不包含与本次发布无关的修改。
- 可写远端是 `gavin20150423/lingqu-ai`，不得向官方上游 `Wei-Shaw/sub2api` 推送。
- 目标提交已经完成与风险相匹配的测试。
- 目标版本标签在远端尚不存在。
- 当前环境具备向可写仓库推送提交和标签的权限。

缺少 GitHub 推送权限不是切换发布方式的理由。遇到该情况必须停止发布，向维护者报告需要配置 GitHub SSH Key、PAT 或由有权限的人员推送。

## 3. 触发 GitHub Actions

推荐使用带说明的版本标签触发发布：

```bash
git push lingqu main
git tag -a v<version> -m "Release v<version>"
git push lingqu v<version>
```

也可以由维护者在 GitHub Actions 页面手动运行 `Release` 工作流，并传入已有的版本标签。除非明确需要完整的多架构产物，否则可以按仓库配置选择 simple release；生产 amd64 服务器仍必须使用 Actions 生成的 GHCR 版本镜像。

不得仅在本地创建标签后就宣称发布完成。必须确认远端标签和 Actions 运行均存在。

## 4. Actions 成功标准

`.github/workflows/release.yml` 至少必须满足：

- `update-version` 成功。
- `test-backend` 成功。
- `build-frontend` 成功。
- `release` 成功。
- GitHub Release 中的版本与标签一致。
- GHCR 可以拉取 `ghcr.io/gavin20150423/sub2api:<version>`。

任一任务失败、取消或超时，都视为发布失败。不得跳过失败任务直接部署本地产物。

## 5. 生产部署

生产目录当前为 `/opt/gavin2api`，Compose 文件为 `/opt/gavin2api/compose.yml`。部署前先确认实际环境没有变化，不得盲目套用路径。

部署步骤：

1. 记录当前 `gavin2api` 镜像、容器状态和健康状态。
2. 拉取 Actions 生成的目标 GHCR 镜像。
3. 将 `compose.yml` 复制到带 UTC 时间戳的 `deploy-backups` 子目录。
4. 只替换 `gavin2api` 服务的 `image:` 值。
5. 执行 `docker compose --profile app config` 校验配置。
6. 使用 `--no-deps --force-recreate gavin2api` 只重建应用容器。
7. 等待容器变为 `healthy`，检查内部 `/health` 和公网 `/health`。
8. 确认 PostgreSQL、Redis、Caddy 和 SubPilot 的启动时间没有变化。

发布操作不得覆盖或修改 `.env`、数据库、Redis 数据、Docker volumes、网络或凭据。

## 6. 回滚条件

出现以下任一情况必须自动回滚：

- 新容器无法创建或启动。
- 容器进入 `unhealthy`、`exited` 或 `dead`。
- 等待窗口内没有变为 `healthy`。
- 内部 `/health` 非 200。
- 公网 `/health` 非 200。
- 启动日志出现 panic、fatal、数据库连接失败或 Redis 连接失败。

回滚时恢复部署前的 Compose 备份，并重新创建上一版本的 `gavin2api` 容器。回滚成功后也必须将本次发布标记为失败，不得报告为已发布。

## 7. 明确禁止的降级方案

除非维护者在当次发布请求中明确批准紧急离线部署，否则禁止：

- 在本地交叉编译生产二进制后用 SCP 上传服务器。
- 以旧生产镜像为基础，通过 `docker create`、`docker cp`、`docker commit` 生成新生产镜像。
- 因 Docker Hub、GitHub、GHCR 或认证不可用而自行改走本地镜像。
- 使用 `local/gavin2api:*` 作为正式生产版本。
- Actions 未成功时把版本标签、GitHub Release 或生产部署描述为已完成。

如果维护者明确批准紧急离线部署，该操作也只能作为临时恢复措施，必须记录构建来源、哈希、配置差异、回滚点和后续替换计划，且不得冒充 Actions 正式发布。

## 8. 2026-08-11 发布偏差记录

### 事件

发布 `0.1.172-lingqu.8` 时，本地环境无法向 GitHub 可写仓库推送，服务器也没有 GHCR/GitHub 发布凭据。执行者错误地把认证失败视为可以切换发布路径，采用了：

1. 本地交叉编译 Linux amd64 二进制。
2. 将二进制上传到生产服务器。
3. 以 `0.1.172-lingqu.7` 的 GHCR 镜像为基础人工重打包 `.8` 本地镜像。
4. 修改生产 Compose，切换到 `local/gavin2api:0.1.172-lingqu.8`。

第一次人工重打包还错误继承了临时容器的 `/bin/sh` 入口，导致健康检查失败。自动回滚恢复了 `.7`。修正入口后再次切换成功。

### 当前偏差

截至该事件记录时：

- 生产运行的是 `local/gavin2api:0.1.172-lingqu.8`。
- 对应代码提交为 `4ccb16fc1`。
- `v0.1.172-lingqu.8` 只在本地创建，尚未确认已推送到 GitHub。
- GitHub Actions 尚未为 `.8` 生成正式 GHCR 镜像。
- 因此当前生产状态健康，但不属于本 Runbook 定义的正式发布。

### 必须完成的纠正动作

1. 恢复对 `gavin20150423/lingqu-ai` 的推送权限。
2. 推送目标提交和 `v0.1.172-lingqu.8` 标签。
3. 等待 `Release` Actions 全部成功。
4. 确认 `ghcr.io/gavin20150423/sub2api:0.1.172-lingqu.8` 可拉取。
5. 备份 Compose，将生产镜像从本地 `.8` 替换为 GHCR `.8`。
6. 重新完成容器、内部健康、公网健康和关联容器未重启检查。
7. 纠正完成后再把 `.8` 标记为正式发布。

## 9. 发布完成报告

最终报告至少包含：

- 远端提交和版本标签。
- GitHub Actions 运行结果。
- GitHub Release 和 GHCR 镜像名称。
- 生产容器镜像、健康状态和启动时间。
- 内部及公网健康检查结果。
- Compose 备份位置和回滚版本。
- 其他关联容器是否保持运行。

不得把本地提交、本地标签或健康的本地镜像等同于 GitHub Actions 正式发布。
