# 数据与持久化规范

> 本项目当前有两类持久化：生产阅读进度数据存放在 PostgreSQL；静态归档内容、资源文件和离线脚本状态仍存放在仓库文件系统中。不要再把项目描述为“没有数据库”，也不要把 `utils/task.json` 当成生产 API 数据源。

---

## 总览

当前持久化方式：

- PostgreSQL：用户、会话和阅读进度数据，由 `db` 容器和 `postgres_data` volume 持久化。
- 页面内容：`index.html`、`content/专栏/**/*.md.html`、`content/其他/{恋爱必修课,文章,极客时间}/**/*.md.html`。
- PDF 内容：`content/其他/PDF/*.pdf` 和 `content/其他/PDF/index.html`。
- 静态资源：`static/`、各课程或分类目录内的 `assets/`、`img/`、`live-2d/`。
- 脚本批处理状态：`utils/task.json`。
- 部署配置：`docker-compose.yml`、`deploy/caddy/Caddyfile`、`deploy/nginx/default.conf`、`backend/Dockerfile`。
- Python 工具脚本依赖：`requirements.txt` 仅包含 `requests`、`beautifulsoup4` 等离线脚本依赖。

生产 Web 运行时不再使用 Flask/Gunicorn，也不依赖根目录 Python `Dockerfile`。

---

## PostgreSQL 数据模型

### users

保存管理员预置账号和后续可扩展用户账号。密码只保存哈希，不保存明文。

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(64),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### sessions

保存会话 token 的哈希、过期时间和撤销时间。Cookie 中的 token 原文不能落库。

```sql
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### reading_progress

按 `(user_id, article_path)` 唯一保存单篇文章阅读进度。

```sql
CREATE TABLE reading_progress (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_path TEXT NOT NULL,
    article_title TEXT NOT NULL,
    progress_percent INT NOT NULL DEFAULT 0,
    scroll_y INT NOT NULL DEFAULT 0,
    finished BOOLEAN NOT NULL DEFAULT FALSE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, article_path),
    CHECK (progress_percent >= 0 AND progress_percent <= 100),
    CHECK (scroll_y >= 0)
);
```

---

## 查询与读取模式

### Go API 读写 PostgreSQL

Go API 通过 `pgx` 和 `sqlc` 访问数据库：

- SQL 查询定义放在 `backend/internal/db/queries/`。
- sqlc 生成代码放在 `backend/internal/db/sqlc/`。
- 处理器不应手写复杂 SQL 或绕过已有 DB 封装。
- 跨接口共享的数据格式应由 Go 类型、SQL 约束和前端脚本共同遵守，不要各自定义一套字段名。

阅读进度按 `(user_id, article_path)` upsert。`article_path` 必须是公开 canonical URL，例如 `/其他/文章/A.md.html`，不能保存 `/content/其他/文章/A.md.html`。

### Nginx 静态文件读取

`deploy/nginx/default.conf` 通过 `try_files` 将公开旧 URL 映射到物理 `content/` 路径：

```nginx
location / {
    try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404;
}
```

`/content/...` 是物理路径，不作为主公开 URL；配置中必须拒绝直接暴露 `content/`、`backend/`、`deploy/`、`utils/` 等内部目录。

### 工具脚本读取 HTML/PDF 列表

工具脚本通常先读取本地 `index.html` 或 `content/<分类>/index.html`，再用 BeautifulSoup 解析链接：

```python
# utils/04_patch_pdf.py
with open(index_path, "r", encoding="utf-8") as f:
    html = f.read()
soup = BeautifulSoup(html, "html.parser")
```

`utils/03_patch_others.py` 从栏目 `index.html` 中查找 `class="menu-item"` 且以 `.md.html` 结尾的链接，再下载对应 `.md` 内容。

### JSON 状态文件

`utils/gen_task_json.py` 会根据内容目录生成 `utils/task.json`：

```python
tasks = [{"name": z, "status": "未完成"} for z in zhuanlans]
json.dump(tasks, f, ensure_ascii=False, indent=2)
```

`utils/03_patch_zhuanlan.sh` 使用 `jq` 读取和更新 `utils/task.json` 中的 `status` 字段。这个 JSON 文件只是批处理状态，不是生产数据库或 API 数据源。

---

## 写入模式

### 数据库写入

- 用户密码必须先哈希再写入 `users.password_hash`。
- 会话只保存 token hash，不保存 token 原文。
- 阅读进度保存使用 upsert，避免同一用户同一文章产生重复记录。
- `last_read_at`、`updated_at` 应随更新刷新。
- 数据库结构变更必须通过 `backend/migrations/` 管理，并提供可回滚或可解释的 down 脚本。

### 内容抓取写入

脚本直接写入目标文件，通常显式使用 UTF-8：

```python
# utils/01_download_index.py
with open(save_path, "w", encoding="utf-8") as f:
    f.write(str(soup))
```

二进制资源使用 `wb`：

```python
# utils/04_patch_pdf.py
with open(save_path, "wb") as f:
    f.write(resp.content)
```

### 目录创建

脚本在写文件前会创建目录：

```python
os.makedirs(css_dir, exist_ok=True)
os.makedirs(os.path.dirname(local_path), exist_ok=True)
```

新增脚本应延续这种模式，避免假设目录一定存在。

---

## 迁移规范

### 数据库迁移

- 迁移脚本放在 `backend/migrations/`。
- 新增表、索引、约束时必须考虑现有数据兼容性。
- 与阅读进度相关的唯一键、范围检查和索引不能只在前端校验。
- PostgreSQL 数据通过 Docker volume 持久化；部署文档必须提醒备份 `postgres_data` 或使用 `pg_dump`。

### 内容文件迁移

涉及内容结构变更时，实际“迁移”是对仓库文件进行批量转换或修补，例如：

- `utils/03_patch_others.py` 将远端 `.md` 内容保存为本地 `.md.html`，并把 HTML 中的 `.md` 链接替换为 `.md.html`。
- `utils/04_patch_pdf.py` 下载 `content/其他/PDF/index.html` 中引用的 PDF。
- `utils/migrate_content_root.py` 将旧根目录内容迁移到 `content/` 物理根。
- `utils/05_migrate_other_content.py` 将三类非课程文章和 PDF 迁移到 `content/其他/`，改写绝对站内链接、重建导航并清理捐赠功能。

若未来需要批量改写归档文件，应新增或修改 `utils/` 下的脚本，并在运行前说明会影响哪些目录。不要为内容文件迁移引入 Java Mapper/XML 体系。

---

## 命名约定

- 文件和目录名称保留原内容语义，允许中文、空格和标点，例如 `content/专栏/22 讲通关 Go 语言-完/`。
- 归档 Markdown 转 HTML 文件保留 `.md.html` 双后缀；现有脚本依赖这个约定。
- 阅读进度 `article_path` 使用公开旧 URL，例如 `/专栏/xxx/001.md.html`。
- `utils/task.json` 中状态值当前是中文：`未完成`、`已完成`。
- 代理账号列表当前在 `utils/proxy_pool.py` 的 `proxy_accounts` 中以元组保存。

---

## Scenario: 阅读进度 articlePath 使用公开 canonical URL

### 1. Scope / Trigger

- Trigger：当内容物理路径迁移到 `content/`、修改阅读进度前端/后端，或调整阅读历史链接时，必须保证 `articlePath` 不使用物理 `/content/...` 路径。
- Scope：阅读进度数据库键、查询参数和前端历史链接都使用公开旧 URL 作为 canonical path。

### 2. Signatures

- 前端保存字段：`articlePath: string`
- 查询接口：`GET /api/reading-progress?articlePath=<path>`
- 保存接口：`PUT /api/reading-progress`，请求体包含 `articlePath`
- 后端规范化函数：`canonicalizeArticlePath(raw string) (string, error)`

### 3. Contracts

- `/content/其他/文章/Java.md.html` 必须规范化为 `/其他/文章/Java.md.html`。
- `/content/专栏/x/index.html` 必须规范化为 `/专栏/x/index.html`。
- `articlePath` 必须以 `/` 开头，不能是 `/`，不能包含路径穿越段 `..`，长度不超过 1024。
- 仅 canonical 后以 `.md.html` 结尾的路径可读写单篇进度；目录、分类页和静态资源路径必须被 API 拒绝。
- 阅读历史页生成链接时同样输出公开 canonical URL，不输出 `/content/...`。

### 4. Validation & Error Matrix

- `articlePath` 为空 -> `INVALID_REQUEST` / `articlePath 不能为空`。
- `articlePath` 含 NUL、路径穿越或长度超过限制 -> `INVALID_REQUEST` / `articlePath 不合法`。
- `articlePath=/content/...` -> 先规范化为旧公开路径，再保存或查询。
- `articlePath=/content` 或 `/` -> 不作为文章路径保存。

### 5. Good/Base/Bad Cases

- Good：用户访问 `/其他/文章/A.md.html`，数据库保存 `/其他/文章/A.md.html`。
- Base：手工访问 `/content/其他/文章/A.md.html`，数据库仍保存 `/其他/文章/A.md.html`。
- Bad：同一篇文章分别保存 `/content/其他/文章/A.md.html` 和 `/其他/文章/A.md.html` 两条记录。

### 6. Tests Required

- `node --check static/reading-progress.js`：断言前端脚本语法正确。
- `cd backend && go test ./...`：断言 Go 后端可编译。
- 手动或自动检查保存请求 payload：访问旧 URL 时 `articlePath` 是旧公开路径。
- 手动访问 `/content/...` 时，保存和阅读历史链接仍归一到旧公开路径。

### 7. Course Progress Batch Contracts

- `POST /api/reading-progress/batch` 接收 1 至 500 条、去重后的 `.md.html` canonical `articlePaths`，只返回当前用户已有记录；前端以请求集合而非数据库前缀统计课程进度。
- `POST /api/reading-progress/course-resumes` 接收 1 至 500 条、去重后的 `/专栏/<单段课程名>` canonical `coursePaths`，每门课程按 `last_read_at DESC` 返回最近的 `.md.html` 文章记录。
- 已登录用户进入 `/专栏/<课程名>/` 时，前端必须先用单元素 `course-resumes` 请求检查最近文章；返回同课程的合法 `.md.html` 时立即 `location.replace`，仅无记录时再请求 `batch` 并渲染目录进度。`/专栏/` 上的点击拦截仅可作为优化，不能替代目录加载时的检查。
- 课程查询必须使用参数化集合和目录边界匹配，禁止将课程名拼接到 `LIKE` 模式，避免 `%`、`_` 扩大匹配范围。
- `finished=true` 仅允许 `progressPercent=100`；95 至 99 的文章仍是未完成。

### 8. Wrong vs Correct

#### Wrong

```js
function getArticlePath() {
    return window.location.pathname || '/';
}
```

如果用户或内部链接访问 `/content/...`，阅读记录会和旧公开 URL 分裂。

#### Correct

```js
function getArticlePath() {
    return canonicalArticlePath(window.location.pathname || '/');
}
```

保存和查询都使用稳定公开 URL。

---

## 常见风险与禁止事项

- 不要重新引入 Flask/Gunicorn 或 Python Web 运行时；Python 只保留为离线内容维护工具。
- 不要把 `utils/task.json` 当成服务端 API 数据源；它只是抓取脚本的本地批处理状态。
- 不要在轻量检查中运行会联网下载大量内容的脚本，例如 `utils/03_patch_others.py`、`utils/04_patch_pdf.py`，除非任务明确要求并已确认影响范围。
- 不要在代码或文档中新增真实代理账号、密钥或访问令牌；`utils/proxy_pool.py` 当前已包含代理账号样式信息，后续应避免扩大泄露范围。
- 不要把中文路径批量转义或重命名为英文路径；Nginx、HTML 链接、阅读进度和工具脚本都依赖当前文件布局。
- 不要删除 PostgreSQL volume 或把生产数据库改成容器内临时文件；阅读进度必须跨容器重建保留。
