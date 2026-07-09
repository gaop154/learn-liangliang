# 后端开发规范

> 本目录记录本项目“后端相关”真实约定：Flask 静态文件服务、Python 抓取/修补脚本、Docker/Gunicorn 运行方式、Nginx 反向代理与缓存配置。项目主体是静态内容归档站点，不是 Java/Spring 或数据库驱动的后端系统。

---

## 总览

后端相关入口与配置：

- `server_flask.py`：Flask 静态文件服务。
- `utils/*.py`：抓取首页、栏目、文章、PDF 和修补链接的脚本。
- `Dockerfile`：使用 Python 3.10 slim，安装 `requirements.txt`，通过 Gunicorn/gevent 启动 `server_flask:app`。
- `docker-compose.yml`：定义 `flask-server` 服务，映射 `60000:60000`。
- `learn-liangliang.conf`：Nginx HTTPS、缓存、限流、静态资源直出和反代配置。
- `static/index.js`：站点前端增强脚本，包括阅读进度、侧边栏、GitHub 入口、Live2D、Giscus 评论区等。

---

## 规范索引

| 指南 | 内容 | 状态 |
|------|------|------|
| [目录结构](./directory-structure.md) | 真实目录布局、Flask/脚本/静态资源/部署配置放置规则 | 已填充 |
| [数据与持久化规范](./database-guidelines.md) | 当前无数据库；记录文件系统、HTML/PDF、JSON 状态文件的真实使用方式 | 已填充 |
| [错误处理](./error-handling.md) | Flask abort、脚本 try/except、Nginx 拒绝规则与限流响应 | 已填充 |
| [质量规范](./quality-guidelines.md) | 路径安全、批量脚本风险、轻量验证、代码评审清单 | 已填充 |
| [日志规范](./logging-guidelines.md) | Nginx 日志、Gunicorn 默认日志、脚本 print 输出风格 | 已填充 |

---

## 开发前检查

- 确认任务是否真的涉及后端相关文件；很多页面内容只是静态 HTML 或资源文件。
- 如果涉及 Flask 服务，先阅读 `server_flask.py` 的路径解码、403 和 404 处理。
- 如果涉及抓取或批量修补，先阅读对应的 `utils/*.py`，确认是否会联网、是否会写入大量文件。
- 如果涉及部署，必须同时检查 `Dockerfile`、`docker-compose.yml`、`learn-liangliang.conf` 的端口和路径是否一致。
- 不要引入 Java/Spring、数据库、ORM 或复杂分层，除非任务明确要求改变项目架构。

---

## 质量检查

- Python 文件改动后优先运行 `python -m py_compile server_flask.py utils/*.py` 或列出具体脚本进行语法检查。
- 规范文档改动后检查 `.trellis/spec/backend/*.md` 不再包含 Trellis 初始英文占位符。
- Nginx 配置改动需要在目标环境运行 `sudo nginx -t` 后再 reload。
- 避免用会联网下载或批量改写内容的脚本作为普通验证命令。

---

**文档语言**：本项目规范文档使用简体中文。
