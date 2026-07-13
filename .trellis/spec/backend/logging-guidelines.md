# 日志规范

> 本项目生产日志主要来自宿主机 Nginx、Go API 容器 stdout/stderr 和宿主机 PostgreSQL；离线内容维护日志来自 `utils/` 脚本中的 `print()` 中文进度输出。生产 Web 运行时不再包含 Gunicorn/Flask 日志。

---

## 总览

当前日志来源：

- Nginx：宿主机 Nginx 输出静态站点访问、认证和反向代理错误日志。
- Go API：`api` 容器输出启动、迁移、管理员预置账号、请求处理和内部错误相关日志。
- PostgreSQL：宿主机 PostgreSQL 输出数据库启动、连接和错误日志。
- Python 工具脚本：使用 `print()` 输出进度、成功、失败、异常、限流和保存率。
- Shell 脚本：`utils/03_patch_zhuanlan.sh` 使用命令输出和 `echo`；该脚本包含提交/推送命令，自动化执行前必须人工确认。

排查生产问题时优先使用：

```bash
docker compose -f docker-compose.yaml ps
docker compose -f docker-compose.yaml logs -f api
sudo journalctl -u nginx -f
```

---

## 日志级别

项目中没有统一跨语言日志级别枚举。按当前实践，可以用输出内容区分语义：

- 进度类：`进度: 1/10`、`开始处理专栏`、`共发现 X 个文件`。
- 成功类：`已保存`、`下载成功`、管理员账号初始化成功、数据库迁移完成。
- 跳过类：`已存在，跳过`。
- 可重试问题：`被限流，Xs后重试`、`下载异常`。
- 不可恢复或当前项失败：`下载失败，状态码`、`抓取失败`、`最多重试X次，放弃`、API 内部错误。

真实示例：

```python
# utils/03_patch_others.py
print(f"进度: {idx}/{total} (当前代理: {username})")
print(f"    已存在，跳过: {filename}")
print(f"    下载异常: {e}，{wait}s后重试（第{retry+1}次）...")
print(f"保存率: {success}/{total} = {success/total:.2%}")
```

---

## 容器日志

### 宿主机 Nginx 反向代理

宿主机 Nginx 将 `/api/*` 和 `/internal/authenticate` 代理至 `127.0.0.1:8080`。如果出现 502/503，应检查 `api` 容器是否运行、API 是否监听 `127.0.0.1:8080`，以及宿主机 Nginx 错误日志。

### Nginx 静态站点

`deploy/nginx/default.conf` 负责静态路径映射、内部目录拒绝和缓存。排查静态资源 403/404 时应检查：

- 请求是否误用了 `/content/...` 物理路径。
- `try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404` 是否仍存在。
- `backend|deploy|utils|ssl|content` 拒绝规则是否命中。

### Go API

Go API 日志应输出到 stdout/stderr，便于 Docker 收集。新增日志时应避免打印：

- 明文密码。
- 会话 token 原文。
- 数据库连接串中的密码。
- Cookie 完整值。

---

## 脚本日志格式

现有脚本日志特点：

- 使用简体中文，便于直接人工观察。
- 包含当前处理对象，例如 URL、文件名、栏目名或保存路径。
- 批量任务包含总数、进度和最终保存率。
- 下载失败时输出 HTTP 状态码或异常文本。

新增或修改脚本时应延续这种风格：

```python
print(f"共发现 {total} 个文件")
print(f"进度: {idx}/{total} (当前代理: {username})")
print(f"    正在处理: {filename}")
print(f"    已保存: {filename}")
```

如果脚本会跳过文件，应明确打印跳过原因：

```python
if os.path.exists(save_path):
    print(f"    已存在，跳过: {filename}")
    continue
```

---

## 应记录的内容

- 批量处理总数：例如发现多少菜单项、多少 PDF、多少 `.md.html` 链接。
- 当前进度：例如 `idx/total` 和当前文件名。
- 写入位置：例如 `已保存：{save_path}`、`已下载静态资源: {resource_path}`。
- 失败原因：HTTP 状态码、异常信息、被限流后的等待时间。
- 跳过原因：文件已存在、链接为空、没有找到目标内容等。
- 部署检查：`docker compose -f docker-compose.yaml config` 输出、容器状态、关键容器日志。

---

## 不应记录的内容

- 不要新增输出代理密码、密钥、证书私钥内容或其他敏感信息。
- `utils/proxy_pool.py` 当前 `test_all_proxies()` 会打印完整代理 URL，其中包含账号和密码；后续修改时应避免扩大这种输出范围。
- 不要把大量 HTML 正文、PDF 二进制内容或完整响应体直接打印到日志。
- 不要在前端 `static/index.js` 或 `static/reading-progress.js` 中新增过多 `console.log` 输出；新增功能应谨慎，避免污染浏览器控制台。
- 不要在 Go API 日志中打印明文密码、session token、Cookie 或带密码的数据库连接串。

---

## 何时引入更正式的日志机制

当前简单脚本使用 `print()` 与现状一致。只有在以下情况明确出现时，才考虑把某个脚本迁移到 Python `logging`：

- 需要区分 debug/info/warning/error 并控制输出级别。
- 脚本长期运行，日志需要写入文件并轮转。
- 同一脚本被自动化定时任务调用，需要机器可解析的日志格式。

Go API 如后续需要审计、追踪请求耗时或排查线上问题，可在 `backend/` 内引入结构化日志中间件，但应先明确字段、敏感信息脱敏规则和输出量控制，不要影响静态站点访问性能。
