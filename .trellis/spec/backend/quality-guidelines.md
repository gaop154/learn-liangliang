# 质量规范

> 本项目是静态归档站点，生产运行时为 Caddy + Nginx 静态站点 + Go API + PostgreSQL，另保留 Python 离线内容维护脚本。质量重点是路径安全、静态链接可用、API 合约一致、数据库持久化、批量脚本可控、部署配置与实际服务名/端口一致，以及不要误伤大量归档内容。

---

## 总览

当前仓库没有发现 pytest、ruff、black、mypy、ESLint、pnpm/npm 构建脚本或 CI 配置。轻量验证应优先使用不会联网、不会批量改写内容的检查方式，例如：

- Go 后端测试：`cd backend && go test ./...`。
- 前端脚本语法检查：`node --check static/index.js static/reading-progress.js`。
- Python 工具脚本语法检查：`python -m py_compile utils/*.py`。
- Docker 编排检查：`docker compose config`。
- 占位符和旧架构检查：确认 `.trellis/spec/backend/*.md` 不再把 Flask/Gunicorn 描述为生产入口。
- 如修改 Nginx/Caddy 配置，应通过 `docker compose config` 和容器实际访问验证；目标环境有独立 Nginx 时再执行 `nginx -t`。

---

## 必须保持的质量要求

### 1. 静态站路径安全

生产静态站点由 `deploy/nginx/default.conf` 提供路径控制：

```nginx
location ~ /\. {
    deny all;
}

location ~* ^/(backend|deploy|utils|ssl|content)(/|$) {
    deny all;
}

location / {
    try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404;
}
```

修改静态服务配置时必须保留或增强以下语义：

- 不直接暴露 `.git`、隐藏路径、`backend/`、`deploy/`、`utils/`、`content/` 等内部目录。
- 旧公开 URL 仍能内部映射到 `content/`。
- 缺失文件返回 404，不回退到敏感目录。

### 2. Go API 合约一致

`backend/` 是生产业务 API 的唯一入口。修改认证、会话或阅读进度接口时必须保持：

- JSON 错误响应使用 `internal/response` 的统一结构。
- `articlePath` 使用公开 canonical URL，不能保存 `/content/...` 物理路径。
- 登录态使用 HttpOnly Cookie，不把长期凭据放进 `localStorage`。
- 数据库访问通过现有 `pgx`/`sqlc` 模式，不在处理器里拼接复杂 SQL。
- 新增数据库结构必须配套迁移脚本和必要索引/约束。

### 3. 不破坏静态归档路径

HTML、脚本和阅读进度依赖当前公开 URL 形态：

- 首页：`index.html`。
- 公开分类 URL：`/专栏/`、`/文章/`、`/极客时间/`、`/恋爱必修课/`、`/PDF/`、`/assets/`。
- 物理内容根：`content/专栏/`、`content/文章/`、`content/极客时间/`、`content/恋爱必修课/`、`content/PDF/`、`content/assets/`。
- 内容文件：`*.md.html`。
- 公共静态资源：`static/`、`img/`、`live-2d/`。

不要随意批量重命名中文目录、空格路径或 `.md.html` 后缀。迁移内容文件时，应保持旧公开 URL 可访问，不能要求用户访问 `/content/...`。

### 4. 批量脚本要可控

`utils/03_patch_others.py`、`utils/04_patch_pdf.py` 等脚本会访问外网并写入大量文件。修改这类脚本时应保持：

- 已存在文件跳过逻辑。
- 单项失败后继续处理后续项目。
- 429 限流重试和等待。
- 输出总数、进度、失败原因和保存率。
- 不引入生产 Web 运行时依赖；Python 依赖只服务离线工具。

### 5. 部署服务名与端口一致性

当前生产编排约定：

- `gateway` 使用 Caddy，对外暴露 `80:80` 和 `443:443`。
- `deploy/caddy/Caddyfile` 将 `/api/*` 反代到 `api:8080`，其他请求到 `web:80`。
- `web` 使用 `nginx:1.27-alpine`，挂载 `deploy/nginx/default.conf`。
- `api` 使用 `backend/Dockerfile` 构建，默认监听 `APP_ADDR=:8080`。
- `db` 使用 PostgreSQL 16 Alpine，数据写入 `postgres_data` volume。

修改服务名、端口或路径时必须同步检查 `docker-compose.yml`、`deploy/caddy/Caddyfile`、`deploy/nginx/default.conf`、`backend/Dockerfile` 和 `.env.example`。

### 6. 简体中文文档与提示

仓库 README、脚本输出和本规范均使用简体中文。新增文档、脚本提示和说明应使用简体中文。

---

## 禁止或高风险模式

- 禁止把本仓库当作 Java/Spring 后端来新增 Mapper、Service、Controller、XML SQL 等结构。
- 禁止重新引入 Flask/Gunicorn 作为生产 Web 运行时；静态站点由 Nginx 承担，业务 API 由 Go 承担。
- 禁止在未确认影响范围时运行 `utils/03_patch_zhuanlan.sh`；该脚本包含 `git commit` 和 `git push -f origin main`。
- 禁止在轻量验证中运行会大规模下载或改写内容的脚本。
- 禁止将代理密码、证书私钥、访问令牌等敏感信息写入新增日志或文档示例。
- 禁止删除 `deploy/nginx/default.conf` 中隐藏路径、内部目录和调试文件拦截规则，除非有明确替代方案。
- 禁止为了“整理”而批量格式化几万份归档 HTML；这会制造巨大 diff 且容易破坏内容。
- 禁止无需求引入前端构建链路；当前站点直接使用静态 HTML/CSS/JS。

---

## 建议验证命令

根据改动范围选择最轻量的检查：

```bash
# Go API 测试
cd backend && go test ./...
```

```bash
# 前端脚本语法检查
node --check static/index.js static/reading-progress.js
```

```bash
# Python 离线工具脚本语法检查，不访问网络、不写业务内容
python -m py_compile utils/*.py
```

```bash
# Docker Compose 配置检查
docker compose config
```

```bash
# 检查 Trellis 后端规范是否仍有初始英文占位符或旧生产入口描述
python - <<'PY'
from pathlib import Path
markers = ['To be filled', 'Flask 静态文件服务入口', '通过 Gunicorn 启动']
for p in Path('.trellis/spec/backend').glob('*.md'):
    text = p.read_text(encoding='utf-8')
    if any(marker in text for marker in markers):
        print(p)
PY
```

---

## 代码评审清单

- 是否符合当前项目形态：静态站点 + Go API + PostgreSQL + Caddy/Nginx + Python 离线脚本？
- 是否引用了真实存在的文件路径，例如 `backend/`、`utils/*.py`、`static/index.js`、`static/reading-progress.js`、`docker-compose.yml`、`deploy/caddy/Caddyfile`、`deploy/nginx/default.conf`？
- 是否保留中文路径、`.md.html` 后缀和现有静态资源目录？
- 是否避免了不必要的大规模格式化或批量重写归档内容？
- 如果改动了抓取脚本，是否保留 timeout、重试、跳过已存在文件、中文进度输出？
- 如果改动了静态站配置，是否保留 403/404 语义、内部目录拒绝和旧 URL 到 `content/` 的映射？
- 如果改动了 Go API，是否保持 JSON 错误结构、认证 Cookie 和 `articlePath` canonical 合约？
- 如果改动了端口或部署方式，是否同步检查 `docker-compose.yml`、`deploy/caddy/Caddyfile`、`deploy/nginx/default.conf`、`backend/Dockerfile` 和 `.env.example`？
- 是否运行了与改动范围匹配的轻量验证？未能验证的部分是否明确说明？
