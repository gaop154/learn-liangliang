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

生产部署使用 Docker Compose 编排，运行时边界固定为 Caddy + Nginx 静态站点 + Go API + PostgreSQL。Python 只用于本地或离线执行 `utils/*.py` 内容维护脚本，不参与生产 Web 请求链路。

- `gateway`：Caddy 入口，对外暴露 80/443，负责将 `/api/*` 转发到 Go API，其余请求转发到静态站点。
- `web`：Nginx 静态站点，托管现有 HTML、PDF、图片和前端脚本，并把旧公开 URL 内部映射到 `content/`。
- `api`：Go 后端 API，提供登录、会话和阅读进度同步。
- `db`：PostgreSQL，使用 `postgres_data` volume 持久化数据。
- `content-sync`：按需执行的一次性内容索引服务，仅只读挂载 `content/`，同步完成后自动退出。

### 4.1 本地启动

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

### 4.2 本地非 Docker 启动

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

    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }

    @blocked path /.git* /.claude* /.trellis* /backend* /deploy* /utils* /content* /docker-compose.yml /requirements.txt /README.md /.env*
    respond @blocked 403

    try_files {path} {path}/ /content{path} /content{path}/index.html /content{path}/
    file_server
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
- `Caddyfile.local` 只是本地临时配置，不建议提交到仓库。
- 如果本机 PostgreSQL 端口、账号或密码不同，请同步修改 `backend/config.yaml` 中的 `databaseUrl`。
- 停止 Go API 或 Caddy 时，在各自终端按 `Ctrl + C`。

### 4.3 CentOS 安装 Docker

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

### 4.4 配置环境变量

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

### 4.5 生产启动

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

### 4.6 PostgreSQL 备份

```bash
docker compose exec db sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' > backup.sql
```

只要不删除 `postgres_data` volume，重建 `web` / `api` 容器不会丢失阅读记录。

### 4.7 同步内容索引

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
