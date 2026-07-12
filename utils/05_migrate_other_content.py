#!/usr/bin/env python3
"""将旧四类静态内容迁移到 content/其他，并重建归档页面导航。

默认仅输出迁移计划；使用 --apply 才会写入文件或移动目录。脚本不访问网络、
不调用其他脚本，也不执行任何 Git 操作。--verify-only 只校验已迁移结构。
"""

from __future__ import annotations

import argparse
import re
import shutil
import sys
from dataclasses import dataclass
from html.parser import HTMLParser
from pathlib import Path
from typing import Iterable
from urllib.parse import quote

ROOT = Path(__file__).resolve().parent.parent
CONTENT = ROOT / "content"
OTHER = CONTENT / "其他"
CATEGORIES = ("恋爱必修课", "文章", "极客时间", "PDF")
ARTICLE_CATEGORIES = CATEGORIES[:3]
DONATION_PATH = "/assets/捐赠.md.html"

MENU_PATTERN = re.compile(
    r"(<div\b[^>]*\bclass=(['\"])[^'\"]*\bbook-menu\b[^'\"]*\2[^>]*>)"
    r".*?"
    r"(</div>\s*</div>\s*<div\b[^>]*\bclass=(['\"])[^'\"]*\bsidebar-toggle\b)",
    re.IGNORECASE | re.DOTALL,
)
DONATION_LI_PATTERN = re.compile(
    r"\s*<li\b[^>]*>\s*<a\b[^>]*\bhref\s*=\s*(['\"])"
    + re.escape(DONATION_PATH)
    + r"\1[^>]*>.*?</a>\s*</li>",
    re.IGNORECASE | re.DOTALL,
)
DONATION_ANCHOR_PATTERN = re.compile(
    r"<a\b[^>]*\bhref\s*=\s*(['\"])"
    + re.escape(DONATION_PATH)
    + r"\1[^>]*>.*?</a>",
    re.IGNORECASE | re.DOTALL,
)


@dataclass(frozen=True)
class FileChange:
    path: Path
    content: str


def legacy_url_pattern(category: str) -> re.Pattern[str]:
    """匹配 URL 值开头的旧分类根路径，不把 /其他/PDF 误判为旧 /PDF。"""
    encoded = quote(category, safe="")
    return re.compile(
        r"(?:^|(?<=[\"'=\s(]))/(?:"
        + re.escape(category)
        + r"|"
        + re.escape(encoded)
        + r")(?=(?:/|[?#'\"<>)\s]|$))",
        re.IGNORECASE,
    )


URL_PATTERNS = {category: legacy_url_pattern(category) for category in CATEGORIES}


def rewrite_legacy_urls(html: str) -> str:
    """只替换旧四分类的绝对站内 URL，不触及相对 URL、外链或其他公开根路径。"""
    for category, pattern in URL_PATTERNS.items():
        html = pattern.sub(f"/其他/{category}", html)
    html = DONATION_LI_PATTERN.sub("", html)
    return DONATION_ANCHOR_PATTERN.sub("", html)


def primary_navigation() -> str:
    return '''
                <ul class="uncollapsible">
                    <li><a href="/" class="current-tab">首页</a></li>
                </ul>
                <ul class="uncollapsible">
                    <li><a class="menu-item" href="/专栏/" id="专栏">专栏</a></li>
                    <li><a class="menu-item" href="/其他/" id="其他">其他</a></li>
                </ul>
'''


def course_navigation(course_name: str, chapters: list[tuple[str, str]]) -> str:
    entries = "\n".join(
        f'                    <li><a class="menu-item" href="{href}" id="{escape_attr(title)}">{escape_html(title)}</a></li>'
        for href, title in chapters
    )
    return f'''
                <ul class="uncollapsible">
                    <li><a href="/专栏/">← 返回专栏</a></li>
                </ul>
                <ul class="uncollapsible">
{entries}
                </ul>
'''


def other_article_navigation(category: str, articles: list[tuple[str, str]]) -> str:
    entries = "\n".join(
        f'                    <li><a class="menu-item" href="{href}" id="{escape_attr(title)}">{escape_html(title)}</a></li>'
        for href, title in articles
    )
    return f'''
                <ul class="uncollapsible">
                    <li><a href="/其他/">← 返回其他</a></li>
                </ul>
                <ul class="uncollapsible">
{entries}
                </ul>
'''


def escape_html(value: str) -> str:
    return value.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def escape_attr(value: str) -> str:
    return escape_html(value).replace('"', "&quot;")


def replace_sidebar(html: str, navigation: str, path: Path) -> str:
    replacement, count = MENU_PATTERN.subn(r"\1" + navigation + r"\3", html, count=1)
    if count != 1:
        raise ValueError(f"无法唯一定位侧边栏: {relative(path)}")
    return replacement


def canonical_public_path(href: str) -> str | None:
    """将索引中的本域绝对链接归一为公开 URL，忽略外链和相对链接。"""
    href = href.strip()
    if not href.startswith("/") or href.startswith("//"):
        return None
    for category in CATEGORIES:
        href = URL_PATTERNS[category].sub(f"/其他/{category}", href)
    return href


class ArticleLinkParser(HTMLParser):
    """仅解析 .book-post 内的链接，避免把侧栏和评论区误当作分类文章集合。"""

    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.post_depth = 0
        self.anchor_href: str | None = None
        self.anchor_text: list[str] = []
        self.links: list[tuple[str, str]] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        attributes = dict(attrs)
        classes = attributes.get("class", "") or ""
        if tag == "div":
            if self.post_depth:
                self.post_depth += 1
            elif "book-post" in classes.split():
                self.post_depth = 1
        elif self.post_depth and tag == "a" and attributes.get("href") is not None:
            self.anchor_href = attributes["href"]
            self.anchor_text = []

    def handle_endtag(self, tag: str) -> None:
        if tag == "a" and self.anchor_href is not None:
            self.links.append((self.anchor_href, "".join(self.anchor_text).strip()))
            self.anchor_href = None
            self.anchor_text = []
        if tag == "div" and self.post_depth:
            self.post_depth -= 1

    def handle_data(self, data: str) -> None:
        if self.anchor_href is not None:
            self.anchor_text.append(data)


def extract_article_links(html: str, required_prefix: str) -> list[tuple[str, str]]:
    """从正文 .book-post 按 DOM 顺序提取实际文章，作为其他分类的唯一导航来源。"""
    parser = ArticleLinkParser()
    parser.feed(html)
    parser.close()
    result: list[tuple[str, str]] = []
    seen: set[str] = set()
    for raw_href, raw_title in parser.links:
        href = canonical_public_path(raw_href)
        if href is None or not href.startswith(required_prefix) or not href.endswith(".md.html"):
            continue
        if href in seen:
            continue
        seen.add(href)
        title = raw_title or Path(href).name
        result.append((href, title))
    return result


def source_or_target(category: str) -> Path:
    source = CONTENT / category
    target = OTHER / category
    if source.exists() and target.exists():
        raise ValueError(f"源目录和目标目录同时存在，拒绝覆盖: {relative(source)} / {relative(target)}")
    if source.exists():
        return source
    if target.exists():
        return target
    raise ValueError(f"缺少内容分类目录: {relative(source)} 或 {relative(target)}")


def relative(path: Path) -> str:
    return path.relative_to(ROOT).as_posix()


def target_path(path: Path) -> Path:
    for category in CATEGORIES:
        source = CONTENT / category
        if path == source or source in path.parents:
            return OTHER / category / path.relative_to(source)
    return path


def html_paths(category_roots: Iterable[Path]) -> Iterable[Path]:
    yield ROOT / "index.html"
    for path in CONTENT.rglob("*.html"):
        if path != CONTENT / "assets" / "捐赠.md.html":
            yield path


def is_course_page(path: Path) -> tuple[str, bool] | None:
    course_root = CONTENT / "专栏"
    if course_root not in path.parents:
        return None
    relative_path = path.relative_to(course_root)
    if len(relative_path.parts) < 2:
        return None
    return relative_path.parts[0], path.name.endswith(".md.html")


def load_course_chapters(course_name: str) -> list[tuple[str, str]]:
    index = CONTENT / "专栏" / course_name / "index.html"
    if not index.exists():
        raise ValueError(f"课程索引不存在: {relative(index)}")
    return extract_article_links(index.read_text(encoding="utf-8"), f"/专栏/{course_name}/")


def classify_other_article(path: Path) -> str | None:
    for category in ARTICLE_CATEGORIES:
        category_root = OTHER / category
        if category_root in path.parents and path.name.endswith(".md.html"):
            return category
    return None


def build_other_index() -> str:
    entries = "\n".join(
        f'                        <li><a href="/其他/{category}/">{category}</a></li>' for category in CATEGORIES
    )
    return f'''<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1.0, user-scalable=no">
    <meta http-equiv="content-language" content="zh-cn">
    <meta name="description" content="其他">
    <link rel="icon" href="/static/favicon.png">
    <title>其他</title>
    <link rel="stylesheet" href="/static/index.css">
    <link rel="stylesheet" href="/static/highlight.min.css">
    <script src="/static/highlight.min.js"></script>
</head>
<body>
    <div class="book-container">
        <div class="book-sidebar">
            <div class="book-brand"><a href="/"><img src="/static/favicon.png"><span>技术文章摘抄</span></a></div>
            <div class="book-menu uncollapsible">{primary_navigation()}            </div>
        </div>
        <div class="sidebar-toggle" onclick="sidebar_toggle()" onmouseover="add_inner()" onmouseleave="remove_inner()"><div class="sidebar-toggle-inner"></div></div>
        <div class="off-canvas-content"><div class="columns"><div class="column col-12 col-lg-12">
            <div class="book-navbar"><header class="navbar"><section class="navbar-section"><a onclick="open_sidebar()"><i class="icon icon-menu"></i></a></section></header></div>
            <div class="book-content" style="max-width: 960px; margin: 0 auto; overflow-x: auto; overflow-y: hidden;">
                <div class="book-post"><p id="tip" align="center"></p><h1 id="title" data-id="其他" class="title">其他</h1><div><ul>
{entries}
                </ul></div></div>
            </div>
        </div></div><div class="copyright"><hr><p>内容源自网络收集，仅供交流学习使用</p></div></div>
        <a class="off-canvas-overlay" onclick="hide_canvas()"></a>
    </div>
    <script src="/static/index.js"></script>
</body>
</html>
'''


def prepare_changes() -> tuple[list[FileChange], list[tuple[Path, Path]], list[Path]]:
    roots = {category: source_or_target(category) for category in CATEGORIES}
    other_articles = {
        category: extract_article_links(
            (roots[category] / "index.html").read_text(encoding="utf-8"), f"/其他/{category}/"
        )
        for category in ARTICLE_CATEGORIES
    }
    course_chapters: dict[str, list[tuple[str, str]]] = {}
    changes: list[FileChange] = []

    for path in html_paths(roots.values()):
        if not path.exists() or path == OTHER / "index.html":
            continue
        html = rewrite_legacy_urls(path.read_text(encoding="utf-8"))
        resulting_path = target_path(path)
        course = is_course_page(path)
        other_category = classify_other_article(resulting_path)

        if course is not None:
            course_name, _is_article = course
            chapters = course_chapters.setdefault(course_name, load_course_chapters(course_name))
            html = replace_sidebar(html, course_navigation(course_name, chapters), path)
        elif other_category is not None:
            html = replace_sidebar(html, other_article_navigation(other_category, other_articles[other_category]), path)
        else:
            html = replace_sidebar(html, primary_navigation(), path)

        destination = target_path(path)
        if not destination.exists() or destination.read_text(encoding="utf-8") != html:
            changes.append(FileChange(destination, html))

    other_index = OTHER / "index.html"
    other_index_content = build_other_index()
    if not other_index.exists() or other_index.read_text(encoding="utf-8") != other_index_content:
        changes.append(FileChange(other_index, other_index_content))

    moves = [(CONTENT / category, OTHER / category) for category in CATEGORIES if (CONTENT / category).exists()]
    donation = CONTENT / "assets" / "捐赠.md.html"
    removals = [donation] if donation.exists() else []
    return changes, moves, removals


def verify() -> list[str]:
    errors: list[str] = []
    for category in CATEGORIES:
        source = CONTENT / category
        target = OTHER / category
        if source.exists():
            errors.append(f"旧源目录仍存在: {relative(source)}")
        if not target.is_dir():
            errors.append(f"目标分类目录不存在: {relative(target)}")
    if not (OTHER / "index.html").is_file():
        errors.append("缺少其他总目录: content/其他/index.html")
    if (CONTENT / "assets" / "捐赠.md.html").exists():
        errors.append("捐赠页仍存在: content/assets/捐赠.md.html")
    if (ROOT / "utils" / "03_patch_donation_md_links.py").exists():
        errors.append("捐赠维护脚本仍存在: utils/03_patch_donation_md_links.py")

    paths = [ROOT / "index.html", *CONTENT.rglob("*.html")]
    for path in paths:
        html = path.read_text(encoding="utf-8")
        if DONATION_PATH in html:
            errors.append(f"仍含捐赠入口: {relative(path)}")
        for category, pattern in URL_PATTERNS.items():
            if pattern.search(html):
                errors.append(f"仍含旧 {category} 绝对 URL: {relative(path)}")
                break
    return errors


def print_plan(changes: list[FileChange], moves: list[tuple[Path, Path]], removals: list[Path]) -> None:
    print("迁移计划（未写入）：")
    for source, target in moves:
        print(f"  移动目录: {relative(source)} -> {relative(target)}")
    print(f"  重写/重建 HTML: {len(changes)} 个")
    for path in removals:
        print(f"  删除捐赠页: {relative(path)}")
    if (ROOT / "utils" / "03_patch_donation_md_links.py").exists():
        print("  删除捐赠维护脚本: utils/03_patch_donation_md_links.py")


def apply(changes: list[FileChange], moves: list[tuple[Path, Path]], removals: list[Path]) -> None:
    """预检完成后才进入写阶段；目录移动失败将抛错，绝不覆盖目标。"""
    for source, target in moves:
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(source), str(target))
    for change in changes:
        change.path.parent.mkdir(parents=True, exist_ok=True)
        change.path.write_text(change.content, encoding="utf-8")
    for path in removals:
        path.unlink()
    donation_script = ROOT / "utils" / "03_patch_donation_md_links.py"
    if donation_script.exists():
        donation_script.unlink()
    assets = CONTENT / "assets"
    if assets.is_dir() and not any(assets.iterdir()):
        asset_reference = re.compile(r"/(?:assets)(?:/|[?#'\"<>)\s]|$)")
        has_references = any(
            asset_reference.search(path.read_text(encoding="utf-8"))
            for path in [ROOT / "index.html", *CONTENT.rglob("*.html")]
        )
        if not has_references:
            assets.rmdir()


def main() -> int:
    parser = argparse.ArgumentParser(description="迁移旧四类内容到 content/其他（默认 dry-run，不联网）")
    modes = parser.add_mutually_exclusive_group()
    modes.add_argument("--apply", action="store_true", help="执行已预检的目录迁移、HTML 改写和捐赠清理")
    modes.add_argument("--verify-only", action="store_true", help="仅校验已迁移的目录、链接和捐赠清理结果")
    args = parser.parse_args()

    if args.verify_only:
        errors = verify()
        if errors:
            print("结构校验失败：", file=sys.stderr)
            for error in errors:
                print(f"  - {error}", file=sys.stderr)
            return 1
        print("结构校验通过：四类内容、导航链接、其他总目录和捐赠清理均符合预期。")
        return 0

    try:
        changes, moves, removals = prepare_changes()
    except (OSError, ValueError) as error:
        print(f"迁移预检失败：{error}", file=sys.stderr)
        return 1
    print_plan(changes, moves, removals)
    if not args.apply:
        print("默认 dry-run 完成；如确认执行，请运行: python utils/05_migrate_other_content.py --apply")
        return 0

    try:
        apply(changes, moves, removals)
    except OSError as error:
        print(f"迁移写入失败：{error}", file=sys.stderr)
        return 1
    errors = verify()
    if errors:
        print("迁移已执行，但后置结构校验失败：", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1
    print("迁移和后置结构校验完成。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
