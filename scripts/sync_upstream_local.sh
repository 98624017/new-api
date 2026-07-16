#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_BRANCH="${UPSTREAM_BRANCH:-main}"
TARGET_BRANCH="${TARGET_BRANCH:-$(git branch --show-current)}"
PUSH_AFTER_SYNC="${PUSH_AFTER_SYNC:-0}"
SKIP_TESTS="${SKIP_TESTS:-0}"

usage() {
  cat <<'EOF'
用法：
  bash scripts/sync_upstream_local.sh

可选环境变量：
  UPSTREAM_REMOTE=upstream        上游 remote，默认 upstream
  UPSTREAM_BRANCH=main            上游分支，默认 main
  TARGET_BRANCH=main              目标分支，默认当前分支
  PUSH_AFTER_SYNC=1               验证通过后自动 push
  SKIP_TESTS=1                    跳过回归测试

推荐：
  make sync-upstream-local
  PUSH_AFTER_SYNC=1 make sync-upstream-local
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "检测到未提交改动，先清理工作区后再同步："
  git status --short
  exit 1
fi

if ! git remote get-url "$UPSTREAM_REMOTE" >/dev/null 2>&1; then
  echo "未找到上游 remote: $UPSTREAM_REMOTE"
  exit 1
fi

if [[ -z "$TARGET_BRANCH" ]]; then
  echo "无法识别当前分支"
  exit 1
fi

echo "==> 切换到目标分支: $TARGET_BRANCH"
git checkout "$TARGET_BRANCH"

echo "==> 拉取上游最新代码"
git fetch "$UPSTREAM_REMOTE" --prune

MERGE_TARGET="$UPSTREAM_REMOTE/$UPSTREAM_BRANCH"
if ! git rev-parse --verify "$MERGE_TARGET" >/dev/null 2>&1; then
  echo "未找到上游分支: $MERGE_TARGET"
  exit 1
fi

echo "==> 计算分叉状态"
git rev-list --left-right --count "${MERGE_TARGET}...HEAD"
echo
git log --oneline --max-count=8 "$MERGE_TARGET"
echo

BACKUP_BRANCH="backup_upstream_sync_$(date +%Y%m%d_%H%M%S)"
echo "==> 创建备份分支: $BACKUP_BRANCH"
git branch "$BACKUP_BRANCH"

echo "==> 合并上游分支: $MERGE_TARGET"
git merge --no-edit "$MERGE_TARGET"

if [[ "$SKIP_TESTS" != "1" ]]; then
  echo "==> 验证补丁重放、编译和全部本地定制"
  bash scripts/verify_patches.sh
  echo
fi

echo "==> 当前状态"
git status --short --branch
echo
git log --oneline --decorate --max-count=5
echo

if [[ "$PUSH_AFTER_SYNC" == "1" ]]; then
  echo "==> 推送到 origin/$TARGET_BRANCH"
  git push origin "$TARGET_BRANCH"
else
  echo "==> 未自动推送。确认无误后手动执行："
  echo "git push origin $TARGET_BRANCH"
fi
