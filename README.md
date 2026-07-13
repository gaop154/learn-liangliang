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

Nginx、静态站点、TLS 证书和 PostgreSQL 由同一台 CentOS 宿主机独立部署、运维和备份。API 容器使用 `network_mode: host`，因此 API 和 PostgreSQL 都使用宿主机回环地址；本部署的 API 必须监听 `127.0.0.1:8081`，宿主机 Nginx 将 `/api/` 和内部认证请求反向代理至该地址。不要对 8081 或 PostgreSQL 端口配置公网监听或防火墙放行。

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
git clone <仓库地址> /data/learn-liangliang
sudo chmod -R a+rX /data/learn-liangliang
```

创建仅供 API 和内容同步容器读取的配置文件。根目录 `.env` 不是此 Compose 的配置来源；不需要创建 `.env`，也不要将数据库密码、`APP_*` 或管理员配置写入其中。

```bash
cd /data/learn-liangliang
sudo install -d -m 700 /data/learn-liangliang
sudo install -m 600 backend/config.example.yaml /data/learn-liangliang/config.yaml
sudo chown root:root /data/learn-liangliang/config.yaml
sudo vi /data/learn-liangliang/config.yaml
```

`/data/learn-liangliang/config.yaml` 至少应设置实际 PostgreSQL 用户名、密码、127.0.0.1 端口和管理员账号：

```yaml
databaseUrl: "postgres://learn:请替换为数据库强密码@127.0.0.1:5432/learn_liangliang?sslmode=disable"

app:
  addr: "127.0.0.1:8081"
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

本项目是独立 HTTPS 站点，访问地址固定为 [https://nicenickname.cn:2085/](https://nicenickname.cn:2085/)，不是默认的 `https://nicenickname.cn/`（443）。`2083` 属于其他服务；不要将本项目的 `/api/` 或 `/internal/authenticate` 路由合入该端口对应的 `server` 块。

`2085` 必须配置为一个**完整静态站点**的 HTTPS `server` 块，而不能只放 `/api/` 与 `/internal/authenticate` 两个反向代理 location。只配置 API 代理会使其他请求落到 Nginx 默认站点，访问时出现 `Welcome to nginx!`。以 `deploy/nginx/default.conf` 的完整规则为基础复制到 `2085` 对应的 `server` 块，再按实际环境修改以下两项：

```nginx
server {
    listen 2085 ssl;
    server_name nicenickname.cn;

    # 证书配置沿用服务器现有 HTTPS 配置。
    root /data/learn-liangliang;
    index index.html;

    # 从 deploy/nginx/default.conf 完整复制其余 location 规则，
    # 并将全部 API 上游由 127.0.0.1:8080 改为 127.0.0.1:8081。
}
```

复制时必须保留 `deploy/nginx/default.conf` 中的全部关键规则，不能只摘取 API 代理：

- `/api/` 与 `location = /internal/authenticate`，上游均为 `http://127.0.0.1:8081`。
- `/专栏/**`、`/其他/**`、`/reading-history.html` 的 `auth_request`、安全的登录跳转，以及受保护内容的 `Cache-Control: private, no-store`。
- 公开 URL 到 `content/` 物理目录的 `try_files` 映射。
- `/content/**`、`backend/`、`deploy/`、`utils/`、隐藏文件和敏感根文件的 `deny` 规则。
- CSS、JavaScript、图片、字体和 PDF 的缓存规则；受保护内容规则必须保持在公共资源缓存规则之前，避免绕过鉴权。

#### Nginx 缺少 `auth_request` 模块时的处理

`auth_request` 是 `/专栏/**`、`/其他/**` 和 `/reading-history.html` 内容登录保护所需的 Nginx 模块。若 `nginx -t` 报 `unknown directive "auth_request"`，不得通过删除 `auth_request` 或受保护 location 的方式绕过；那会使受保护内容失去统一的登录校验。应使用与服务器二进制相同的 Nginx 版本重新编译，并显式加入 `--with-http_auth_request_module`。

已确认的服务器二进制为 `/usr/local/nginx/sbin/nginx`，版本是 `nginx/1.24.0`，现有编译参数仅为：

```text
--prefix=/usr/local/nginx --with-http_ssl_module --with-http_v2_module
```

以下命令保留上述参数，只追加认证子请求模块。命令以 `/usr/local/src` 作为源码目录；`curl` 和 `wget` 二选一即可。执行前先备份当前二进制与配置，且只执行 `make`，**不要执行 `make install`**，避免安装过程覆盖现有部署文件。

```bash
# 备份时间戳请保留，回退时需要使用对应的二进制备份。
backup_suffix="$(date +%F-%H%M%S)"
sudo cp -a /usr/local/nginx/sbin/nginx "/usr/local/nginx/sbin/nginx.backup-${backup_suffix}"
sudo tar -C /usr/local/nginx -czf "/usr/local/nginx/conf.backup-${backup_suffix}.tar.gz" conf

cd /usr/local/src
sudo curl -fLO https://nginx.org/download/nginx-1.24.0.tar.gz
# 或：sudo wget https://nginx.org/download/nginx-1.24.0.tar.gz
sudo tar -xzf nginx-1.24.0.tar.gz
cd nginx-1.24.0
sudo ./configure \
  --prefix=/usr/local/nginx \
  --with-http_ssl_module \
  --with-http_v2_module \
  --with-http_auth_request_module
sudo make

# 必须同时确认新二进制已编入模块，且使用现有配置能够通过语法检查。
sudo ./objs/nginx -V
sudo ./objs/nginx -t -c /usr/local/nginx/conf/nginx.conf
```

`./objs/nginx -V` 的输出应包含 `--with-http_auth_request_module`；仅在这两项验证成功后，才复制构建产物替换运行时二进制：

```bash
sudo install -m 755 ./objs/nginx /usr/local/nginx/sbin/nginx
```

替换二进制后，普通 `reload` 仅向正在运行的旧 master 发送重载信号，旧进程不会获得新编译的模块；不能只执行 `systemctl reload nginx` 或 `nginx -s reload` 来完成此次模块升级。为尽量不中断连接，应保留旧 master，按以下平滑二进制升级流程操作。下列 PID 文件路径基于当前 `--prefix=/usr/local/nginx`；如配置中通过 `pid` 指令另行指定，须改用实际路径。

```bash
# 先确认当前旧 master PID；此时 nginx.pid 指向旧 master。
sudo cat /usr/local/nginx/logs/nginx.pid

# 让旧 master 启动新二进制；成功后会保留旧 PID 到 nginx.pid.oldbin，
# 并创建新的 nginx.pid。两个 master 短时间内会同时存在。
sudo sh -c 'kill -USR2 "$(cat /usr/local/nginx/logs/nginx.pid)"'
sudo cat /usr/local/nginx/logs/nginx.pid
sudo cat /usr/local/nginx/logs/nginx.pid.oldbin
sudo ps -fp "$(sudo cat /usr/local/nginx/logs/nginx.pid)" \
  "$(sudo cat /usr/local/nginx/logs/nginx.pid.oldbin)"

# 在新 master 确认可用后，优雅退出旧 master；不要误向 nginx.pid 发送 QUIT。
sudo sh -c 'kill -QUIT "$(cat /usr/local/nginx/logs/nginx.pid.oldbin)"'
```

`nginx.pid.oldbin` 仅在 `USR2` 平滑升级期间指向旧 master；在确认新进程、站点和认证链路正常前，不要执行最后的 `QUIT`，也不要删除该文件或备份。后续仅修改配置而不更换模块时，才可使用正常的 `nginx -t` 加 `reload` 流程。

若新二进制或新配置异常，且旧 master 尚未执行 `QUIT`，旧 master 及其 worker 仍在运行；只需优雅退出新 master，再恢复磁盘上的备份二进制。不要对旧 master 发送 `HUP`，因为它不具备 `auth_request` 模块，重载包含该指令的配置会失败。将示例备份文件名替换为实际生成的时间戳文件名。

```bash
# nginx.pid.oldbin 是仍在服务的旧 master，nginx.pid 是新 master。
sudo sh -c 'kill -QUIT "$(cat /usr/local/nginx/logs/nginx.pid)"'
sudo install -m 755 /usr/local/nginx/sbin/nginx.backup-<时间戳> \
  /usr/local/nginx/sbin/nginx
sudo /usr/local/nginx/sbin/nginx -t -c /usr/local/nginx/conf/nginx.conf
```

若已经结束旧 master，不能再依赖 `nginx.pid.oldbin` 回退；应在维护窗口内恢复备份二进制和配置备份，再按实际服务管理方式启动 Nginx。上述编译、升级和回退命令是针对已确认版本及参数的操作指引，尚未在该服务器上实际执行或验证。

配置完成后，在服务器上验证、重载并检查 API 与 HTTPS 首页：

```bash
sudo nginx -t
sudo systemctl reload nginx
curl -i http://127.0.0.1:8081/api/health
curl -k -I https://nicenickname.cn:2085/
```

若仍看到 `Welcome to nginx!`，依次检查：请求是否确实使用 `https://nicenickname.cn:2085/`；该域名和端口是否命中此 `server` 块；该块是否含有 `root /data/learn-liangliang`、`index index.html` 和完整 `try_files` 规则；以及修改后的配置是否已通过 `nginx -t` 并成功重载。以上命令仅为部署后的人工验证步骤，本文档不表示已在服务器上验证成功。

### 3.4 启动 API

先验证 Compose，再构建并启动默认的 `api` 服务：

```bash
cd /data/learn-liangliang
docker-compose -f docker-compose.yaml config
docker-compose -f docker-compose.yaml up -d --build
docker-compose -f docker-compose.yaml ps
docker-compose -f docker-compose.yaml logs -f api
```

默认 `docker-compose -f docker-compose.yaml up -d` 仅启动 `api`；`content-sync` 因 `tools` profile 不会自动运行。API 启动时会执行数据库迁移，并创建或更新管理员账号。

若服务器网络无法访问 Go 官方模块服务，可仅在本次构建前临时指定模块代理和校验库：

```bash
GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn \
  docker-compose -f docker-compose.yaml up -d --build
```

该配置只作为 Docker 构建参数传入 `go mod download`，不会写入私有 `config.yaml` 或容器运行时环境；网络正常时无需设置，仍使用 Go 官方默认值。

可在服务器本机检查 API：

```bash
curl -i http://127.0.0.1:8081/api/health
```

### 3.5 首次与按需内容同步

首次部署、数据库迁移后，或 `content/` 有新增、删除、恢复或顺序调整时，必须执行内容同步。同步服务只读挂载与 Compose 文件相对的 `./content` 目录，完成后自动退出：

```bash
cd /data/learn-liangliang
docker-compose -f docker-compose.yaml --profile tools run --rm content-sync
```

常驻 `api` 不挂载 `content/`，不会因同步工具运行而读取或修改归档文件。内容同步成功前，API 不会把文章视为活动内容或接受阅读进度写入。

### 3.6 更新、停机和备份

更新 API 或内容时：

```bash
cd /data/learn-liangliang
sudo -u postgres pg_dump learn_liangliang > "backup-$(date +%F-%H%M%S).sql"
git pull --ff-only
docker-compose -f docker-compose.yaml config
docker-compose -f docker-compose.yaml up -d --build
docker-compose -f docker-compose.yaml --profile tools run --rm content-sync
docker-compose -f docker-compose.yaml logs -f api
```

若更新不涉及 `content/` 或索引逻辑，可省略最后的 `content-sync` 命令。停止 API 容器：

```bash
docker-compose -f docker-compose.yaml down
```

该命令不会停止宿主机 Nginx 或 PostgreSQL，也不会删除宿主机数据库数据。仅停止 API 而保留 Compose 资源可执行：

```bash
docker-compose -f docker-compose.yaml stop api
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
