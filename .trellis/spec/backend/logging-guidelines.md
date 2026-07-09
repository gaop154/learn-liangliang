# 日志规范

> 本项目没有集中式应用日志框架。运行时日志主要来自 Nginx access/error log、Gunicorn/Flask 默认输出，以及 `utils/` 脚本中的 `print()` 中文进度日志。

---

## 总览

当前日志来源：

- Nginx：`learn-liangliang.conf` 明确写入访问日志和错误日志。
- Gunicorn/Flask：`Dockerfile` 通过 Gunicorn 启动 `server_flask:app`，使用 Gunicorn 默认日志行为。
- Python 工具脚本：使用 `print()` 输出进度、成功、失败、异常、限流和保存率。
- Shell 脚本：`restart_nginx.sh`、`utils/03_patch_zhuanlan.sh` 使用命令输出和 `echo`。

当前没有使用 Python `logging` 模块，也没有 JSON 结构化日志、trace id、链路追踪或第三方日志平台配置。

---

## 日志级别

项目中没有显式日志级别枚举。按当前实践，可以用输出内容区分语义：

- 进度类：`进度: 1/10`、`开始处理专栏`、`共发现 X 个文件`。
- 成功类：`已保存`、`下载成功`、`Git 操作完成`。
- 跳过类：`已存在，跳过`。
- 可重试问题：`被限流，Xs后重试`、`下载异常`。
- 不可恢复或当前项失败：`下载失败，状态码`、`抓取失败`、`最多重试X次，放弃`。

真实示例：

```python
# utils/03_patch_others.py
print(f"进度: {idx}/{total} (当前代理: {username})")
print(f"    已存在，跳过: {filename}")
print(f"    下载异常: {e}，{wait}s后重试（第{retry+1}次）...")
print(f"保存率: {success}/{total} = {success/total:.2%}")
```

---

## Nginx 日志

`learn-liangliang.conf` 中配置了固定日志路径：

```nginx
access_log /data/logs/ngx.learn-liangliang.access.log;
error_log /data/logs/ngx.learn-liangliang.error.log;
```

Nginx 同时承担以下运行保护，排查访问问题时应结合 access/error log：

- `limit_req_zone` 和 `limit_req`：按 IP 限速，`rate=5r/s`，`burst=10 nodelay`。
- 静态目录缓存：`assets|img|live-2d|static|PDF` 以及中文内容目录设置 `expires 30d`。
- 兜底代理：`proxy_pass http://127.0.0.1:60000`。

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
- 部署检查：`restart_nginx.sh` 中保留 `nginx -t` 输出，reload 前必须能看到配置测试结果。

---

## 不应记录的内容

- 不要新增输出代理密码、密钥、证书私钥内容或其他敏感信息。
- `utils/proxy_pool.py` 当前 `test_all_proxies()` 会打印完整代理 URL，其中包含账号和密码；后续修改时应避免扩大这种输出范围。
- 不要把大量 HTML 正文、PDF 二进制内容或完整响应体直接打印到日志。
- 不要在前端 `static/index.js` 中新增过多 `console.log` 输出；当前已有 `console.log(path)`、`console.log(cookie)`、`console.log("title=" + title)` 等调试输出，新增功能应谨慎，避免污染浏览器控制台。

---

## 何时引入 logging 模块

当前简单脚本使用 `print()` 与现状一致。只有在以下情况明确出现时，才考虑把某个脚本迁移到 Python `logging`：

- 需要区分 debug/info/warning/error 并控制输出级别。
- 脚本长期运行，日志需要写入文件并轮转。
- 同一脚本被自动化定时任务调用，需要机器可解析的日志格式。

即便引入 `logging`，也应优先局部改造目标脚本，不要为了统一风格一次性重写所有 `utils/*.py`。
