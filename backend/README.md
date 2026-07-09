# learn-liangliang API

这是阅读进度同步后端，使用 Go + chi + pgx + PostgreSQL。

## 环境变量

参考仓库根目录 `.env.example`。

## 本地运行

```bash
cd backend
go run ./cmd/api
```

服务启动时会执行 `migrations/*.up.sql`，并在配置了 `ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 时创建或更新管理员账号。

## 接口

- `GET /api/health`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `PUT /api/reading-progress`
- `GET /api/reading-progress?articlePath=...`
- `GET /api/reading-progress/recent?limit=20`
