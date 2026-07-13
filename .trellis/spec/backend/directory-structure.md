# 目录结构规范

> 本项目不是传统 Java/Spring 后端业务系统；当前真实形态是“静态内容归档站点 + 宿主机 Nginx + Docker Compose 中的 Go API + 宿主机 PostgreSQL + Python 离线内容维护脚本”。本文件记录当前仓库已经存在的组织方式，避免后续任务误按旧 Flask/Gunicorn 入口或大型后端分层来改造。

---

## 总览

当前仓库的核心内容是已归档的 HTML、PDF、图片和静态资源。生产运行时职责分为三类：

1. 宿主机 Nginx：托管静态 HTML/CSS/JS/PDF/图片、把旧公开 URL 内部映射到 `content/`，通过 `auth_request` 保护内容区域，并反向代理 API。
2. Go API：由 Docker Compose 常驻 `api` 服务运行，负责认证、会话、Nginx 内部会话校验和阅读进度同步；使用宿主机网络监听 `127.0.0.1:8081`。
3. 宿主机 PostgreSQL：持久化用户、会话和阅读进度数据。

Python 代码仅保留在 `utils/*.py` 中，承担一次性或批处理抓取、修补归档内容的离线脚本职责，直接读写仓库中的 HTML/PDF/assets 文件。生产 Web 请求链路不再依赖 Python、Flask、Gunicorn 或根目录 Python `Dockerfile`。

---

## 当前目录布局

```text
learn-liangliang/
├── index.html                         # 站点首页，静态 HTML
├── requirements.txt                   # Python 离线工具脚本依赖：requests、beautifulsoup4
├── docker-compose.yaml                # api 常驻与一次性 content-sync 工具编排
├── .env.example                       # API-only Compose 不使用 .env 的说明
├── backend/                           # Go API 服务
│   ├── Dockerfile                     # Go API 多阶段构建镜像
│   ├── cmd/api/main.go                # API 启动入口
│   ├── cmd/content-sync/main.go       # 一次性内容目录同步入口
│   ├── internal/                      # 配置、认证、内容索引、阅读进度、DB、响应封装
│   ├── migrations/                    # PostgreSQL 迁移脚本
│   └── sqlc.yaml                      # sqlc 配置
├── deploy/
│   ├── caddy/Caddyfile                # Caddy 网关配置
│   └── nginx/default.conf             # 静态站点 Nginx 配置
├── static/                            # 前端 CSS/JS、highlight、评论区和阅读进度脚本等
│   ├── index.css
│   ├── index.js
│   ├── reading-progress.js
│   ├── highlight.min.css
│   └── highlight.min.js
├── img/                               # README 截图、GitHub 图标、分类图片
├── live-2d/                           # Live2D 模型和脚本资源
├── content/                           # 内容物理根目录，不作为主公开 URL 暴露
│   ├── 专栏/
│   └── 其他/
│       ├── 恋爱必修课/
│       ├── 文章/
│       ├── 极客时间/
│       └── PDF/
└── utils/                             # 抓取、修补、批处理脚本
    ├── 01_download_index.py
    ├── 02_download_menu_index.py
    ├── 03_patch_others.py
    ├── 03_patch_zhuanlan.sh
    ├── 04_patch_pdf.py
    ├── 05_migrate_other_content.py
    ├── gen_task_json.py
    ├── migrate_content_root.py
    ├── proxy_pool.py
    └── task.json
```

---

## 模块组织约定

### Go API 服务

- Go 服务代码集中放在 `backend/`。
- `backend/cmd/api/main.go` 是启动入口。
- `backend/internal/config/` 负责环境变量配置。
- `backend/internal/auth/` 负责密码哈希、会话令牌、管理员预置账号和认证接口。
- `backend/internal/reading/` 负责阅读进度接口。
- `backend/internal/db/` 与 `backend/internal/db/sqlc/` 负责数据库连接和 sqlc 生成代码。
- `backend/internal/response/` 负责统一 JSON 响应。
- SQL 迁移放在 `backend/migrations/`，查询定义放在 `backend/internal/db/queries/`。

不要把新业务 API 加入 Python 脚本或重新引入 Flask 服务。需要新增阅读进度、账号或会话能力时，应在 `backend/` 内按现有 Go 模块边界扩展。

### 工具脚本

- 抓取和修补脚本集中放在 `utils/`，只作为离线内容维护工具。
- 脚本按处理顺序使用数字前缀命名，例如：
  - `utils/01_download_index.py`：抓取首页并下载 CSS/图片/JS 到 `static/`。
  - `utils/02_download_menu_index.py`：抓取栏目菜单页。
  - `utils/03_patch_others.py`：按 `--column` 下载栏目中的 `.md` 内容并保存为 `.md.html`。
  - `utils/04_patch_pdf.py`：从 `content/其他/PDF/index.html` 解析 PDF 链接并下载 PDF。
- 批处理 Shell 脚本也放在 `utils/`，例如 `utils/03_patch_zhuanlan.sh` 调用 `03_patch_others.py` 并更新 `utils/task.json`。
- `requirements.txt` 仅为这些离线脚本保留 `requests`、`beautifulsoup4` 等依赖，不应重新加入 Flask/Gunicorn 运行时依赖。

真实示例：

```python
# utils/03_patch_others.py
parser = argparse.ArgumentParser(description="批量下载专栏内容")
parser.add_argument("--column", type=str, default="恋爱必修课", help="专栏目录名，默认'恋爱必修课'")
```

### 静态前端资源

- 站点主页面和内容页是静态 HTML，不使用前端构建工具。
- 公共脚本和样式放在 `static/`，例如 `static/index.js`、`static/reading-progress.js` 和 `static/index.css`。
- `static/index.js` 承担站点基础增强职责：记录上次阅读路径、侧边栏交互、GitHub 悬浮入口、Live2D 注入、页脚修改；不得注入第三方评论区或统计脚本。
- `static/reading-progress.js` 承担读取登录态、查询/上报阅读进度、提示恢复进度等跨设备同步逻辑。
- `img/` 用于站点截图和图标，各课程或分类内的 `assets/` 用于被内容页直接引用的归档资源，`live-2d/` 用于 Live2D 模型资源。

### 部署配置

- `docker-compose.yaml` 是 API 部署入口，只定义：
  - `api`：使用 `backend/Dockerfile` 构建、`network_mode: host`、`restart: unless-stopped` 的常驻 Go API；固定只读挂载宿主机 `/data/learn-liangliang/config.yaml` 至 `/app/config.yaml`，并设置 `APP_CONFIG_FILE=/app/config.yaml`。
  - `content-sync`：仅在 `tools` profile 中执行、`network_mode: host`、`restart: "no"` 的同步容器；复用同一私有 YAML，另只读挂载与 Compose 文件相对的 `./content`，完成后退出。
- Compose 不定义 gateway、web、db、端口映射、网络或持久化 volume；宿主机 Nginx 和 PostgreSQL 分别管理静态内容、TLS、反向代理和数据持久化。
- `.env` 不被当前 Compose 消费；API 与内容同步的私密配置固定来自宿主机 YAML，不应写入 `.env`。
- `deploy/nginx/default.conf` 是以 `/opt/learn-liangliang` 为站点根目录的宿主机 Nginx 配置基线，负责静态站点路径映射、缓存、敏感目录拒绝和 API 代理；API 上游必须为 `127.0.0.1:8081`。
- `deploy/caddy/Caddyfile` 是历史配置参考，当前 Compose 不启动 Caddy。
- 根目录旧 Python `Dockerfile`、`server_flask.py`、旧 `learn-liangliang.conf`、`restart_nginx.sh` 不再作为生产运行时入口存在。

---

## 命名约定

- 归档内容目录保留中文名称：`content/专栏/`、`content/其他/恋爱必修课/`、`content/其他/文章/`、`content/其他/极客时间/`、`content/其他/PDF/`，不要为了代码习惯批量改成英文目录。
- 公开 URL 使用：`/专栏/...`、`/其他/恋爱必修课/...`、`/其他/文章/...`、`/其他/极客时间/...`、`/其他/PDF/...`；不保留旧分类路径映射。
- 归档文章文件当前使用 `*.md.html` 后缀，例如 `content/专栏/10x程序员工作法/00 开篇词 程序员解决的问题，大多不是程序问题.md.html`；脚本中也有把 `.md` 替换为 `.md.html` 的逻辑。
- 工具脚本按执行阶段使用数字前缀：`01_`、`02_`、`03_`、`04_`、`05_`。
- Python 变量和函数使用 snake_case，例如 `safe_filename`、`get_links`、`download_static_resources`。
- Go 包名使用小写短名，按现有 `auth`、`catalog`、`reading`、`config`、`response` 风格组织。
- 静态文件沿用原站或现有资源命名，例如 `static/highlight.min.js`、`static/email-decode.min.js` 与分类内 `assets/` 资源。

---

## 新增或修改文件时放哪里

- 新的生产 API：放在 `backend/` 对应模块内；不要新增 Flask 路由。
- 新的数据库表或索引：放在 `backend/migrations/`，查询放在 `backend/internal/db/queries/`，生成代码放在 `backend/internal/db/sqlc/`。
- 新的内容抓取/修补流程：放在 `utils/`，按现有数字前缀延续命名，并尽量支持命令行参数，避免硬编码只能处理一个栏目。
- 新的全站前端增强：放在 `static/index.js`、`static/reading-progress.js` 或对应 CSS 文件中；如果是第三方静态库，放在 `static/` 或更明确的子目录。
- 新的站点公共图片：根据用途放到 `img/`；内容型页面和内容私有资源放到 `content/<分类>/...`，内容页局部图片保持在对应内容目录的 `assets/` 子目录。
- 部署相关变更：API 工具编排改 `docker-compose.yaml`，Go API 镜像改 `backend/Dockerfile`，宿主机 Nginx 静态站点规则参考 `deploy/nginx/default.conf`；Caddy 配置不属于当前 Compose 运行时。

---

## Scenario: content 内容根目录与旧 URL 兼容

### 1. Scope / Trigger

- Trigger：当文章、专栏、PDF、捐赠页等内容型静态文件需要整理目录结构，或修改静态站点 Nginx/抓取脚本的内容路径时，必须遵守本契约。
- Scope：`content/` 是物理内容根；公开 URL 继续保持历史路径，避免破坏站内链接、外部收藏、阅读进度和评论映射。

### 2. Signatures

- 迁移命令：`python utils/migrate_content_root.py [--dry-run|--execute]`
- 静态站点公开路径：`GET /<path:filename>`
- Nginx 旧路径映射：`try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404`
- 内容生成脚本输出根：`content_dir = os.path.join(base_dir, "content")`

### 3. Contracts

- 物理内容目录只能放在：`content/专栏/` 与 `content/其他/{恋爱必修课,文章,极客时间,PDF}/`；内容私有资源随所属目录保留在 `assets/` 子目录。
- 公开 URL 必须使用：`/专栏/...` 与 `/其他/{恋爱必修课,文章,极客时间,PDF}/...`。
- `/content/...` 是物理路径，不作为主公开 URL；Nginx 应拒绝直接暴露。
- `static/`、`img/`、`live-2d/` 是全站公共资源目录，不迁入 `content/`。
- 文章内私有图片继续使用同级相对路径 `assets/...`，整体迁移时必须保持页面和同级 `assets/` 的相对关系。

### 4. Validation & Error Matrix

- 源目录和目标目录同时存在 -> 迁移脚本必须报错并拒绝覆盖。
- 请求路径包含隐藏路径、`..` 或直接访问 `/content/...` -> Nginx deny 或返回 404/403。
- 旧公开 URL 对应的 `content/` 文件不存在 -> 返回 404，不回退到敏感目录。
- 修改 Nginx 配置但未在目标环境通过容器或 `nginx -t` 验证 -> 结果必须标注“未验证”。

### 5. Good/Base/Bad Cases

- Good：`/专栏/从 0 开始学架构/index.html` 公开访问，Nginx 内部读取 `content/专栏/从 0 开始学架构/index.html`。
- Base：`/static/index.js` 仍直接读取根目录 `static/index.js`。
- Bad：把站内链接批量改成 `/content/专栏/...`，导致阅读进度和外部链接分裂。

### 6. Tests Required

- `python utils/migrate_content_root.py --dry-run`：断言只处理白名单内容目录，已迁移时只提示跳过。
- `python -m py_compile utils/*.py`：断言离线工具脚本语法可加载。
- `docker-compose -f docker-compose.yaml config`：断言 api 与带 `tools` profile 的 content-sync 编排语法有效。
- 抽样访问迁移后 URL：断言 `/其他/文章/index.html`、`/专栏/.../index.html`、`/其他/PDF/index.html` 可访问，旧分类路径不被映射。
- 静态资源抽样：断言 `/static/index.js`、`/img/github.svg`、`/live-2d/js/live2d.js` 未被 content 映射破坏。

### 7. Wrong vs Correct

#### Wrong

```nginx
location / {
    try_files $uri $uri/ =404;
}
```

迁移后旧 URL 只查根目录，`/专栏/...` 会 404。

#### Correct

```nginx
location / {
    try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404;
}
```

公开 URL 不变，物理文件从 `content/` 读取。

---

## 避免的做法

- 不要把本项目误认为 Java 后端；仓库没有 Maven、Spring Boot、Mapper、XML SQL 或 Java service 层。
- 不要重新引入 Flask/Gunicorn 作为生产 Web 入口；生产静态服务由宿主机 Nginx 承担，业务 API 由 Go 容器承担。
- 不要批量重命名中文归档目录或 `*.md.html` 文件；这些路径已经被静态页面、阅读进度和脚本引用。
- 不要运行会大量联网抓取、提交或推送的脚本作为轻量验证；例如 `utils/03_patch_zhuanlan.sh` 内含 `git commit` 和 `git push -f origin main`，自动化执行前必须人工审查并获得确认。
