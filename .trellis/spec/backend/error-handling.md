# 错误处理规范

> 本项目生产运行时由宿主机 Nginx 静态站点、Docker Compose 中的 Go API 和宿主机 PostgreSQL 组成；Python 仅作为离线内容维护脚本存在。错误处理分为 Go API JSON 错误、宿主机 Nginx 拒绝/404 和代理错误、以及 `utils/` 脚本异常处理。

---

## 总览

真实错误处理分为三类：

1. Go API 请求错误：通过 `backend/internal/response` 返回统一 JSON 错误响应。
2. 宿主机 Nginx 静态站点错误/拒绝：隐藏路径、内部目录、调试文件和缺失文件由 Nginx 拒绝或返回 404；API 上游 `127.0.0.1:8081` 不可用时返回代理错误。
3. `utils/*.py` 脚本错误：捕获异常，打印失败原因，尽量继续处理后续文件。

生产 Web 运行时不再包含 Flask/Gunicorn，也不存在 Flask `abort()` 作为请求错误处理方式。

---

## Go API 错误响应

统一 JSON 错误响应形态：

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "请先登录"
  }
}
```

修改或新增 API 时应复用 `backend/internal/response`，不要在不同处理器中手写不一致的错误结构。

常见错误语义：

- `UNAUTHORIZED`：未登录、会话不存在、会话过期或已撤销。
- `INVALID_REQUEST`：请求体解析失败、字段为空、`articlePath` 不合法、进度百分比越界等。
- `INTERNAL_ERROR`：数据库连接、迁移、查询或其他服务端内部错误。

认证和阅读进度接口必须保持以下边界：

- 登录失败不能泄露“用户名存在但密码错误”等细节。
- `articlePath` 为空、含 NUL、路径穿越、过长或不是文章路径时应返回 `INVALID_REQUEST`。
- `/content/...` 输入应在后端规范化为公开旧 URL 后再查询或保存，不能形成两条记录。

---

## Nginx 静态站错误处理

`deploy/nginx/default.conf` 中存在显式拒绝：

```nginx
location ~ /\. {
    deny all;
}

location ~* ^/(backend|deploy|utils|ssl|content)(/|$) {
    deny all;
}

location ~* \.(?:js\.map)$ {
    return 404;
}
```

普通静态请求通过 `try_files` 查找根目录或 `content/` 内部路径：

```nginx
location / {
    try_files $uri $uri/ /content$uri /content$uri/index.html /content$uri/ =404;
}
```

要求：

- 缺失文件必须返回 404。
- `/content/...` 不作为主公开 URL 暴露。
- `backend/`、`deploy/`、`utils/` 等源码和运维目录不能被浏览器直接访问。
- 中文路径、URL 编码路径和目录 `index.html` 访问必须保持可用。
- `/专栏/**`、`/其他/**`（包括 PDF 和内容私有资源）与 `/reading-history.html` 必须在静态文件读取前执行 `auth_request /internal/authenticate`；受保护响应必须为 `Cache-Control: private, no-store`，不可被公开资源长缓存规则绕过。
- 内部认证端点仅接受 Nginx 子请求：有效会话返回无正文 `204`，缺失或失效会话返回无正文 `401`，数据库异常返回无正文 `5xx` 并拒绝访问。仅 Nginx 将 `401` 重定向到登录页；`next` 只能来自 `$uri` pathname，不能使用含查询参数的 `$request_uri`。
- `/api/reading-progress/**` 的未认证请求仍按 API 契约返回 JSON `401`，不由 Nginx 重定向。

---

## 宿主机 Nginx API 代理错误处理

宿主机 Nginx 将 API 请求和内部认证请求反代至回环地址：

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:8081;
}

location = /internal/authenticate {
    internal;
    proxy_pass http://127.0.0.1:8081/internal/authenticate;
}
```

若 API 容器不可用，Nginx 返回代理层错误。排查时检查 `docker-compose -f docker-compose.yaml ps api`、`docker-compose -f docker-compose.yaml logs -f api`、宿主机 Nginx 错误日志，以及 API 是否只监听 `127.0.0.1:8081`。

---

## 脚本错误处理模式

### 网络下载异常

抓取脚本通常对单个资源包裹 `try/except`，失败后打印原因并继续：

```python
# utils/01_download_index.py
try:
    css_content = requests.get(css_url, headers=headers).text
    with open(css_path, "w", encoding="utf-8") as f:
        f.write(css_content)
    print(f"已保存CSS: {css_path}")
except Exception as e:
    print(f"下载CSS失败: {css_url}，原因: {e}")
```

`utils/03_patch_others.py` 和 `utils/04_patch_pdf.py` 对 429 限流有重试和等待：

```python
elif resp.status_code == 429:
    wait = wait_times[retry] if retry < len(wait_times) else 16
    print(f"    被限流，{wait}s后重试（第{retry+1}次）...")
    time.sleep(wait)
    retry += 1
```

### HTTP 状态码处理

脚本通常只把 200 当成功，非 200 打印状态码：

```python
if resp.status_code == 200:
    ...
else:
    print(f"抓取失败：{menu_id}，状态码：{resp.status_code}")
```

### 跳过已存在文件

下载类脚本会跳过已存在文件，避免重复写入：

```python
# utils/04_patch_pdf.py
if os.path.exists(save_path):
    print(f"    已存在，跳过: {filename}")
    continue
```

---

## 修改错误处理时的要求

- Go API 应复用统一响应封装；不要让不同接口返回不同 JSON 错误形态。
- 静态站点必须保留内部目录拒绝、缺失文件 404 和旧公开 URL 映射。
- 对批量抓取脚本，单个文件失败不应中断整批任务，除非失败会导致后续结果不可信。
- 继续使用中文错误提示，符合现有脚本输出风格，例如“下载失败”“抓取异常”“被限流”。
- 网络请求应设置合理 timeout；现有 `utils/03_patch_others.py` 使用 `timeout=20`，`utils/04_patch_pdf.py` 使用 `timeout=30`，新脚本不要无限等待。
- 对 429、网络异常等可恢复错误，可延续 `wait_times` + `max_retry` 的重试方式。

---

## 常见错误与注意事项

- 不要重新引入 Flask 静态文件服务来处理 403/404；生产静态错误由 Nginx 承担。
- 不要给静态 HTML 请求硬套 API JSON 错误格式；JSON 错误只属于 `/api/*`。
- 不要在异常日志里打印新的密钥、代理密码或完整敏感配置。
- 不要在轻量验证时触发大规模下载脚本来“测试错误处理”；这些脚本会访问外网并写入大量文件。
- 排查访问问题时，先确认 `api` 容器状态、宿主机 PostgreSQL 连接、`127.0.0.1:8081` 监听和宿主机 Nginx 上游是否与 `docker-compose.yaml` 一致。
