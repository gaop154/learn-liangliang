# 错误处理规范

> 本项目的后端错误处理很轻量：Flask 服务使用 `abort()` 返回 HTTP 错误；抓取/修补脚本用 `try/except` 捕获网络和文件处理异常并打印中文提示。当前没有统一异常类、JSON API 错误响应或全局错误处理中间件。

---

## 总览

真实错误处理分为三类：

1. `server_flask.py` 的请求错误：返回 Flask 默认的 403/404 HTML 错误页。
2. `utils/*.py` 的脚本错误：捕获异常，打印失败原因，尽量继续处理后续文件。
3. `learn-liangliang.conf` 的 Nginx 层错误/拒绝：禁止访问 `.git`，`.js.map` 返回 404，请求限流由 Nginx 处理。

项目当前不是 JSON API 服务，不存在标准 `{code, message, data}` 这类响应格式。

---

## 服务端错误类型

### 404：文件或目录索引不存在

`server_flask.py` 在首页、文件路径、目录 `index.html` 不存在时都使用 `abort(404)`：

```python
if os.path.isfile(index_path):
    return send_from_directory(BASE_DIR, 'index.html')
else:
    abort(404)
```

```python
elif os.path.isdir(file_path):
    index_path = os.path.join(file_path, 'index.html')
    if os.path.isfile(index_path):
        return send_from_directory(file_path, 'index.html')
    else:
        abort(404)
else:
    abort(404)
```

### 403：可疑路径访问

`server_flask.py` 解码 URL 后拒绝访问隐藏路径和目录穿越：

```python
filename = urllib.parse.unquote(filename)

if filename.startswith('.') or '..' in filename:
    abort(403)
```

### Nginx 层拒绝

`learn-liangliang.conf` 中存在两类显式拒绝：

```nginx
location ~ /\.git {
    deny all;
}

location ~* \.js\.map$ {
    return 404;
}
```

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

## API 错误响应

当前没有 API，也没有 JSON 错误响应规范。Flask 的 `abort(403)`、`abort(404)` 会返回 Flask 默认 HTML 错误页；Nginx 的 `return 404` 和 `deny all` 返回 Nginx 默认错误页。

如果未来只是继续服务静态文件，应优先保持 HTTP 状态码语义清晰即可，不要引入复杂 API 错误包装。只有在明确新增 JSON API 时，才需要另行设计统一错误响应格式。

---

## 修改错误处理时的要求

- 保留路径安全检查：`filename.startswith('.')` 和 `..` 检查不可删除；如果要增强，应先验证中文路径、URL 编码路径和目录首页访问仍可用。
- 对批量抓取脚本，单个文件失败不应中断整批任务，除非失败会导致后续结果不可信。
- 继续使用中文错误提示，符合现有脚本输出风格，例如“下载失败”“抓取异常”“被限流”。
- 网络请求应设置合理 timeout；现有 `utils/03_patch_others.py` 使用 `timeout=20`，`utils/04_patch_pdf.py` 使用 `timeout=30`，新脚本不要无限等待。
- 对 429、网络异常等可恢复错误，可延续 `wait_times` + `max_retry` 的重试方式。

---

## 常见错误与注意事项

- 不要把 Flask 静态文件服务改成吞掉所有异常并返回 200；缺失文件必须仍然是 404。
- 不要给静态站点硬套 JSON API 错误格式，除非真的新增 API。
- 不要在异常日志里打印新的密钥、代理密码或完整敏感配置。
- 不要在轻量验证时触发大规模下载脚本来“测试错误处理”；这些脚本会访问外网并写入大量文件。
- `server_flask.py` 本地 `app.run()` 端口是 `60005`，生产注释和 Docker/Gunicorn 使用 `60000`；排查访问问题时先确认运行方式和端口。
