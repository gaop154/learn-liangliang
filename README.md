# learn-liangliangleang 技术内容归档

原始网站：[learn.lianglianglee.com](https://learn.lianglianglee.com)

推荐访问：[learn-liangliang.wenxuanhe.top](https://learn-liangliang.wenxuanhe.top)

---

## 1. 项目说明

本项目对原始网站内容进行了系统性备份和整理，长期保存中文技术专栏和文章。生产运行时由宿主机 Nginx 提供静态内容、HTTPS 和内容访问控制，由 Docker Compose 仅运行 Go API；PostgreSQL 同样由宿主机单独管理。

## 2. 项目目录说明

- **content/**：文章相关归档内容物理根目录。
  - **content/专栏/**：技术专栏归档，对外路径为 `/专栏/`。
  - **content/其他/**：非课程分类归档，对外路径为 `/其他/`。
  - 各分类和课程保留各自的 `assets/` 子目录。
- **static/**：前端样式、脚本与公共静态资源。
- **img/**：页面截图与图片资源。
- **backend/**：Go API 服务，提供登录、会话和阅读进度同步接口。
- **deploy/nginx/**：供宿主机 Nginx 参考的静态站点配置，负责公开 URL 到 `content/` 的内部映射和访问控制。
- **deploy/caddy/**：历史 Caddy 配置参考；当前 Compose 不启动 Caddy。
- **utils/**：离线内容抓取、修补和迁移脚本，不属于生产 Web 运行时。

公开访问 URL 使用 `/专栏/...` 与 `/其他/...`，不会暴露为 `/content/...`。旧 Flask/Gunicorn 运行时已移除。

## 3. 同机部署

根目录 `docker-compose.yaml` 仅定义两个服务：

- `api`：常驻 Go API，使用宿主机网络并以 `restart: unless-stopped` 运行。
- `content-sync`：带 `tools` profile 的一次性内容索引工具，默认 `up -d` 不会启动它。

Nginx、静态站点、TLS 证书和 PostgreSQL 由同一台 CentOS 宿主机独立部署、运维和备份。API 容器使用 `network_mode: host`，因此 API 和 PostgreSQL 都使用宿主机回环地址；API 必须监听 `127.0.0.1:8080`，宿主机 Nginx 将 `/api/` 和内部认证请求反向代理至该地址。不要对 8080 或 PostgreSQL 端口配置公网监听或防火墙放行。

### 3.1 准备 PostgreSQL

以 PostgreSQL 管理员身份创建专用账号和数据库；请替换示例密码：

```bash
sudo -u postgres psql
```

```sql
CREATE USER learn WITH PASSWORD '请替换为数据库强密码';
CREATE DATABASE learn_liangliang OWNER learn;
\q
```

PostgreSQL 保持只监听宿主机所需地址即可。本 Compose 不创建数据库容器、网络或数据卷；备份请使用宿主机 PostgreSQL 的 `pg_dump`。

### 3.2 准备站点和 API 私有配置

将项目部署到宿主机，例如：

```bash
git clone <仓库地址> /opt/learn-liangliang
sudo chmod -R a+rX /opt/learn-liangliang
```

创建仅供 API 和内容同步容器读取的配置文件。根目录 `.env` 不是此 Compose 的配置来源；不需要创建 `.env`，也不要将数据库密码、`APP_*` 或管理员配置写入其中。

```bash
cd /opt/learn-liangliang
sudo install -d -m 700 /etc/learn-liangliang
sudo install -m 600 backend/config.example.yaml /etc/learn-liangliang/config.yaml
sudo chown root:root /etc/learn-liangliang/config.yaml
sudo vi /etc/learn-liangliang/config.yaml
```

`/etc/learn-liangliang/config.yaml` 至少应设置实际 PostgreSQL 用户名、密码、127.0.0.1 端口和管理员账号：

```yaml
databaseUrl: "postgres://learn:请替换为数据库强密码@127.0.0.1:5432/learn_liangliang?sslmode=disable"

app:
  addr: "127.0.0.1:8080"
  cookieName: "learn_session"
  cookieSecure: true
  sessionTtlHours: 720

admin:
  username: "admin"
  password: "请替换为至少12位管理员强密码"
  displayName: "管理员"
```

Compose 使用固定的只读 bind mount 将该文件挂载为 `/app/config.yaml`，并设置 `APP_CONFIG_FILE=/app/config.yaml`。挂载采用 `create_host_path: false`：源文件缺失、路径错误或无读取权限时，Compose 会失败，不会创建空文件或回退到默认配置。

### 3.3 配置宿主机 Nginx

宿主机 Nginx 负责 HTTPS、静态文件和鉴权。`deploy/nginx/default.conf` 是以 `/opt/learn-liangliang` 为站点根目录的宿主机部署配置基线，可按域名、证书路径和 HTTPS `server` 块合并到实际配置；其 API 上游为本机回环地址：

```nginx
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
    proxy_set_header X-Original-URI $uri;
    proxy_set_header X-Original-Method $request_method;
}
```

站点配置还必须保留以下语义：

- `/专栏/**`、`/其他/**` 和 `/reading-history.html` 使用 `auth_request /internal/authenticate`；认证失败时仅携带原始 pathname 跳转到 `/login.html`。
- 受保护内容响应使用 `Cache-Control: private, no-store`。
- `/content/**`、`backend/`、`deploy/`、`utils/` 和隐藏文件保持禁止公开访问。
- 使用 `try_files` 将公开 URL 映射到 `content/` 物理目录。

变更后先验证和加载宿主机 Nginx：

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### 3.4 启动 API

先验证 Compose，再构建并启动默认的 `api` 服务：

```bash
cd /opt/learn-liangliang
docker compose -f docker-compose.yaml config
docker compose -f docker-compose.yaml up -d --build
docker compose -f docker-compose.yaml ps
docker compose -f docker-compose.yaml logs -f api
```

默认 `docker compose -f docker-compose.yaml up -d` 仅启动 `api`；`content-sync` 因 `tools` profile 不会自动运行。API 启动时会执行数据库迁移，并创建或更新管理员账号。

可在服务器本机检查 API：

```bash
curl -i http://127.0.0.1:8080/api/health
```

### 3.5 首次与按需内容同步

首次部署、数据库迁移后，或 `content/` 有新增、删除、恢复或顺序调整时，必须执行内容同步。同步服务只读挂载与 Compose 文件相对的 `./content` 目录，完成后自动退出：

```bash
cd /opt/learn-liangliang
docker compose -f docker-compose.yaml --profile tools run --rm content-sync
```

常驻 `api` 不挂载 `content/`，不会因同步工具运行而读取或修改归档文件。内容同步成功前，API 不会把文章视为活动内容或接受阅读进度写入。

### 3.6 更新、停机和备份

更新 API 或内容时：

```bash
cd /opt/learn-liangliang
sudo -u postgres pg_dump learn_liangliang > "backup-$(date +%F-%H%M%S).sql"
git pull --ff-only
docker compose -f docker-compose.yaml config
docker compose -f docker-compose.yaml up -d --build
docker compose -f docker-compose.yaml --profile tools run --rm content-sync
docker compose -f docker-compose.yaml logs -f api
```

若更新不涉及 `content/` 或索引逻辑，可省略最后的 `content-sync` 命令。停止 API 容器：

```bash
docker compose -f docker-compose.yaml down
```

该命令不会停止宿主机 Nginx 或 PostgreSQL，也不会删除宿主机数据库数据。仅停止 API 而保留 Compose 资源可执行：

```bash
docker compose -f docker-compose.yaml stop api
```

## 4. 本地非 Docker 开发

本地开发可以直接运行 PostgreSQL 和 Go API。复制配置样例为未提交的本地配置：

```bash
cd backend
cp config.example.yaml config.yaml
# 编辑 config.yaml，使用本机 PostgreSQL 用户名、密码和端口
go run ./cmd/api
```

本地 HTTP 调试应将 `app.cookieSecure` 设为 `false`。如需同步本地归档内容，使用相同 Go 实现：

```bash
cd backend
go run ./cmd/content-sync --content-root ../content
```

## 5. 免责声明

本项目仅用于技术学习与资料备份，所有内容版权归原作者所有，未经许可，请勿用于商业用途。

## 6. 许可证

本项目采用 MIT License 开源，详见 [LICENSE](./LICENSE) 文件。
