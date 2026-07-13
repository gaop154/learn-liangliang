# learn-liangliang API

这是阅读进度同步后端，使用 Go + chi + pgx + PostgreSQL。

## 同机 Docker Compose 配置

根目录 `docker-compose.yaml` 仅运行常驻 `api` 与按需 `content-sync` 工具服务；Nginx、静态站点、TLS 和 PostgreSQL 由同一台宿主机独立管理。

Compose 使用 `network_mode: host`，将 Docker 宿主机的 `/etc/learn-liangliang/config.yaml` 只读挂载到容器内 `/app/config.yaml`，并设置 `APP_CONFIG_FILE=/app/config.yaml`。`api` 与 `content-sync` 都使用该配置；源文件缺失或不可读时，`create_host_path: false` 会使 Compose 失败，不会回退到默认配置。

`databaseUrl` 可直接使用宿主机 PostgreSQL 的用户名、密码及 `127.0.0.1` 端口，例如：

```yaml
databaseUrl: "postgres://learn:请替换为数据库强密码@127.0.0.1:5432/learn_liangliang?sslmode=disable"

app:
  addr: "127.0.0.1:8080"
  cookieName: "learn_session"
  cookieSecure: true
  sessionTtlHours: 720
```

API 只监听 `127.0.0.1:8080`，由宿主机 Nginx 反向代理 `/api/` 与 `/internal/authenticate`。`config.example.yaml` 默认适用于 HTTPS 生产部署，设置了 `cookieSecure: true`；本地 HTTP 调试复制为未提交的 `config.yaml` 后，才将其改为 `false`。根目录 `.env` 不被当前 API-only Compose 消费；不要将 API 数据库密码、`APP_*` 或管理员配置写入 `.env`。

启动与同步命令均显式指定 Compose 文件：

```bash
docker compose -f docker-compose.yaml config
docker compose -f docker-compose.yaml up -d --build
docker compose -f docker-compose.yaml --profile tools run --rm content-sync
docker compose -f docker-compose.yaml logs -f api
docker compose -f docker-compose.yaml down
```

默认 `docker compose -f docker-compose.yaml up -d` 不会启动带 `tools` profile 的 `content-sync`。首次部署、数据库迁移后或 `content/` 变更后必须手动运行一次同步。

## 本地运行

```bash
cd backend
cp config.example.yaml config.yaml
# 编辑 config.yaml 后启动
go run ./cmd/api
```

服务启动时会执行 `migrations/*.up.sql`，并在 YAML 或显式环境变量配置管理员账号后创建或更新管理员账号。

## 接口

- `GET /api/health`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `PUT /api/reading-progress`
- `GET /api/reading-progress?articlePath=...`
- `GET /api/reading-progress/recent?limit=20`
