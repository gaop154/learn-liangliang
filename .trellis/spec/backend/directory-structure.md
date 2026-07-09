# 目录结构规范

> 本项目不是传统后端业务系统；真实形态是“静态内容归档站点 + Python/Flask 辅助服务 + Docker/Gunicorn + Nginx 反向代理”。本文件记录当前仓库已经存在的组织方式，避免后续任务误按 Java/Spring 或大型后端分层来改造。

---

## 总览

当前仓库的核心内容是已归档的 HTML、PDF、图片和静态资源。Python 代码主要承担两类职责：

1. `server_flask.py`：用 Flask 按文件系统路径提供静态文件访问，生产环境由 `Dockerfile` 中的 Gunicorn 启动。
2. `utils/*.py`：一次性或批处理抓取、修补归档内容的脚本，直接读写仓库中的 HTML/PDF/assets 文件。

部署入口由 `Dockerfile`、`docker-compose.yml`、`learn-liangliang.conf` 和 `restart_nginx.sh` 组成。Nginx 负责 HTTPS、缓存、限流、静态资源直出，以及把动态兜底请求反代到 Flask/Gunicorn。

---

## 当前目录布局

```text
learn-liangliang/
├── index.html                         # 站点首页，静态 HTML
├── server_flask.py                    # Flask 静态文件服务入口
├── requirements.txt                   # Python 运行/脚本依赖：flask、gunicorn、requests、beautifulsoup4、gevent
├── Dockerfile                         # Python 3.10 slim + Gunicorn/gevent 启动 Flask
├── docker-compose.yml                 # flask-server 服务，映射 60000 端口并挂载当前目录到 /app
├── learn-liangliang.conf              # Nginx HTTPS、缓存、限流、反向代理配置
├── restart_nginx.sh                   # 拷贝 Nginx 配置、nginx -t、reload
├── static/                            # 前端 CSS/JS、highlight、评论区注入脚本等
│   ├── index.css
│   ├── index.js
│   ├── highlight.min.css
│   └── highlight.min.js
├── assets/                            # 捐赠页、站点图片等公共资源
├── img/                               # README 截图、GitHub 图标、分类图片
├── live-2d/                           # Live2D 模型和脚本资源
├── PDF/                               # PDF 资料及 PDF/index.html
├── 专栏/                              # 技术专栏归档，目录下包含 index.html 和 *.md.html
├── 文章/                              # 文章归档
├── 极客时间/                          # 极客时间内容归档
├── 恋爱必修课/                        # 非技术类专栏归档
└── utils/                             # 抓取、修补、批处理脚本
    ├── 01_download_index.py
    ├── 02_download_menu_index.py
    ├── 03_patch_donation_md_links.py
    ├── 03_patch_others.py
    ├── 03_patch_zhuanlan.sh
    ├── 04_patch_pdf.py
    ├── gen_task_json.py
    ├── proxy_pool.py
    └── task.json
```

---

## 模块组织约定

### Flask 服务入口

- 当前只有一个 Flask 服务文件：`server_flask.py`。
- 路由直接定义在文件顶层：
  - `/` 返回仓库根目录的 `index.html`。
  - `/<path:filename>` 对 URL 解码后按文件系统路径查找文件或目录下的 `index.html`。
- 不存在 `routes/`、`controllers/`、`services/`、`models/` 等后端分层目录；不要为小改动引入这类大型框架结构。

真实示例：

```python
# server_flask.py
@app.route('/')
def index():
    index_path = os.path.join(BASE_DIR, 'index.html')
    if os.path.isfile(index_path):
        return send_from_directory(BASE_DIR, 'index.html')
    else:
        abort(404)
```

### 工具脚本

- 抓取和修补脚本集中放在 `utils/`。
- 脚本按处理顺序使用数字前缀命名，例如：
  - `utils/01_download_index.py`：抓取首页并下载 CSS/图片/JS 到 `static/`。
  - `utils/02_download_menu_index.py`：抓取栏目菜单页。
  - `utils/03_patch_others.py`：按 `--column` 下载栏目中的 `.md` 内容并保存为 `.md.html`。
  - `utils/04_patch_pdf.py`：从 `PDF/index.html` 解析 PDF 链接并下载 PDF。
- 批处理 Shell 脚本也放在 `utils/`，例如 `utils/03_patch_zhuanlan.sh` 调用 `03_patch_others.py` 并更新 `utils/task.json`。

真实示例：

```python
# utils/03_patch_others.py
parser = argparse.ArgumentParser(description="批量下载专栏内容")
parser.add_argument("--column", type=str, default="恋爱必修课", help="专栏目录名，默认'恋爱必修课'")
```

### 静态前端资源

- 站点主页面和内容页是静态 HTML，不使用前端构建工具。
- 公共脚本和样式放在 `static/`，例如 `static/index.js` 和 `static/index.css`。
- `static/index.js` 当前承担以下站点增强职责：记录上次阅读路径、侧边栏交互、GitHub 悬浮入口、Live2D 注入、Giscus 评论区注入、页脚修改。
- `img/` 用于站点截图和图标，`assets/` 用于被内容页直接引用的资源，`live-2d/` 用于 Live2D 模型资源。

### 部署配置

- 容器入口写在 `Dockerfile`，真实启动命令为：

```dockerfile
CMD ["gunicorn", "-w", "4", "-k", "gevent", "-b", "0.0.0.0:60000", "server_flask:app"]
```

- `docker-compose.yml` 只定义 `flask-server` 服务，容器名也是 `flask-server`，端口映射为 `60000:60000`，并将仓库根目录挂载到 `/app`。
- `learn-liangliang.conf` 是站点 Nginx 配置，静态目录直接 `root /home/wenxuan/code_ws/learn-liangliang`，兜底 `location /` 反代到 `http://127.0.0.1:60000`。

---

## 命名约定

- 归档内容目录保留中文名称：`专栏/`、`文章/`、`极客时间/`、`恋爱必修课/`，不要为了代码习惯批量改成英文目录。
- 归档文章文件当前使用 `*.md.html` 后缀，例如 `专栏/10x程序员工作法/00 开篇词 程序员解决的问题，大多不是程序问题.md.html`；脚本中也有把 `.md` 替换为 `.md.html` 的逻辑。
- 工具脚本按执行阶段使用数字前缀：`01_`、`02_`、`03_`、`04_`。
- Python 变量和函数使用 snake_case，例如 `safe_filename`、`get_links`、`download_static_resources`。
- 静态文件沿用原站或现有资源命名，例如 `static/highlight.min.js`、`static/email-decode.min.js`、`assets/捐赠.md.html`。

---

## 新增或修改文件时放哪里

- 新的 Flask 路由或静态服务安全修补：优先在 `server_flask.py` 内小范围修改，除非文件明显失控，否则不要拆分成多层包。
- 新的内容抓取/修补流程：放在 `utils/`，按现有数字前缀延续命名，并尽量支持命令行参数，避免硬编码只能处理一个栏目。
- 新的全站前端增强：放在 `static/index.js` 或对应 CSS 文件中；如果是第三方静态库，放在 `static/` 或更明确的子目录。
- 新的站点公共图片：根据用途放到 `img/` 或 `assets/`；内容页局部资源保持在对应内容目录的 `assets/` 子目录。
- 部署相关变更：Docker 改 `Dockerfile`/`docker-compose.yml`，Nginx 改 `learn-liangliang.conf`，不要把运行配置散落到工具脚本中。

---

## 避免的做法

- 不要把本项目误认为 Java 后端；仓库没有 Maven、Spring Boot、Mapper、XML SQL 或 Java service 层。
- 不要在没有明确需求时新增数据库、ORM、迁移目录或复杂分层。
- 不要批量重命名中文归档目录或 `*.md.html` 文件；这些路径已经被静态页面和脚本引用。
- 不要运行会大量联网抓取、提交或推送的脚本作为轻量验证；例如 `utils/03_patch_zhuanlan.sh` 内含 `git commit` 和 `git push -f origin main`，自动化执行前必须人工审查并获得确认。
