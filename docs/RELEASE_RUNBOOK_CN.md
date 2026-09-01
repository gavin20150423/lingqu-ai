# Gavin2API 正式发布 Runbook

本文档规定 Gavin2API 的正式发布路径，适用于 `gavin20150423/lingqu-ai` 仓库和生产环境。发布人员与自动化代理都必须遵守。

## 1. 唯一正式发布路径

正式发布必须依次经过以下链路：

1. 所有功能修改都从本地 `main` 分支开始；完成后先通过测试并提交回本地 `main`，不得在未合并的临时分支或脏工作树上发布。
2. 确认本地 `main` 已包含可访问远端默认分支的最新提交；远端认证不可用时只能报告并保留当前提交，不能用不确定的旧代码覆盖发布。
3. 按目标提交生成 Linux amd64 后端和嵌入式前端，构建一个带版本与提交短哈希的本地 Docker 镜像，并记录镜像摘要和构建提交。
4. 备份生产 Compose、Caddy 配置和当前容器状态；通过 `docker save`/`docker load` 将镜像上传到服务器，不上传裸二进制替换正在运行的应用。
5. 使用相同生产环境变量和网络启动新的 `gavin2api` 容器，使用独立容器名和未占用端口进行预热；只允许应用容器连接现有 PostgreSQL、Redis 和其他依赖，禁止重启这些依赖。
6. 新容器通过容器健康检查、内部 `/health`、关键 API 冒烟检查和日志检查后，才让 Caddy 将上游切换到新容器，并使用 reload 平滑加载配置。
7. 切换前必须确认 Caddy 的实际运行配置已经解析到新 upstream；切换时只能执行 Caddy 的平滑 reload，禁止先停止旧应用容器。切换后必须再次执行公网健康检查和关键 API 冒烟检查，并确认请求确实进入新容器。
8. 只有在新容器已承载流量、连续观察窗口通过、且 PostgreSQL、Redis、Caddy、SubPilot 的容器 ID 和启动时间均未变化后，才允许停止旧应用容器。旧容器在此之前必须保持运行，作为即时回滚点。
9. 任一步骤失败都保持旧容器承载流量，或先启动旧容器并确认健康，再将 Caddy reload 回旧容器；确认回切公网健康后才允许停止/清理新容器。不得以数据库重启或整栈重启作为排障手段。

### 平滑发布硬门禁

以下规则是不可跳过的发布门禁，任何一条无法证明时都必须暂停发布：

- **先验证、后切流、最后停旧**：旧线上容器在切流前、切流中和公网验证完成前必须保持 `running`，不得先 `docker stop`、`docker rm` 或通过 Compose 重建旧服务。
- **候选必须隔离**：候选容器使用独立名称、独立 Compose project/service 或显式 `docker run`，不得绑定旧应用宿主端口，不得复用固定 `container_name`。
- **Caddy 配置必须双重确认**：先对即将加载的文件执行 `caddy validate`，再从 Caddy 管理 API `/config/` 或等价的运行时状态确认 upstream 已从旧容器变为候选容器。只检查宿主机文件内容不算切换成功。
- **reload 必须检查返回值**：Caddy reload 的管理地址必须使用合法格式（例如容器内 `127.0.0.1:2019`，不能写成 `http://127.0.0.1:2019` 传给会自动补协议的参数）。命令非零退出、管理 API 不可达或运行时 upstream 未变化时，立即停止后续动作。
- **公网成功才算切流**：切流后至少连续三次检查 API/CDN `/health`，并执行一个未授权 `/v1/models` 检查；任一返回 5xx、DNS upstream 解析失败或管理 API 仍显示旧 upstream，都必须先回切旧容器。
- **回滚顺序固定**：新容器异常时，先保证旧容器 `running/healthy`，再 reload Caddy 回旧 upstream，确认公网恢复 200 后才处理新容器。旧容器已经被误停时，禁止继续等待或清理，必须立即启动旧容器并完成同样的公网验证。
- **禁止并发发布**：发布开始前在生产目录创建带持有者和时间的发布锁；发现其他发布、Caddy reload 或 Compose 操作正在进行时必须暂停，不得两个会话同时改 Caddy 或应用容器。
- **配置文件挂载必须核对**：如果 Caddy 使用单文件 bind mount，必须比较宿主机文件和容器内 `/etc/caddy/Caddyfile` 的摘要，并确认容器重启后仍能读取新文件。文件 inode 未同步时，只能使用已确认的运行时 reload；不得把“宿主机文件已修改”当作当前运行配置已切换。

本地发布镜像必须采用以下命名：

```text
local/gavin2api:<version>-<short-commit>
```

正式版本号必须遵循 `<上游 Sub2API 版本>-lingqu.<本地小版本序号>`，例如上游为
`0.1.186` 时，本地首个定制发布使用 `0.1.186-lingqu.1`，同一上游版本的后续
定制发布依次递增 `.2`、`.3`；上游版本变化时重新从 `.1` 开始。`backend/cmd/server/VERSION`、
Release Log、镜像标签和发布报告必须使用同一个完整版本号。

镜像必须由当前 `main` 的 Dockerfile 构建，禁止以旧容器为基础执行 `docker create`、`docker cp` 或 `docker commit`。生产 Compose 只允许临时指向已记录摘要的本地构建镜像；发布完成后保留上一版本镜像作为回滚点。

生产 Compose 如果为应用声明了固定的 `container_name: gavin2api`，不得通过复用
`gavin2api` 服务名的临时 override 来创建蓝绿候选；即使 override 指定了新的
`container_name`，Compose 仍可能先重建或删除原服务容器。蓝绿候选必须使用独立的
Compose 服务/项目标识，或直接使用显式参数执行 `docker run`，并明确不绑定旧应用的
宿主端口。候选启动后必须立即核对旧线上容器 ID、状态和端口没有变化；若发生变化，
停止后续切流，先依据备份恢复原应用容器。

## 2. 发布前检查

在创建版本标签或开始本地构建前完成以下检查：

```bash
git status --short --branch
git diff --check
git remote -v
git log -1 --oneline
```

必须确认：

- 发布提交位于本地 `main`，且工作区不存在未审查的本次发布改动；其他会话产生的改动不得覆盖、回退或混入本次提交。
- 可写远端如需同步，必须是 `gavin20150423/lingqu-ai`，不得向官方上游 `Wei-Shaw/sub2api` 推送。
- 目标提交已经完成与风险相匹配的测试。
- 版本号、提交短哈希、镜像摘要和回滚镜像均已记录。
- `docs/RELEASE_LOG.md` 已新增当前版本条目，并分别写明“修复了什么”、“增加了什么”和“当前已有功能”；没有新增功能时必须明确写“无新增”，不得省略。

GitHub 非交互认证失败只阻止远端同步，不阻止已经完成本地测试、镜像摘要校验和蓝绿检查的本地镜像发布。本机已经配置 SSH key 时，仍应确认当前进程实际使用了正确的 `IdentityFile`，并检查 SSH agent、密钥口令会话、`GIT_SSH_COMMAND` 和 Git remote 配置。不得仅凭一次 `Permission denied (publickey)` 就用不确定的旧代码或未审查产物发布；远端同步应在认证恢复后补做并记录。

## 3. 可选的远端同步

如果 GitHub SSH/HTTPS 认证可用，可以同步主分支和版本标签，供代码审计和备份：

```bash
git push lingqu main
git tag -a v<version> -m "Release v<version>"
git push lingqu v<version>
```

远端同步不是本地发布的前置条件；认证失败时不得改用未知分支或不受控的构建产物。若触发 GitHub Actions，仍需单独记录 Actions 结果，但不得因此跳过本地镜像摘要、蓝绿预热和 Caddy 回切检查。

## 4. 本地构建与上传

发布前在仓库根目录执行并记录以下信息（版本号示例为 `0.1.186-lingqu.1`）：

```bash
git switch main
git pull --ff-only <writable-remote> main
git status --short --branch
git diff --check
make test-backend
pnpm --dir frontend run lint:check
pnpm --dir frontend run typecheck
pnpm --dir frontend run build
git rev-parse --short HEAD
docker build --platform linux/amd64 --build-arg VERSION=<version> \
  -t local/gavin2api:<version>-<short-commit> .
docker image inspect local/gavin2api:<version>-<short-commit> \
  --format '{{index .RepoDigests 0}} {{.Id}}'
docker save local/gavin2api:<version>-<short-commit> | gzip > gavin2api-<version>-<short-commit>.tar.gz
```

上传后在服务器执行 `docker load`，并核对镜像 ID/摘要与本地记录一致；上传失败或摘要不一致时不得启动新容器。

## 5. 生产部署

生产目录当前为 `/opt/gavin2api`，Compose 文件为 `/opt/gavin2api/compose.yml`。部署前先确认实际环境没有变化，不得盲目套用路径。

部署步骤：

1. 创建生产发布锁，并记录当前时间、操作者、目标提交和目标镜像；发现锁已存在时停止。
2. 记录当前旧 `gavin2api` 容器的名称、ID、镜像、状态、健康状态、端口和启动时间；旧容器不允许在后续候选验证完成前停止。
3. 将 `compose.yml`、Caddy 配置、实际运行配置和当前容器信息复制到带 UTC 时间戳的 `deploy-backups` 子目录。
4. `docker load` 新镜像，并用独立容器名、独立端口启动新 `gavin2api`；不得执行 `docker compose down`，不得重启 PostgreSQL、Redis、Caddy 或 SubPilot。启动后立即重新核对旧容器 ID、状态和端口没有变化。
5. 执行候选容器健康检查、内部 `/health`、关键 API 冒烟请求、候选容器日志错误检查，并确认候选能访问现有数据库/Redis。
6. 在不停止旧容器的前提下，生成只包含新 upstream 的 Caddy 配置，执行 `caddy validate`，然后 reload；检查 reload 返回值，并从 Caddy 运行时管理 API 确认 upstream 已经变成候选容器。
7. 通过 API 和 CDN 公网域名连续至少三次执行 `/health`，执行关键 API 冒烟检查和 `/v1/models` 未授权检查；同时查看 Caddy 日志，确认没有 5xx、DNS upstream 解析失败或旧 upstream 请求。
8. 记录 PostgreSQL、Redis、Caddy、SubPilot 的容器 ID、启动时间和重启次数，确认均未变化；确认观察窗口通过后，才停止旧 `gavin2api` 容器。旧容器镜像至少保留一个发布周期。
9. 释放发布锁，并将版本、镜像摘要、切流时间、验证结果、旧容器和回滚点写入发布报告。

任何步骤失败时，必须按“先恢复旧容器、再 reload 旧 upstream、再验证公网、最后处理候选”的顺序回滚；未完成回滚验证不得释放发布锁，也不得报告发布成功。

发布操作不得覆盖或修改 `.env`、数据库、Redis 数据、Docker volumes、网络或凭据。

## 6. 回滚条件

出现以下任一情况必须自动回滚：

- 新容器无法创建或启动。
- 容器进入 `unhealthy`、`exited` 或 `dead`。
- 等待窗口内没有变为 `healthy`。
- 内部 `/health` 非 200。
- 公网 `/health` 非 200。
- 启动日志出现 panic、fatal、数据库连接失败或 Redis 连接失败。

回滚时先将 Caddy reload 回旧容器，确认旧容器健康后再清理新容器；只有在旧容器本身已损坏时，才允许依据备份重新创建 `gavin2api`。回滚全过程不得重启 PostgreSQL、Redis、Caddy 或 SubPilot。回滚成功后也必须将本次发布标记为失败，不得报告为已发布。

## 7. 数据库与依赖保护规则

以下规则每次发布都强制执行：

- 不得删除、重建、初始化或清空 PostgreSQL/Redis 容器、卷、网络和数据目录。
- 没有明确必要不得重启 PostgreSQL、Redis、Caddy 或 SubPilot；应用发布只允许启动/停止 `gavin2api` 新旧容器及执行 Caddy reload。
- 数据库迁移必须是新增、可回滚、经过测试的迁移；若迁移风险无法确认，先停止发布，不得通过重启数据库规避问题。
- 新容器必须使用现有 `.env` 和现有外部依赖，不得在发布过程中改密码、JWT、存储桶或数据库连接信息。
- 新容器必须沿用现有应用的数据库连接池配置；不得为蓝绿发布临时设置或缩小 `DATABASE_MAX_OPEN_CONNS`、`DATABASE_MAX_IDLE_CONNS`。如需调整连接池，必须作为独立、评审过的配置变更执行。
- 新容器健康检查失败时优先 Caddy 回切旧容器；回滚不得触碰数据库和 Redis。
- 每次已完成的代码修改必须提交到 `main`；下一次功能修改必须从最新 `main` 检出新工作分支，完成后再合并回 `main`。

本地构建上传是本项目批准的正式发布方式，但必须同时记录构建提交、镜像 ID/摘要、服务器上传时间、Caddy 切换时间、健康检查结果和回滚点；不得把未验证的本地镜像宣称为已发布。

## 8. 2026-08-11 发布偏差记录

### 事件

发布 `0.1.172-lingqu.8` 时，本机实际已经存在可访问 GitHub 的 SSH key，SSH 配置也为 `github.com` 指定了该 key。当前自动化进程的非交互认证没有成功，且没有接入可用的 SSH agent 会话。执行者没有继续排查和接入本机已有的正常认证环境，反而错误地把当前进程的认证失败解释成缺少 GitHub 发布能力，并据此切换了发布路径。服务器没有 GHCR/GitHub 推送凭据与此无关：正式发布本来就应由本机推送标签并触发 GitHub Actions，服务器只负责拉取 Actions 生成的 GHCR 镜像。错误采用的步骤为：

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

1. 使用本机已有的 GitHub SSH key，在具备正常密钥解密/agent 会话的终端中验证 `github.com` 认证；如果自动化进程无法继承该会话，必须停止并明确报告，不能改走其他发布路径。
2. 使用本机正常 SSH 环境推送目标提交和 `v0.1.172-lingqu.8` 标签到 `gavin20150423/lingqu-ai`。
3. 等待 `Release` Actions 全部成功。
4. 确认 `ghcr.io/gavin20150423/sub2api:0.1.172-lingqu.8` 可拉取。
5. 备份 Compose，将生产镜像从本地 `.8` 替换为 GHCR `.8`。
6. 重新完成容器、内部健康、公网健康和关联容器未重启检查。
7. 纠正完成后再把 `.8` 标记为正式发布。

## 9. 发布完成报告

最终报告至少包含：

- `main` 提交、版本号和构建提交短哈希。
- 本地镜像名称、镜像 ID/摘要和服务器上传时间。
- 生产新旧应用容器、健康状态、Caddy 切换时间和回滚镜像。
- 内部及公网健康检查、关键 API 冒烟检查结果。
- Compose/Caddy 备份位置。
- PostgreSQL、Redis、Caddy、SubPilot 是否保持原容器 ID 和启动时间。
- 如果同步远端或触发 GitHub Actions，再附上远端提交、标签、Actions、Release 和镜像结果。
- 本次发布日志中记录的修复、新增和当前已有功能。

不得把本地提交、本地标签或健康的本地镜像等同于 GitHub Actions 发布；本项目的正式生产发布以本 Runbook 规定的本地镜像蓝绿切换和健康检查完成为准。

## 10. 事故记录：v0.1.186-lingqu.2 发布期间短暂 502（2026-09-01）

本节记录本次发布事故，作为后续发布的强制反例和检查依据。详细复盘见
`docs/RELEASE_INCIDENT_20260901_CN.md`。

### 事故概况

- 发布目标：`0.1.186-lingqu.2`，提交 `130ce1222`。
- 事故表现：旧应用容器停止后，Caddy 实际仍指向旧 upstream；旧容器已停止，公网 API/CDN 一度返回 HTTP 502。
- 影响范围：`api.gavinteam.online`、`cdn.gavinteam.online` 的请求在切换窗口内失败；数据库、Redis、SubPilot 未重启，数据未丢失。
- 恢复方式：保持候选应用运行，在 Caddy 容器内用正确的管理地址 reload 候选配置，确认运行时 upstream 和公网健康恢复后完成发布。

### 根因与永久措施

1. 违反“先切流验证、后停旧”的平滑发布顺序，在 Caddy 运行时配置未确认前停止了旧容器。
2. Caddy reload 参数把带协议的地址传给会自动补协议的参数，实际请求变成了 `http:///127.0.0.1:2019`，reload 失败未被当作硬门禁拦截。
3. 只验证了宿主机 Caddy 文件，没有在切流前确认 Caddy 管理 API 的实际 upstream；同时单文件 bind mount 的容器内 inode 与宿主机文件不同步，增加了误判风险。
4. 发布过程中缺少发布锁和统一的失败即回滚脚本，导致故障窗口扩大。

### 后续发布禁止事项

- 禁止在公网切换成功并连续验证通过前停止旧应用容器。
- 禁止忽略 Caddy reload 的非零退出、管理 API 错误或运行时 upstream 未变化。
- 禁止只看宿主机配置文件就认为 Caddy 已完成切流。
- 禁止多个会话同时执行发布、Caddy reload 或应用容器启停。
