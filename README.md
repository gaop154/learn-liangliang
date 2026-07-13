# learn-liangliangleang 技术内容归档 📚

原始网站： [learn.lianglianglee.com](https://learn.lianglianglee.com)

推荐访问： [learn-liangliang.wenxuanhe.top](https://learn-liangliang.wenxuanhe.top) (我们对网站进行了性能优化，并且增加了一些趣味功能)

---

## 1. 项目说明

本项目对原始网站内容进行了系统性备份和整理。learn.lianglianglee.com 是一个极具价值的中文技术学习网站，聚合了大量高质量的技术专栏和精选文章，涵盖后端开发、分布式系统、DevOps、架构设计、面试指南等多个领域。由于服务器到期、费用等问题，原站点有随时关闭的风险，宝贵的技术资料可能会丢失。备用站点将长期维护，确保资料持久保存，不会因原站点关闭而丢失。

## 2. 网站内容介绍 📝

## 3. 项目目录说明 🗃️

- **content/**：文章相关归档内容物理根目录。
  - **content/专栏/**：技术专栏归档，对外路径为 `/专栏/`。
  - **content/其他/**：非课程分类归档，对外路径为 `/其他/`。
    - **content/其他/恋爱必修课/**、**content/其他/文章/**、**content/其他/极客时间/**：支持单篇阅读进度。
    - **content/其他/PDF/**：PDF 资料归档，不记录阅读进度。
  - 各分类和课程保留各自的 `assets/` 子目录；已清理的捐赠页不再提供入口。
- **static/**：前端样式、脚本与公共静态资源
- **img/**：页面截图与图片资源
- **backend/**：Go API 服务，提供登录、会话和阅读进度同步接口
- **deploy/caddy/**：生产入口 Caddy 配置，负责 `/api/*` 与静态站点分流
- **deploy/nginx/**：静态站点容器内的 Nginx 配置，负责旧公开 URL 到 `content/` 的内部映射
- **utils/**：离线内容抓取、修补和迁移脚本；这些脚本可使用 Python 依赖，但不属于生产 Web 运行时
- **index.html**：网站首页
- 公开访问 URL 使用 `/专栏/...` 与 `/其他/...`，不会暴露为 `/content/...`；旧 `/文章/...`、`/极客时间/...`、`/恋爱必修课/...`、`/PDF/...` 路径不再保留静态映射或重定向。
- 旧 Flask/Gunicorn 运行时已移除，生产部署不再使用 Python 提供 Web 服务。

## 4. Docker 部署 🚀

生产部署支持两种方式：默认使用 Docker Compose 编排 Caddy + Nginx 静态站点 + Go API + PostgreSQL；如果服务器已经运行 PostgreSQL 和 Nginx，则可只将 Go API 运行在 Docker 中，由宿主机 Nginx 负责 HTTPS、静态站点、内容鉴权和 API 反向代理。Python 只用于本地或离线执行 `utils/*.py` 内容维护脚本，不参与生产 Web 请求链路。

- `gateway`：Caddy 入口，对外暴露 80/443，负责将 `/api/*` 转发到 Go API，其余请求转发到静态站点。
- `web`：Nginx 静态站点，托管现有 HTML、PDF、图片和前端脚本；把公开 URL 内部映射到 `content/`，并在读取受保护内容前通过 `auth_request` 校验会话。
- `api`：Go 后端 API，提供登录、会话、供静态网关调用的内部会话校验，以及阅读进度同步。
- `db`：PostgreSQL，使用 `postgres_data` volume 持久化数据。
- `content-sync`：按需执行的一次性内容索引服务，仅只读挂载 `content/`，同步完成后自动退出。

### 4.1 内容访问控制

以下页面和内容目录必须先登录：

```text
/
/index.html
/专栏/**
/其他/**
/reading-history.html
```

其中 `/其他/PDF/**` 和课程、文章目录中的图片等资源也属于受保护内容。未登录请求会在静态文件返回前重定向到：

```text
/login.html?next=<原路径>
```

登录成功后会安全返回同源原路径。受保护响应使用 `Cache-Control: private, no-store`，避免浏览器或中间缓存保存私有内容。

以下路径保持公开，以保证登录页和公共资源可用：`/login.html`、`/api/**`、`/static/**`、`/img/**`、`/live-2d/**`。`/content/**` 是物理内容目录，始终禁止直接访问；不要单独暴露 `web:80` 或 `api:8080` 端口，否则会绕过网关边界。

### 4.2 本地启动

本地推荐也使用 Docker Compose 启动完整环境，避免手工安装 PostgreSQL、Nginx 和 Caddy。

1. 复制环境变量文件：

```bash
cp .env.example .env
```

Windows PowerShell 可使用：

```powershell
Copy-Item .env.example .env
```

2. 编辑 `.env`，本地 HTTP 调试可以使用下面这组配置思路：

```env
SITE_DOMAIN=:80
ACME_EMAIL=local@example.com

POSTGRES_DB=learn_liangliang
POSTGRES_USER=learn
POSTGRES_PASSWORD=local_dev_password_123
DATABASE_URL=postgres://learn:local_dev_password_123@db:5432/learn_liangliang?sslmode=disable

APP_ADDR=:8080
APP_COOKIE_NAME=learn_session
APP_COOKIE_SECURE=false
APP_SESSION_TTL_HOURS=720

ADMIN_USERNAME=admin
ADMIN_PASSWORD=local_admin_password_123
ADMIN_DISPLAY_NAME=管理员
```

注意：

- 本地使用 `SITE_DOMAIN=:80`，浏览器访问 `http://localhost/`。
- 本地 HTTP 调试必须设置 `APP_COOKIE_SECURE=false`，否则浏览器不会在 HTTP 下发送登录 Cookie。
- `POSTGRES_PASSWORD` 和 `DATABASE_URL` 中的密码必须保持一致。
- `ADMIN_PASSWORD` 不能使用示例占位符，且至少 12 位。

3. 启动：

```bash
docker compose up -d --build
```

4. 查看状态和日志：

```bash
docker compose ps
docker compose logs -f api
```

5. 浏览器访问：

```text
http://localhost/
```

登录入口：

```text
http://localhost/login.html
```

阅读记录入口：

```text
http://localhost/reading-history.html
```

6. 停止本地环境：

```bash
docker compose down
```

如果想同时删除本地 PostgreSQL 数据卷，使用：

```bash
docker compose down -v
```

### 4.3 本地非 Docker 启动

如果不想使用 Docker，也可以在本机直接启动 PostgreSQL、Go API 和 Caddy。这个方式适合开发调试，但需要自己安装和管理本机服务。

需要先安装：

- Go 1.23+
- PostgreSQL 16+
- Caddy 2+

Windows 推荐通过 PowerShell 安装 Caddy：

```powershell
winget install CaddyServer.Caddy
```

安装完成后新开一个 PowerShell，并确认：

```powershell
caddy version
```

如果系统没有 `winget`，可以从 [Caddy 下载页](https://caddyserver.com/download) 下载 Windows ZIP 包，将 `caddy.exe` 所在目录加入 `Path`，或在该目录直接执行 `caddy.exe`。

本地非 Docker 模式的端口分工：

| 服务 | 端口 | 职责 |
| --- | --- | --- |
| PostgreSQL | `5432` 或本机实际端口 | 数据库 |
| Go API | `8080` | 登录、会话、阅读进度 API |
| Caddy | `8081` | 浏览器访问入口、静态文件服务、`/api/*` 反向代理 |

浏览器应访问 Caddy 的 `http://localhost:8081/`，不要直接访问 Go API 的 `http://localhost:8080/`；Go API 只提供 `/api/*` 接口，访问其首页会返回 404。

1. 创建本地数据库和账号：

```bash
psql -U postgres
```

在 `psql` 中执行：

```sql
CREATE USER learn WITH PASSWORD 'local_dev_password_123';
CREATE DATABASE learn_liangliang OWNER learn;
\q
```

2. 准备 Go API 本地配置并启动后端。

Docker / 生产部署继续使用项目根目录 `.env` 注入环境变量；本地非 Docker 调试推荐使用 `backend/config.yaml`，避免每次手动导出多项环境变量。环境变量优先级始终最高，可用于临时覆盖 `config.yaml` 中的配置。

Git Bash / Linux / macOS：

```bash
cd backend
cp config.example.yaml config.yaml
# 按本机 PostgreSQL 账号、密码和端口编辑 config.yaml
go run ./cmd/api
```

Windows PowerShell：

```powershell
cd backend
Copy-Item config.example.yaml config.yaml
# 按本机 PostgreSQL 账号、密码和端口编辑 config.yaml
go run ./cmd/api
```

如需指定其他配置文件路径，可以设置可选环境变量 `APP_CONFIG_FILE`：

```bash
APP_CONFIG_FILE=/path/to/config.yaml go run ./cmd/api
```

也可以继续完全使用环境变量启动，或只覆盖其中少数字段。例如本地 HTTP 调试可临时覆盖 Cookie Secure：

```bash
APP_COOKIE_SECURE=false go run ./cmd/api
```

后端启动后会自动执行数据库迁移，并创建或更新管理员账号。

3. 新开一个终端，在项目根目录创建本地 Caddy 配置文件，例如 `Caddyfile.local`：

```caddyfile
:8081 {
    encode gzip
    root * .

    @blocked path /.git* /.claude* /.trellis* /backend* /deploy* /utils* /content* /docker-compose.yml /requirements.txt /README.md /.env*
    @protected path / /index.html /专栏 /专栏/* /其他 /其他/* /reading-history.html

    route {
        handle /api/* {
            reverse_proxy 127.0.0.1:8080
        }

        respond @blocked 403

        handle @protected {
            route {
                intercept {
                    @unauthorized status 401
                    handle_response @unauthorized {
                        redir * /login.html?next={http.request.uri.path} 302
                    }
                }

                forward_auth 127.0.0.1:8080 {
                    uri /internal/authenticate
                }

                try_files {path} {path}/ /content{path} /content{path}/index.html /content{path}/
                file_server
            }
        }

        try_files {path} {path}/ /content{path} /content{path}/index.html /content{path}/
        file_server
    }
}
```

4. 启动本地静态站点网关：

```bash
caddy run --config Caddyfile.local
```

Windows PowerShell 同样执行：

```powershell
caddy run --config Caddyfile.local
```

Caddy 会持续运行，请保持此终端窗口打开。

5. 浏览器访问：

```text
http://localhost:8081/
```

登录入口：

```text
http://localhost:8081/login.html
```

阅读记录入口：

```text
http://localhost:8081/reading-history.html
```

注意：

- 非 Docker 模式下 Go API 不会自动读取项目根目录 `.env` 文件；推荐使用 `backend/config.yaml`，或通过环境变量覆盖配置。
- 本地 HTTP 调试必须设置 `app.cookieSecure: false`，或用环境变量 `APP_COOKIE_SECURE=false` 覆盖。
- `backend/config.yaml` 只用于本地私有配置，已被 `.gitignore` 忽略，不要提交真实密码。
- `Caddyfile.local` 只是本地临时配置，不建议提交到仓库；但它必须保留上方的 `forward_auth` 和 `intercept` 配置，否则本地预览会绕过生产环境的内容登录保护。
- 未登录访问 `/`、`/专栏/**`、`/其他/**` 或 `/reading-history.html` 时，应返回 `302` 至 `/login.html?next=<原路径>`；`/login.html`、`/static/**`、`/img/**` 保持公开，`/content/**` 返回 `403`。
- 变更本地 Caddyfile 后，先执行 `caddy adapt --config Caddyfile.local --validate`，再重新启动或 reload Caddy。
- 如果本机 PostgreSQL 端口、账号或密码不同，请同步修改 `backend/config.yaml` 中的 `databaseUrl`。
- 停止 Go API 或 Caddy 时，在各自终端按 `Ctrl + C`。

### 4.4 CentOS 安装 Docker

```bash
sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable docker
sudo systemctl start docker
```

放行端口：

```bash
sudo firewall-cmd --add-service=http --permanent
sudo firewall-cmd --add-service=https --permanent
sudo firewall-cmd --reload
```

云服务器安全组也需要放行 80/443。生产环境建议使用域名访问，Caddy 会自动申请 HTTPS 证书。

### 4.5 配置环境变量

```bash
cp .env.example .env
```

然后编辑 `.env`，至少替换：

- `SITE_DOMAIN`
- `ACME_EMAIL`
- `POSTGRES_PASSWORD`
- `DATABASE_URL` 中的密码
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`

生产环境保持 `APP_COOKIE_SECURE=true`；只有本地 HTTP 调试时才临时改为 `false`。

### 4.6 生产启动

```bash
docker compose up -d --build
```

查看状态：

```bash
docker compose ps
docker compose logs -f api
```

浏览器访问：

```text
https://你的域名
```

登录入口：

```text
https://你的域名/login.html
```

阅读记录入口：

```text
https://你的域名/reading-history.html
```

### 4.7 复用宿主机 PostgreSQL 和 Nginx 部署

如果服务器已经有 PostgreSQL 和 Nginx，推荐仅将 Go API 打包为 Docker 镜像运行，不再启动 Compose 中的 `db`、`web`、`gateway` 服务。这样可避免与宿主机现有的数据库、80/443 端口和证书管理产生冲突。

运行边界如下：

```text
浏览器
  ↓ HTTPS :443
宿主机 Nginx
  ├─ 提供 /opt/learn-liangliang 下的静态文件和 content/
  ├─ 通过 auth_request 调用本机 Go API 校验登录会话
  └─ 将 /api/* 反向代理至 127.0.0.1:8080
                         ↓
                  Docker 中的 Go API
                         ↓
              宿主机 PostgreSQL（127.0.0.1:5432）
```

前提：域名已解析到服务器；宿主机 Nginx 已管理该域名的 HTTP/HTTPS 与证书；PostgreSQL 可由服务器本机访问。不要在防火墙或云安全组中开放 `8080`、`5432`。

#### 4.7.1 准备宿主机数据库和站点目录

若尚未创建数据库和账号，以 PostgreSQL 管理员身份执行：

```bash
sudo -u postgres psql
```

```sql
CREATE USER learn WITH PASSWORD '请替换为数据库强密码';
CREATE DATABASE learn_liangliang OWNER learn;
\q
```

将项目部署到宿主机目录，静态文件由 Nginx 直接读取：

```bash
git clone <仓库地址> /opt/learn-liangliang
sudo chmod -R a+rX /opt/learn-liangliang
```

#### 4.7.2 管理 API 私密配置

推荐把 API 配置放在项目目录外，通过 `--env-file` 注入，而不是打入镜像或特意挂载 YAML。这样镜像保持可移植，配置、密码与代码发布分离。

```bash
sudo install -d -m 700 /etc/learn-liangliang
sudo cp /opt/learn-liangliang/.env.example /etc/learn-liangliang/api.env
sudo chown root:root /etc/learn-liangliang/api.env
sudo chmod 600 /etc/learn-liangliang/api.env
sudo vi /etc/learn-liangliang/api.env
```

其中至少配置以下变量：

```env
# 使用宿主机 PostgreSQL；host 网络下 API 容器可通过 127.0.0.1 访问它
DATABASE_URL=postgres://learn:请替换为数据库强密码@127.0.0.1:5432/learn_liangliang?sslmode=disable

# API 仅供同一宿主机的 Nginx 调用
APP_ADDR=127.0.0.1:8080
APP_COOKIE_NAME=learn_session
APP_COOKIE_SECURE=true
APP_SESSION_TTL_HOURS=720

ADMIN_USERNAME=admin
ADMIN_PASSWORD=请替换为至少12位管理员强密码
ADMIN_DISPLAY_NAME=管理员
```

`POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD`、`SITE_DOMAIN` 与 `ACME_EMAIL` 可保留为空或删除：它们分别用于 Compose 数据库容器和 Caddy，不被此部署模式使用。`DATABASE_URL` 中的密码必须与实际 PostgreSQL 账号一致；HTTPS 生产环境必须保持 `APP_COOKIE_SECURE=true`。

如明确偏好 YAML，也可以将宿主机私有 YAML 以只读方式挂载，并设置 `APP_CONFIG_FILE`；但当前配置项数量较少，使用单一 `api.env` 可避免环境变量与 YAML 的优先级混淆。Nginx 配置和 TLS 证书不应挂载到 API 容器，应继续由宿主机 `/etc/nginx/` 和现有证书工具管理。

#### 4.7.3 构建并启动 API 镜像

在项目根目录构建带版本号的镜像：

```bash
cd /opt/learn-liangliang
docker build \
  -t learn-api:20260713 \
  -f backend/Dockerfile \
  backend
```

启动 API：

```bash
docker run -d \
  --name learn-api \
  --restart unless-stopped \
  --network host \
  --env-file /etc/learn-liangliang/api.env \
  learn-api:20260713
```

`--network host` 使容器可访问宿主机的 `127.0.0.1:5432` PostgreSQL。由于 `APP_ADDR=127.0.0.1:8080`，API 不会暴露到公网；不需要也不应添加 `-p 8080:8080`。API 镜像已内置数据库迁移文件，启动时会自动执行迁移并创建或更新管理员账号，无需再挂载 `backend/migrations`。

查看状态和日志：

```bash
docker ps --filter name=learn-api
docker logs -f learn-api
```

#### 4.7.4 配置宿主机 Nginx

建议为站点创建 `/etc/nginx/conf.d/learn-liangliang.conf`（Debian/Ubuntu 也可使用 `sites-available` 和 `sites-enabled`）。现有 Docker Nginx 配置中的上游服务名 `api:8080` 仅适用于 Compose 网络；宿主机 Nginx 必须改为 `127.0.0.1:8080`。

以下片段需合并到该站点的 HTTPS `server` 块中，证书路径与现有 HTTP 跳转配置沿用服务器原有写法：

```nginx
root /opt/learn-liangliang;
index index.html;

location /api/ {
    proxy_pass http://127.0.0.1:8080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

location = /internal/authenticate {
    internal;
    proxy_pass http://127.0.0.1:8080/internal/authenticate;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
    proxy_set_header X-Original-URI $request_uri;
    proxy_set_header X-Original-Method $request_method;
}
```

站点还必须保留 [`deploy/nginx/default.conf`](./deploy/nginx/default.conf) 中的以下规则：

- 对 `/专栏/**`、`/其他/**`、`/reading-history.html` 使用 `auth_request /internal/authenticate`，认证失败时跳转 `/login.html?next=<原 pathname>`；
- 对受保护内容使用 `Cache-Control: private, no-store`；
- 禁止直接访问 `/content/**`、`backend/`、`deploy/`、`.env` 等内部文件；
- 使用既有 `try_files` 规则将公开 URL 映射到物理 `content/` 目录。

修改后必须先检查再平滑加载：

```bash
sudo nginx -t && sudo systemctl reload nginx
```

#### 4.7.5 首次内容同步与验证

首次部署、数据库迁移后或 `content/` 发生变更后，必须运行一次内容同步；在同步成功前，API 不会将文章视为活动内容，也不会接受阅读进度写入。

```bash
docker run --rm \
  --network host \
  --env-file /etc/learn-liangliang/api.env \
  -v /opt/learn-liangliang/content:/content:ro \
  learn-api:20260713 \
  /app/content-sync --content-root /content
```

只在这个一次性同步容器中以只读形式挂载 `content/`；常驻 API 容器不需要挂载内容目录。

在服务器上检查 API：

```bash
curl -i http://127.0.0.1:8080/api/health
```

在任意可访问域名的机器上验证：

```bash
curl -I https://你的域名/login.html
curl -I https://你的域名/
curl -I https://你的域名/content/
```

预期登录页返回 `200`；未登录首页返回 `302` 至登录页；`/content/` 返回 `403`。还应在浏览器中验证登录、受保护内容跳转和阅读进度同步。

#### 4.7.6 更新与回滚

更新前先备份数据库，再拉取代码、构建新版本镜像并重建 API 容器：

```bash
cd /opt/learn-liangliang
sudo -u postgres pg_dump learn_liangliang > "backup-$(date +%F-%H%M%S).sql"

git pull --ff-only
IMAGE_TAG=20260713-$(git rev-parse --short HEAD)
docker build -t "learn-api:${IMAGE_TAG}" -f backend/Dockerfile backend

docker stop learn-api
docker rm learn-api

docker run -d \
  --name learn-api \
  --restart unless-stopped \
  --network host \
  --env-file /etc/learn-liangliang/api.env \
  "learn-api:${IMAGE_TAG}"

docker logs -f learn-api
```

如果本次更新修改了 `content/`，还需要用新镜像重新执行一次本节的“首次内容同步与验证”命令。镜像使用明确版本号而不是固定 `latest`，出现问题时可停止当前容器，再使用上一版本镜像按相同启动命令恢复。

### 4.8 PostgreSQL 备份

Docker Compose 模式可执行：

```bash
docker compose exec db sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > backup.sql
```

复用宿主机 PostgreSQL 的模式可执行：

```bash
sudo -u postgres pg_dump learn_liangliang > backup.sql
```

只要不删除 Compose 的 `postgres_data` volume，或不清理宿主机 PostgreSQL 数据库，重建 `web` / `api` 容器不会丢失阅读记录。

### 4.9 同步内容索引

内容目录变更后，执行一次性同步服务以更新活动内容、课程章节及用户课程汇总：

```bash
docker compose --profile tools run --rm content-sync
```

本地非 Docker 调试可使用同一 Go 实现：

```bash
cd backend
go run ./cmd/content-sync --content-root ../content
```

`content-sync` 仅以只读方式挂载或读取内容目录；它在事务内协调同步与进度保存，完成后退出，不会常驻或影响 `gateway`、`web`、`api`、`db` 的运行。

## 5. 免责声明 ⚠️

本项目仅用于技术学习与资料备份，所有内容版权归原作者所有，未经许可，请勿用于商业用途。

## 6. 相关信息

[![Star History Chart](https://api.star-history.com/svg?repos=xixiwenxuanhe/learn-liangliang&type=Date)](https://www.star-history.com/#xixiwenxuanhe/learn-liangliang&Date)

## 7. 许可证 📝

本项目采用 MIT License 开源，详见 [LICENSE](./LICENSE) 文件。
