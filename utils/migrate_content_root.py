#!/usr/bin/env python3
"""将文章相关内容目录迁移到 content/ 下。

默认只输出 dry-run 迁移计划；只有显式传入 --execute 才会实际移动目录。
脚本只处理白名单目录，且在目标路径已存在时拒绝覆盖，避免误伤已有内容。
"""

from __future__ import annotations

import argparse
import shutil
from dataclasses import dataclass
from pathlib import Path

CONTENT_DIR_NAME = "content"
MIGRATE_DIRS = ["专栏", "文章", "极客时间", "恋爱必修课", "PDF", "assets"]


@dataclass(frozen=True)
class MovePlan:
    name: str
    source: Path
    target: Path
    file_count: int
    dir_count: int


def count_tree(path: Path) -> tuple[int, int]:
    file_count = 0
    dir_count = 0
    for item in path.rglob("*"):
        if item.is_dir():
            dir_count += 1
        elif item.is_file():
            file_count += 1
    return file_count, dir_count


def build_plan(root_dir: Path) -> tuple[list[MovePlan], list[str], list[str]]:
    content_dir = root_dir / CONTENT_DIR_NAME
    plans: list[MovePlan] = []
    warnings: list[str] = []
    errors: list[str] = []

    for name in MIGRATE_DIRS:
        source = root_dir / name
        target = content_dir / name

        if source.exists() and target.exists():
            errors.append(f"冲突：源目录和目标目录同时存在，拒绝覆盖：{source} -> {target}")
            continue
        if not source.exists() and target.exists():
            warnings.append(f"已迁移，跳过：{target}")
            continue
        if not source.exists():
            warnings.append(f"源目录不存在，跳过：{source}")
            continue
        if not source.is_dir():
            errors.append(f"源路径不是目录，拒绝迁移：{source}")
            continue

        file_count, dir_count = count_tree(source)
        plans.append(MovePlan(name=name, source=source, target=target, file_count=file_count, dir_count=dir_count))

    return plans, warnings, errors


def print_plan(plans: list[MovePlan], warnings: list[str], errors: list[str], execute: bool) -> None:
    mode = "执行模式" if execute else "Dry-run 模式"
    print(f"内容目录迁移计划（{mode}）：")
    print(f"白名单目录：{', '.join(MIGRATE_DIRS)}")
    print()

    if plans:
        for plan in plans:
            print(f"- {plan.source} -> {plan.target}（文件 {plan.file_count} 个，子目录 {plan.dir_count} 个）")
    else:
        print("- 没有需要迁移的目录")

    if warnings:
        print("\n提示：")
        for warning in warnings:
            print(f"- {warning}")

    if errors:
        print("\n错误：")
        for error in errors:
            print(f"- {error}")


def execute_plan(plans: list[MovePlan], root_dir: Path) -> None:
    content_dir = root_dir / CONTENT_DIR_NAME
    content_dir.mkdir(exist_ok=True)

    for plan in plans:
        if plan.target.exists():
            raise FileExistsError(f"目标目录已存在，拒绝覆盖：{plan.target}")
        shutil.move(str(plan.source), str(plan.target))
        print(f"已迁移：{plan.source} -> {plan.target}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="将文章相关内容目录迁移到 content/ 下")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--dry-run", action="store_true", help="只输出迁移计划，不移动任何目录（默认行为）")
    mode.add_argument("--execute", action="store_true", help="实际执行迁移")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root_dir = Path(__file__).resolve().parents[1]
    plans, warnings, errors = build_plan(root_dir)
    print_plan(plans, warnings, errors, args.execute)

    if errors:
        return 1
    if not args.execute:
        print("\n未执行任何移动。如确认无误，请重新运行并添加 --execute。")
        return 0

    execute_plan(plans, root_dir)
    total_files = sum(plan.file_count for plan in plans)
    total_dirs = sum(plan.dir_count for plan in plans) + len(plans)
    print(f"\n迁移完成：移动顶层目录 {len(plans)} 个，包含文件 {total_files} 个，目录 {total_dirs} 个。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
