# 数据与持久化规范

> 本项目当前没有数据库、ORM、迁移工具或服务端业务数据模型。所谓“数据”主要是仓库中的静态文件、归档 HTML/PDF、资源文件，以及少量脚本状态文件。本规范记录现状，避免后续任务凭空引入数据库方案。

---

## 总览

当前持久化方式是文件系统和 Git 仓库：

- 页面内容：`index.html`、`专栏/**/*.md.html`、`文章/**/*.md.html`、`极客时间/**/*.md.html`、`恋爱必修课/**/*.md.html`。
- PDF 内容：`PDF/*.pdf` 和 `PDF/index.html`。
- 静态资源：`static/`、`assets/`、`img/`、`live-2d/`。
- 脚本批处理状态：`utils/task.json`。
- 部署配置：`Dockerfile`、`docker-compose.yml`、`learn-liangliang.conf`。

仓库中没有 `models/`、`migrations/`、数据库连接配置、SQL 文件或表结构定义。`requirements.txt` 只包含 `flask`、`gunicorn`、`requests`、`beautifulsoup4`、`gevent`，没有 SQLAlchemy、Alembic、Django ORM、PyMySQL、psycopg 等数据库依赖。

---

## 查询与读取模式

### Flask 静态文件读取

`server_flask.py` 通过文件系统判断路径并返回静态文件：

```python
file_path = os.path.join(BASE_DIR, filename)

if os.path.isfile(file_path):
    return send_from_directory(BASE_DIR, filename)
elif os.path.isdir(file_path):
    index_path = os.path.join(file_path, 'index.html')
    if os.path.isfile(index_path):
        return send_from_directory(file_path, 'index.html')
```

现有服务不查询数据库；访问 URL 与仓库文件路径直接对应。路径处理时已有基础防护：URL 解码后拒绝以 `.` 开头或包含 `..` 的路径。

### 工具脚本读取 HTML/PDF 列表

工具脚本通常先读取本地 `index.html`，再用 BeautifulSoup 解析链接：

```python
# utils/04_patch_pdf.py
with open(index_path, "r", encoding="utf-8") as f:
    html = f.read()
soup = BeautifulSoup(html, "html.parser")
```

`utils/03_patch_others.py` 从栏目 `index.html` 中查找 `class="menu-item"` 且以 `.md.html` 结尾的链接，再下载对应 `.md` 内容。

### JSON 状态文件

`utils/gen_task_json.py` 会根据 `专栏/` 下的目录生成 `utils/task.json`：

```python
tasks = [{"name": z, "status": "未完成"} for z in zhuanlans]
json.dump(tasks, f, ensure_ascii=False, indent=2)
```

`utils/03_patch_zhuanlan.sh` 使用 `jq` 读取和更新 `utils/task.json` 中的 `status` 字段。这个 JSON 文件只是批处理状态，不是业务数据库。

---

## 写入模式

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

当前没有数据库迁移。涉及内容结构变更时，实际“迁移”是对仓库文件进行批量转换或修补，例如：

- `utils/03_patch_donation_md_links.py` 下载并修补 `assets/捐赠.md.html` 及其资源路径。
- `utils/03_patch_others.py` 将远端 `.md` 内容保存为本地 `.md.html`，并把 HTML 中的 `.md` 链接替换为 `.md.html`。
- `utils/04_patch_pdf.py` 下载 `PDF/index.html` 中引用的 PDF。

若未来需要批量改写归档文件，应新增或修改 `utils/` 下的脚本，并在运行前说明会影响哪些目录。不要为这类静态文件迁移引入 Alembic、Flyway、Liquibase 或 Java Mapper/XML 体系。

---

## 命名约定

- 文件和目录名称保留原内容语义，允许中文、空格和标点，例如 `专栏/22 讲通关 Go 语言-完/`。
- 归档 Markdown 转 HTML 文件保留 `.md.html` 双后缀；现有脚本依赖这个约定。
- `utils/task.json` 中状态值当前是中文：`未完成`、`已完成`。
- 代理账号列表当前在 `utils/proxy_pool.py` 的 `proxy_accounts` 中以元组保存。

---

## 常见风险与禁止事项

- 不要为当前需求新增数据库或 ORM；现有站点没有数据库运行时依赖。
- 不要把 `utils/task.json` 当成服务端 API 数据源；它只是抓取脚本的本地批处理状态。
- 不要在轻量检查中运行会联网下载大量内容的脚本，例如 `utils/03_patch_others.py`、`utils/04_patch_pdf.py`，除非任务明确要求并已确认影响范围。
- 不要在代码或文档中新增真实代理账号、密钥或访问令牌；`utils/proxy_pool.py` 当前已包含代理账号样式信息，后续应避免扩大泄露范围。
- 不要把中文路径批量转义或重命名为英文路径；Nginx、HTML 链接和 Flask 文件查找都依赖当前文件布局。
