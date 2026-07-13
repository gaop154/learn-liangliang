"""`06_remove_third_party_tracking.py` 的字节级安全删除回归测试。"""

from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

MODULE_PATH = Path(__file__).with_name("06_remove_third_party_tracking.py")
MODULE_SPEC = importlib.util.spec_from_file_location("third_party_cleaner", MODULE_PATH)
assert MODULE_SPEC and MODULE_SPEC.loader
cleaner = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = cleaner
MODULE_SPEC.loader.exec_module(cleaner)


def page(*scripts: bytes, newline: bytes = b"\n") -> bytes:
    """构造满足结构不变量的最小归档页面。"""
    return newline.join(
        (
            b"<!doctype html>",
            b"<html><body>",
            b'<div class="book-container">',
            b'<aside class="book-sidebar"></aside>',
            '<main class="book-content"><article class="book-post">正文</article></main>'.encode("utf-8"),
            '<footer class="copyright"><a href="/cdn-cgi/l/email-protection#abc">邮箱</a></footer>'.encode("utf-8"),
            b"</div>",
            *scripts,
            b"</body></html>",
        )
    )


UMAMI = b'<script data-website-id="site-id" src="https://umami.lianglianglee.com/script.js"></script>'
HOME_UMAMI = b'<script data-website-id="site-id" src="static/script.js"></script>'
GTAG_LOADER = b'<script async src="https://www.googletagmanager.com/gtag/js?id=G-TEST"></script>'
GTAG_INLINE = b"<script>window.dataLayer=[]; function gtag(){}</script><script>gtag('config', 'G-TEST');</script>"
EMAIL_DECODE = b'<script src="/cdn-cgi/scripts/a/cloudflare-static/email-decode.min.js"></script>'
CHALLENGE = b"<script>(function(){var src='/cdn-cgi/challenge-platform/scripts/jsd/main.js';})()</script>"
NORMAL = b"<script>window.keepThisScript = true;</script>"


class RemoveThirdPartyTrackingTests(unittest.TestCase):
    def write_repository(self, root: Path, source: bytes) -> Path:
        (root / "content" / "nested").mkdir(parents=True)
        path = root / "content" / "nested" / "article.html"
        (root / "index.html").write_bytes(page())
        path.write_bytes(source)
        return path

    def test_removes_all_four_target_kinds_as_complete_blocks(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_repository(
                Path(directory),
                page(UMAMI, GTAG_LOADER, GTAG_INLINE, EMAIL_DECODE, CHALLENGE),
            )

            plan = cleaner.plan_file(path)

            self.assertEqual(
                plan.counts,
                {
                    "umami": 1,
                    "gtag": 3,
                    "email_decode": 1,
                    "challenge": 1,
                },
            )
            for target in (
                b"umami.lianglianglee.com",
                b"googletagmanager.com",
                b"gtag('config'",
                b"email-decode.min.js",
                b"challenge-platform/scripts/jsd/main.js",
            ):
                self.assertNotIn(target, plan.cleaned)

    def test_keeps_non_target_scripts_and_email_protection_anchor(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_repository(Path(directory), page(NORMAL, EMAIL_DECODE))

            plan = cleaner.plan_file(path)

            self.assertIn(NORMAL, plan.cleaned)
            self.assertIn(b"/cdn-cgi/l/email-protection#abc", plan.cleaned)
            self.assertIn(">邮箱</a>".encode("utf-8"), plan.cleaned)
            self.assertNotIn(EMAIL_DECODE, plan.cleaned)

    def test_removes_only_target_residual_whitespace_line_and_preserves_newline(self) -> None:
        for newline in (b"\n", b"\r\n"):
            with self.subTest(newline=newline), tempfile.TemporaryDirectory() as directory:
                source = page(b"        " + HOME_UMAMI, NORMAL, newline=newline)
                path = self.write_repository(Path(directory), source)

                plan = cleaner.plan_file(path)
                expected = source.replace(b"        " + HOME_UMAMI, b"")

                self.assertEqual(plan.cleaned, expected)
                self.assertEqual(plan.residual_whitespace_lines, 1)
                self.assertEqual(plan.cleaned.count(b"\r\n"), expected.count(b"\r\n"))
                self.assertEqual(plan.cleaned.count(b"\n"), expected.count(b"\n"))

    def test_keeps_unrelated_whitespace_only_line(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = page(HOME_UMAMI, b"    ", NORMAL)
            path = self.write_repository(Path(directory), source)

            plan = cleaner.plan_file(path)

            self.assertIn(b"\n    \n" + NORMAL, plan.cleaned)
            self.assertEqual(plan.residual_whitespace_lines, 0)

    def test_cleans_only_known_legacy_umami_residual_line(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = (
                b'<meta name="generator" content="Hexo 4.2.0">\n'
                b"        \n    </head>\n"
                + page(NORMAL)
            )
            path = self.write_repository(Path(directory), source)

            plan = cleaner.plan_file(path)

            self.assertIn(b'<meta name="generator" content="Hexo 4.2.0">\n\n    </head>', plan.cleaned)
            self.assertEqual(plan.residual_whitespace_lines, 1)

    def test_handles_target_script_after_body_close(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_repository(Path(directory), page(NORMAL) + b"\n" + EMAIL_DECODE)

            plan = cleaner.plan_file(path)

            self.assertIn(NORMAL, plan.cleaned)
            self.assertNotIn(EMAIL_DECODE, plan.cleaned)

    def test_finds_target_script_when_preceded_by_comment_opening_text(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = page(NORMAL) + b"<!-- text --><script async src=\"https://www.googletagmanager.com/gtag/js?id=G-TEST\"></script>"
            path = self.write_repository(Path(directory), source)

            plan = cleaner.plan_file(path)

            self.assertEqual(plan.counts["gtag"], 1)
            self.assertNotIn(b"googletagmanager.com", plan.cleaned)

    def test_ignores_literal_script_text_in_article_content(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            article_text = "<p>将&lt;script>编码为文本。</p>".encode("utf-8")
            source = page(NORMAL) + article_text + GTAG_LOADER
            path = self.write_repository(Path(directory), source)

            plan = cleaner.plan_file(path)

            self.assertIn("将&lt;script>".encode("utf-8"), plan.cleaned)
            self.assertNotIn(b"googletagmanager.com", plan.cleaned)

    def test_ignores_unclosed_literal_script_before_external_target(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            article_text = "<p>请将<script>转义后显示给用户。</p>".encode("utf-8")
            source = page(NORMAL) + article_text + GTAG_LOADER
            path = self.write_repository(Path(directory), source)

            plan = cleaner.plan_file(path)

            self.assertIn(article_text, plan.cleaned)
            self.assertNotIn(GTAG_LOADER, plan.cleaned)

    def test_rejects_unclosed_target_script(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_repository(
                Path(directory),
                page()
                + b'<script data-website-id="site-id" src="https://umami.lianglianglee.com/script.js">',
            )

            with self.assertRaisesRegex(cleaner.SafetyError, "缺少 </script>"):
                cleaner.plan_file(path)

    def test_dry_run_apply_and_check_modes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            article = self.write_repository(root, page(UMAMI, GTAG_LOADER, GTAG_INLINE, EMAIL_DECODE, CHALLENGE))
            original = article.read_bytes()
            expected = {
                "umami": 1,
                "gtag": 3,
                "email_decode": 1,
                "challenge": 1,
            }

            with patch.object(cleaner, "EXPECTED_BASELINE", expected):
                self.assertEqual(cleaner.main([], root), 0)
                self.assertEqual(article.read_bytes(), original)
                self.assertEqual(cleaner.main(["--apply"], root), 0)
                self.assertNotEqual(article.read_bytes(), original)
                self.assertEqual(cleaner.main(["--check"], root), 0)

    def test_apply_refuses_baseline_exceeding_without_writing(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            article = self.write_repository(root, page(UMAMI, UMAMI))
            original = article.read_bytes()

            with patch.object(
                cleaner,
                "EXPECTED_BASELINE",
                {"umami": 1, "gtag": 0, "email_decode": 0, "challenge": 0},
            ):
                self.assertEqual(cleaner.main(["--apply"], root), 1)
            self.assertEqual(article.read_bytes(), original)

    def test_apply_allows_remaining_targets_after_partial_cleanup(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            article = self.write_repository(root, page(GTAG_LOADER))

            with patch.object(
                cleaner,
                "EXPECTED_BASELINE",
                {"umami": 1, "gtag": 1, "email_decode": 1, "challenge": 1},
            ):
                self.assertEqual(cleaner.main(["--apply"], root), 0)
            self.assertNotIn(GTAG_LOADER, article.read_bytes())


if __name__ == "__main__":
    unittest.main()
