"""以字节级精确替换安全移除归档 HTML 中的指定第三方脚本。"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence

REPOSITORY_ROOT = Path(__file__).resolve().parents[1]
CONTENT_ROOT = REPOSITORY_ROOT / "content"
MAX_TARGET_COUNT = 500_000
EXPECTED_BASELINE = {
    "umami": 6464,
    "gtag": 6458,
    "email_decode": 6458,
    "challenge": 6464,
}

SCRIPT_OPEN_RE = re.compile(rb"<script\b", re.IGNORECASE)
SCRIPT_CLOSE_RE = re.compile(rb"</script\s*>", re.IGNORECASE)
SCRIPT_TAG_RE = re.compile(rb"<script\b[^>]*>", re.IGNORECASE)
HTML_COMMENT_OPEN = b"<!--"
HTML_COMMENT_CLOSE = b"-->"
SRC_ATTRIBUTE_RE = re.compile(
    rb"\bsrc\s*=\s*(?:\"([^\"]*)\"|'([^']*)'|([^\s>]+))", re.IGNORECASE
)
DATA_WEBSITE_ID_RE = re.compile(rb"\bdata-website-id\b", re.IGNORECASE)
GTAG_INLINE_RE = re.compile(
    rb"(?:\bgtag\s*\(\s*(['\"])config\1\s*,|\bwindow\.dataLayer\b|\bdataLayer\.push\s*\(\s*arguments\s*\))",
    re.IGNORECASE,
)
EMAIL_ANCHOR_RE = re.compile(
    rb"<a\b[^>]*\bhref\s*=\s*(?:\"[^\"]*cdn-cgi/l/email-protection[^\"]*\"|'[^']*cdn-cgi/l/email-protection[^']*'|[^\s>]*cdn-cgi/l/email-protection[^\s>]*)[^>]*>",
    re.IGNORECASE,
)
# 兼容首次清理已执行后留下的、原 Umami 独占行的缩进；固定元标签和闭合 head 上下文避免触及其他空白。
LEGACY_UMAMI_RESIDUAL_RE = re.compile(
    rb'(<meta\b[^>]*\bname\s*=\s*(?:"generator"|\'generator\')[^>]*>\r?\n)[ \t]+(?=\r?\n[ \t]*</head>)',
    re.IGNORECASE,
)
STRUCTURE_MARKERS = {
    "body": re.compile(rb"<body\b", re.IGNORECASE),
    "book-container": re.compile(
        rb"<[^>]+\bclass\s*=\s*(?:\"[^\"]*\bbook-container\b[^\"]*\"|'[^']*\bbook-container\b[^']*')",
        re.IGNORECASE,
    ),
    "book-sidebar": re.compile(
        rb"<[^>]+\bclass\s*=\s*(?:\"[^\"]*\bbook-sidebar\b[^\"]*\"|'[^']*\bbook-sidebar\b[^']*')",
        re.IGNORECASE,
    ),
    "book-content": re.compile(
        rb"<[^>]+\bclass\s*=\s*(?:\"[^\"]*\bbook-content\b[^\"]*\"|'[^']*\bbook-content\b[^']*')",
        re.IGNORECASE,
    ),
    "book-post": re.compile(
        rb"<[^>]+\bclass\s*=\s*(?:\"[^\"]*\bbook-post\b[^\"]*\"|'[^']*\bbook-post\b[^']*')",
        re.IGNORECASE,
    ),
    "copyright": re.compile(
        rb"<[^>]+\bclass\s*=\s*(?:\"[^\"]*\bcopyright\b[^\"]*\"|'[^']*\bcopyright\b[^']*')",
        re.IGNORECASE,
    ),
}


class SafetyError(ValueError):
    """文件结构或目标脚本无法被唯一、安全地识别。"""


@dataclass(frozen=True)
class ScriptBlock:
    start: int
    end: int
    categories: tuple[str, ...]


@dataclass(frozen=True)
class FilePlan:
    path: Path
    source: bytes
    cleaned: bytes
    counts: dict[str, int]
    residual_whitespace_lines: int

    @property
    def changed(self) -> bool:
        return self.source != self.cleaned


@dataclass(frozen=True)
class RepositoryPlan:
    files: tuple[FilePlan, ...]
    counts: dict[str, int]

    @property
    def changed_files(self) -> int:
        return sum(file_plan.changed for file_plan in self.files)

    @property
    def removed_blocks(self) -> int:
        return sum(sum(file_plan.counts.values()) for file_plan in self.files)

    @property
    def normalized_whitespace_lines(self) -> int:
        return sum(file_plan.residual_whitespace_lines for file_plan in self.files)


def iter_html_files(repository_root: Path = REPOSITORY_ROOT) -> Iterable[Path]:
    """按稳定顺序返回首页和全部归档 HTML。"""
    index_file = repository_root / "index.html"
    if not index_file.is_file():
        raise SafetyError(f"未找到首页: {index_file}")
    yield index_file

    content_root = repository_root / "content"
    if not content_root.is_dir():
        raise SafetyError(f"未找到内容目录: {content_root}")
    yield from sorted(content_root.rglob("*.html"))


def _find_tag_end(source: bytes, start: int, path: Path) -> int:
    quote: int | None = None
    for index in range(start, len(source)):
        character = source[index]
        if quote is not None:
            if character == quote:
                quote = None
        elif character in (ord("'"), ord('"')):
            quote = character
        elif character == ord(">"):  # 标签结束。
            return index + 1
    raise SafetyError(f"{path}: 未闭合的 <script> 起始标签")


def _next_script_start(source: bytes, cursor: int, path: Path) -> int | None:
    """跳过真正的 HTML 注释，避免把注释文本误识别为标签。"""
    while True:
        script_match = SCRIPT_OPEN_RE.search(source, cursor)
        comment_start = source.find(HTML_COMMENT_OPEN, cursor)
        if comment_start == -1 or (
            script_match is not None and script_match.start() < comment_start
        ):
            return script_match.start() if script_match else None

        comment_end = source.find(HTML_COMMENT_CLOSE, comment_start + len(HTML_COMMENT_OPEN))
        if comment_end == -1:
            raise SafetyError(f"{path}: 未闭合 HTML 注释，拒绝判断脚本边界")
        if script_match is not None and script_match.start() < comment_end:
            return script_match.start()
        cursor = comment_end + len(HTML_COMMENT_CLOSE)


def _attribute_value(opening_tag: bytes) -> bytes | None:
    match = SRC_ATTRIBUTE_RE.search(opening_tag)
    if match is None:
        return None
    return next(value for value in match.groups() if value is not None)


def _categories_for_block(opening_tag: bytes, content: bytes) -> tuple[str, ...]:
    source = _attribute_value(opening_tag)
    source_lower = source.lower() if source is not None else b""
    opening_lower = opening_tag.lower()
    content_lower = content.lower()
    categories: list[str] = []

    is_known_umami_source = (
        b"umami.lianglianglee.com/script.js" in source_lower
        or source_lower.rstrip(b"/").endswith(b"static/script.js")
    )
    if DATA_WEBSITE_ID_RE.search(opening_tag) and is_known_umami_source:
        categories.append("umami")

    if b"googletagmanager.com/gtag/js" in source_lower:
        categories.append("gtag")
    if source is None and GTAG_INLINE_RE.search(content):
        categories.append("gtag")

    if source is not None and b"email-decode.min.js" in source_lower:
        categories.append("email_decode")

    if source is None and b"challenge-platform/scripts/jsd/main.js" in content_lower:
        categories.append("challenge")

    return tuple(categories)


def find_script_blocks(source: bytes, path: Path) -> tuple[ScriptBlock, ...]:
    """只按字节确定完整 ``<script>...</script>`` 区间，不重序列化 HTML。"""
    blocks: list[ScriptBlock] = []
    cursor = 0
    while True:
        start = _next_script_start(source, cursor, path)
        if start is None:
            break
        opening_end = _find_tag_end(source, start, path)
        opening_tag = source[start:opening_end]
        if SCRIPT_TAG_RE.fullmatch(opening_tag) is None:
            cursor = opening_end
            continue

        source_attribute = _attribute_value(opening_tag)
        closing_match = SCRIPT_CLOSE_RE.search(source, opening_end)
        if closing_match is None:
            if _categories_for_block(opening_tag, b""):
                raise SafetyError(f"{path}: <script> 块缺少 </script>，拒绝处理")
            cursor = opening_end
            continue

        content = source[opening_end : closing_match.start()]
        # 历史正文中有未闭合的字面量“<script>”。若无 src 的候选块在
        # 闭合前又出现新的 <script>，当前开头只能是正文文本；跳过它，
        # 让后续真实脚本单独参与识别，不能吞没后续标签。
        if source_attribute is None and SCRIPT_OPEN_RE.search(content):
            cursor = opening_end
            continue
        categories = _categories_for_block(opening_tag, content)
        if len(categories) > 1:
            raise SafetyError(
                f"{path}: 一个 <script> 块同时命中 {', '.join(categories)}，拒绝处理"
            )
        if categories:
            blocks.append(ScriptBlock(start, closing_match.end(), categories))
            if len(blocks) > MAX_TARGET_COUNT:
                raise SafetyError("目标脚本数量超出安全上限")
        cursor = closing_match.end()
    return tuple(blocks)


def _validate_utf8_and_nonempty(source: bytes, path: Path) -> None:
    if not source:
        raise SafetyError(f"{path}: HTML 文件为空")
    try:
        source.decode("utf-8", errors="strict")
    except UnicodeDecodeError as error:
        raise SafetyError(f"{path}: 不是合法 UTF-8: {error}") from error


def _structure_counts(source: bytes) -> dict[str, int]:
    return {name: len(pattern.findall(source)) for name, pattern in STRUCTURE_MARKERS.items()}


def _validate_structure(source: bytes, cleaned: bytes, path: Path) -> None:
    source_counts = _structure_counts(source)
    cleaned_counts = _structure_counts(cleaned)
    missing = [name for name, count in source_counts.items() if count == 0]
    if missing:
        raise SafetyError(f"{path}: 缺少结构标记: {', '.join(missing)}")
    changed = [
        name
        for name, count in source_counts.items()
        if cleaned_counts[name] != count
    ]
    if changed:
        raise SafetyError(f"{path}: 清理改变了结构标记: {', '.join(changed)}")
    _validate_utf8_and_nonempty(cleaned, path)


def _validate_email_anchors(source: bytes, cleaned: bytes, path: Path) -> None:
    source_count = len(EMAIL_ANCHOR_RE.findall(source))
    cleaned_count = len(EMAIL_ANCHOR_RE.findall(cleaned))
    if cleaned_count != source_count:
        raise SafetyError(f"{path}: 清理改变了 Cloudflare 邮箱保护锚点")


def _remove_residual_whitespace_line(cleaned: bytes, start: int, end: int) -> bytes:
    """仅删除目标块所在行因移除后留下的空格或制表符，保留原换行。"""
    line_start = cleaned.rfind(b"\n", 0, start) + 1
    line_end = cleaned.find(b"\n", end)
    if line_end == -1:
        line_end = len(cleaned)
    if cleaned[line_start:line_end].strip(b" \t\r"):
        return cleaned
    newline_start = line_end - 1 if line_end > line_start and cleaned[line_end - 1] == ord("\r") else line_end
    return cleaned[:line_start] + cleaned[newline_start:]


def plan_file(path: Path) -> FilePlan:
    """读取原始 bytes，构造候选 bytes，并验证所有保留不变量。"""
    source = path.read_bytes()
    _validate_utf8_and_nonempty(source, path)
    blocks = find_script_blocks(source, path)

    cleaned = source
    cleaned, residual_whitespace_lines = LEGACY_UMAMI_RESIDUAL_RE.subn(rb"\1", cleaned)
    for block in reversed(blocks):
        cleaned = cleaned[: block.start] + cleaned[block.end :]
        normalized = _remove_residual_whitespace_line(cleaned, block.start, block.start)
        if normalized != cleaned:
            residual_whitespace_lines += 1
            cleaned = normalized

    _validate_structure(source, cleaned, path)
    _validate_email_anchors(source, cleaned, path)
    counts = {category: 0 for category in EXPECTED_BASELINE}
    for block in blocks:
        counts[block.categories[0]] += 1
    return FilePlan(
        path=path,
        source=source,
        cleaned=cleaned,
        counts=counts,
        residual_whitespace_lines=residual_whitespace_lines,
    )


def build_repository_plan(repository_root: Path = REPOSITORY_ROOT) -> RepositoryPlan:
    """在不写入文件的前提下完成全量扫描与候选结果验证。"""
    file_plans = tuple(plan_file(path) for path in iter_html_files(repository_root))
    counts = {category: 0 for category in EXPECTED_BASELINE}
    for file_plan in file_plans:
        for category, count in file_plan.counts.items():
            counts[category] += count
    return RepositoryPlan(files=file_plans, counts=counts)


def validate_baseline(plan: RepositoryPlan) -> None:
    if plan.counts != EXPECTED_BASELINE:
        actual = ", ".join(
            f"{name}={plan.counts[name]}" for name in EXPECTED_BASELINE
        )
        expected = ", ".join(
            f"{name}={count}" for name, count in EXPECTED_BASELINE.items()
        )
        raise SafetyError(f"目标基线不符（实际: {actual}；预期: {expected}）")


def validate_remaining_targets(plan: RepositoryPlan) -> None:
    """允许已通过首轮审计后的剩余目标继续按同一精确规则清理。"""
    unexpected = {
        name: count
        for name, count in plan.counts.items()
        if count > EXPECTED_BASELINE[name]
    }
    if unexpected:
        details = ", ".join(f"{name}={count}" for name, count in unexpected.items())
        raise SafetyError(f"目标数量超过已审计基线: {details}")


def validate_clean_repository(plan: RepositoryPlan) -> None:
    remaining = {name: count for name, count in plan.counts.items() if count != 0}
    if remaining:
        details = ", ".join(f"{name}={count}" for name, count in remaining.items())
        raise SafetyError(f"仍存在目标第三方脚本: {details}")

    email_anchor_count = sum(
        len(EMAIL_ANCHOR_RE.findall(file_plan.source)) for file_plan in plan.files
    )
    if email_anchor_count == 0:
        raise SafetyError("未找到任何 Cloudflare 邮箱保护锚点，拒绝将检查视为通过")


def apply_plan(plan: RepositoryPlan) -> None:
    """仅在全量计划已完成全部验证后，按原始 bytes 写入实际变更。"""
    for file_plan in plan.files:
        if file_plan.changed:
            file_plan.path.write_bytes(file_plan.cleaned)


def print_report(plan: RepositoryPlan, mode: str) -> None:
    print(f"{mode}：扫描 {len(plan.files)} 个 HTML 文件。")
    print(f"Umami: {plan.counts['umami']}")
    print(f"Google gtag: {plan.counts['gtag']}")
    print(f"email-decode: {plan.counts['email_decode']}")
    print(f"challenge: {plan.counts['challenge']}")
    print(f"待移除完整 script 块: {plan.removed_blocks}")
    print(f"同时清理的目标残留空白行: {plan.normalized_whitespace_lines}")
    print(f"涉及文件: {plan.changed_files}")


def main(
    argv: Sequence[str] | None = None,
    repository_root: Path = REPOSITORY_ROOT,
) -> int:
    parser = argparse.ArgumentParser(description="安全移除归档页面中的指定第三方脚本")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--apply", action="store_true", help="通过全量校验后写入删除结果")
    mode.add_argument("--check", action="store_true", help="验证目标脚本为零且保留不变量成立")
    args = parser.parse_args(argv)

    try:
        plan = build_repository_plan(repository_root)
        if args.check:
            validate_clean_repository(plan)
            if plan.changed_files:
                raise SafetyError(
                    f"仍有 {plan.normalized_whitespace_lines} 行已知目标残留空白，"
                    "请执行 --apply"
                )
            print_report(plan, "检查通过")
            return 0

        is_already_clean = all(count == 0 for count in plan.counts.values())
        if is_already_clean:
            validate_clean_repository(plan)
            if plan.changed_files == 0:
                print_report(plan, "已是清理完成状态")
                return 0
            if not args.apply:
                print_report(plan, "Dry-run（仅修复已知目标残留空白行）")
                return 0
            print_report(plan, "将仅修复已知目标残留空白行")
        else:
            validate_remaining_targets(plan)
            print_report(plan, "将执行写入" if args.apply else "Dry-run")
        if not args.apply:
            return 0

        apply_plan(plan)
        cleaned_plan = build_repository_plan(repository_root)
        validate_clean_repository(cleaned_plan)
        print_report(cleaned_plan, "写入后检查通过")
        return 0
    except SafetyError as error:
        print(f"安全检查失败: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
