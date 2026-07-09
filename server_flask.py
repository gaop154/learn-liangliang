from flask import Flask, send_from_directory, abort
import os
import urllib.parse

app = Flask(__name__)
BASE_DIR = os.path.abspath(os.path.dirname(__file__))
CONTENT_DIR = os.path.join(BASE_DIR, "content")
CONTENT_PREFIXES = ("专栏", "文章", "极客时间", "恋爱必修课", "PDF", "assets")


def is_safe_path(filename):
    parts = [part for part in filename.split('/') if part]
    return (
        filename
        and parts
        and not filename.startswith('/')
        and not filename.startswith('.')
        and '..' not in parts
        and all(not part.startswith('.') for part in parts)
    )


def send_existing_path(root_dir, filename):
    root_dir = os.path.abspath(root_dir)
    file_path = os.path.abspath(os.path.join(root_dir, filename))

    try:
        if os.path.commonpath([root_dir, file_path]) != root_dir:
            abort(403)
    except ValueError:
        abort(403)

    if os.path.isfile(file_path):
        return send_from_directory(root_dir, filename)
    if os.path.isdir(file_path):
        index_path = os.path.join(file_path, 'index.html')
        if os.path.isfile(index_path):
            return send_from_directory(file_path, 'index.html')
    return None


@app.route('/')
def index():
    index_path = os.path.join(BASE_DIR, 'index.html')
    if os.path.isfile(index_path):
        return send_from_directory(BASE_DIR, 'index.html')
    else:
        abort(404)

@app.route('/<path:filename>')
def serve_static(filename):
    filename = urllib.parse.unquote(filename).replace('\\', '/')  # 先URL解码，并统一分隔符

    if not is_safe_path(filename):
        abort(403)

    first_part = filename.split('/', 1)[0]
    if first_part == 'content':
        abort(403)
    if first_part in CONTENT_PREFIXES:
        response = send_existing_path(CONTENT_DIR, filename)
        if response is not None:
            return response

    response = send_existing_path(BASE_DIR, filename)
    if response is not None:
        return response

    abort(404)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=60005)


# 生产环境启动
# gunicorn -w 4 -k gevent -b 127.0.0.1:60000 server_flask:app


