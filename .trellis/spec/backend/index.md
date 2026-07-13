# 后端开发规范

> 本目录记录本项目当前真实约定：生产运行时是宿主机 Nginx 静态站点 + Docker Compose 中的 Go API + 宿主机 PostgreSQL；Python 仅保留为 `utils/*.py` 离线内容抓取、修补和迁移脚本。项目主体仍是静态内容归档站点，不是 Java/Spring 业务系统。

---

## 总览

后端相关入口与配置：

- `backend/`：Go API 服务，负责认证、会话和阅读进度同步。
- `backend/migrations/`：PostgreSQL 表结构迁移。
- `deploy/nginx/default.conf`：宿主机静态站点配置参考，负责旧公开 URL 到 `content/` 的内部映射、敏感目录拒绝和 API 鉴权；API 上游使用 `127.0.0.1:8081`。
- `deploy/caddy/Caddyfile`：历史网关配置参考，不由当前 Compose 运行。
- `docker-compose.yaml`：仅定义常驻 `api` 和带 `tools` profile 的一次性 `content-sync`；二者使用宿主机网络，并只读挂载宿主机 `/data/learn-liangliang/config.yaml`。
- `utils/*.py`：离线抓取首页、栏目、文章、PDF 和修补链接的脚本；不属于生产 Web 运行时。
- `static/index.js`、`static/reading-progress.js`：站点前端增强脚本，包括阅读进度、侧边栏、GitHub 入口与 Live2D；不加载第三方评论区或统计脚本。

已移除旧生产入口：`server_flask.py`、根 `Dockerfile`、旧 `learn-liangliang.conf` 和 `restart_nginx.sh` 不再作为运行时契约存在。

---

## 规范索引

| 指南 | 内容 | 状态 |
|------|------|------|
| [目录结构](./directory-structure.md) | 真实目录布局、Go API、Python 工具脚本、静态资源和部署配置放置规则 | 已填充 |
| [数据与持久化规范](./database-guidelines.md) | PostgreSQL 阅读进度数据、静态归档文件和脚本状态文件的使用方式 | 已填充 |
| [错误处理](./error-handling.md) | Go API JSON 错误、Nginx 拒绝规则、脚本 try/except | 已填充 |
| [质量规范](./quality-guidelines.md) | 路径安全、批量脚本风险、轻量验证、代码评审清单 | 已填充 |
| [日志规范](./logging-guidelines.md) | 宿主机 Nginx/PostgreSQL、Go API 容器日志与 Python 脚本 print 输出风格 | 已填充 |

---

## 开发前检查

- 确认任务是否真的涉及后端相关文件；很多页面内容只是静态 HTML 或资源文件。
- 如果涉及生产 API，先阅读 `backend/` 中的配置、认证、阅读进度和响应封装。
- 如果涉及静态访问路径，先阅读 `deploy/nginx/default.conf` 的 `try_files` 和 deny 规则。
- 如果涉及抓取或批量修补，先阅读对应的 `utils/*.py`，确认是否会联网、是否会写入大量文件。
- 如果涉及部署，必须同时检查 `docker-compose.yaml`、`deploy/nginx/default.conf` 和 `backend/Dockerfile` 的主机网络、`127.0.0.1:8081` 上游、私有配置挂载和内容挂载路径是否一致。
- 不要引入 Java/Spring、Mapper、XML SQL 或传统三层目录；当前已明确采用 Go + PostgreSQL。

---

## 质量检查

- Go 后端改动后运行 `cd backend && go test ./...`。
- 前端脚本改动后运行 `node --check static/index.js static/reading-progress.js`。
- Python 工具脚本改动后运行 `python -m py_compile utils/*.py`。
- Docker 编排改动后运行 `docker-compose -f docker-compose.yaml config`。
- 规范文档改动后检查 `.trellis/spec/backend/*.md` 不再描述旧 Flask/Gunicorn 为生产入口。
- 宿主机 Nginx 配置在目标环境运行后，仍需通过 `nginx -t`、服务日志或实际访问验证。

---

**文档语言**：本项目规范文档使用简体中文。
