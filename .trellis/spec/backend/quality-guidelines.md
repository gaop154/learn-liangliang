# 质量规范

> 本项目是静态归档站点，辅以 Flask 静态服务和 Python 抓取脚本。质量重点不是后端业务分层，而是路径安全、静态链接可用、批量脚本可控、部署配置与实际端口一致、不要误伤大量归档内容。

---

## 总览

当前仓库没有发现 pytest、ruff、black、mypy、ESLint、pnpm/npm 构建脚本或 CI 配置。轻量验证应优先使用不会联网、不会批量改写内容的检查方式，例如：

- Python 语法检查：`python -m py_compile server_flask.py utils/*.py`。
- 占位符检查：确认 `.trellis/spec/backend/*.md` 中不再有 Trellis 初始英文占位内容。
- 针对改动文件做人工或脚本化文本检查，例如确认真实路径示例存在。
- 如修改 Nginx 配置，在目标服务器上执行 `nginx -t` 后再 reload；本地 Windows 环境通常无法直接验证 Nginx。

---

## 必须保持的质量要求

### 1. Flask 路径安全

`server_flask.py` 当前对 URL 解码后拒绝隐藏路径和目录穿越：

```python
filename = urllib.parse.unquote(filename)

if filename.startswith('.') or '..' in filename:
    abort(403)
```

修改静态服务时必须保留或增强这类防护，并确保中文路径、URL 编码路径和目录 `index.html` 访问仍可用。

### 2. 不破坏静态归档路径

HTML 和脚本依赖当前路径形态：

- 首页：`index.html`。
- 分类目录：`专栏/`、`文章/`、`极客时间/`、`恋爱必修课/`、`PDF/`。
- 内容文件：`*.md.html`。
- 静态资源：`static/`、`assets/`、`img/`、`live-2d/`。

不要随意批量重命名中文目录、空格路径或 `.md.html` 后缀。

### 3. 批量脚本要可控

`utils/03_patch_others.py`、`utils/04_patch_pdf.py` 等脚本会访问外网并写入大量文件。修改这类脚本时应保持：

- 已存在文件跳过逻辑。
- 单项失败后继续处理后续项目。
- 429 限流重试和等待。
- 输出总数、进度、失败原因和保存率。

### 4. 部署端口一致性

- `Dockerfile` 暴露并绑定 `60000`。
- `docker-compose.yml` 映射 `60000:60000`。
- `learn-liangliang.conf` 反代到 `http://127.0.0.1:60000`。
- `server_flask.py` 的本地 `app.run()` 使用 `60005`，生产注释使用 `60000`。

修改端口时必须同步检查上述文件，不要只改其中一个。

### 5. 简体中文文档与提示

仓库 README、脚本输出和本规范均使用简体中文。新增文档、脚本提示和说明应使用简体中文。

---

## 禁止或高风险模式

- 禁止把本仓库当作 Java/Spring 后端来新增 Mapper、Service、Controller、数据库迁移等结构。
- 禁止在未确认影响范围时运行 `utils/03_patch_zhuanlan.sh`；该脚本包含 `git commit` 和 `git push -f origin main`。
- 禁止在轻量验证中运行会大规模下载或改写内容的脚本。
- 禁止将代理密码、证书私钥、访问令牌等敏感信息写入新增日志或文档示例。
- 禁止删除 `learn-liangliang.conf` 中 `.git` 访问拦截和 `.js.map` 屏蔽规则，除非有明确替代方案。
- 禁止为了“整理”而批量格式化几万份归档 HTML；这会制造巨大 diff 且容易破坏内容。
- 禁止无需求引入前端构建链路；当前站点直接使用静态 HTML/CSS/JS。

---

## 建议验证命令

根据改动范围选择最轻量的检查：

```bash
# Python 语法检查，不访问网络、不写业务内容
python -m py_compile server_flask.py utils/01_download_index.py utils/02_download_menu_index.py utils/03_patch_donation_md_links.py utils/03_patch_others.py utils/04_patch_pdf.py utils/gen_task_json.py utils/proxy_pool.py
```

```bash
# 检查 Trellis 后端规范是否仍有初始英文占位符
python - <<'PY'
from pathlib import Path
markers = ['To be ' + 'filled', 'To ' + 'fill']
for p in Path('.trellis/spec/backend').glob('*.md'):
    text = p.read_text(encoding='utf-8')
    if any(marker in text for marker in markers):
        print(p)
PY
```

如果修改 Docker 配置：

```bash
docker compose config
```

如果修改 Nginx 配置，应在具备 Nginx 的目标环境执行：

```bash
sudo nginx -t
sudo nginx -s reload
```

---

## 代码评审清单

- 是否符合当前项目形态：静态站点 + Flask 辅助 + Python 脚本 + Docker/Nginx，而不是传统业务后端？
- 是否引用了真实存在的文件路径，例如 `server_flask.py`、`utils/*.py`、`static/index.js`、`Dockerfile`、`docker-compose.yml`、`learn-liangliang.conf`？
- 是否保留中文路径、`.md.html` 后缀和现有静态资源目录？
- 是否避免了不必要的大规模格式化或批量重写归档内容？
- 如果改动了抓取脚本，是否保留 timeout、重试、跳过已存在文件、中文进度输出？
- 如果改动了服务入口，是否保留 403/404 语义和路径穿越防护？
- 如果改动了端口或部署方式，是否同步检查 `Dockerfile`、`docker-compose.yml`、`learn-liangliang.conf`？
- 是否运行了与改动范围匹配的轻量验证？未能验证的部分是否明确说明？
