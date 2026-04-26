#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_BRANCH="${UPSTREAM_BRANCH:-main}"
PATCH_BASE_REF="${PATCH_BASE_REF:-${UPSTREAM_REMOTE}/${UPSTREAM_BRANCH}}"
ALLOW_UNPATCHED_CHANGES="${ALLOW_UNPATCHED_CHANGES:-0}"

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

info() {
  echo "==> $*"
}

mapfile -t PATCH_FILES < <(find patches -maxdepth 1 -type f -name '[0-9][0-9][0-9]-*.patch' | sort)
if [[ "${#PATCH_FILES[@]}" -eq 0 ]]; then
  fail "patches/ 下没有找到 NNN-*.patch"
fi

info "校验二开文档与 patch 一一对应"
for patch in "${PATCH_FILES[@]}"; do
  base="$(basename "$patch" .patch)"
  doc="docs/customizations/${base}.md"
  [[ -f "$doc" ]] || fail "缺少二开说明文档: $doc"
  grep -Fq "$(basename "$patch")" patches/README.md || fail "patches/README.md 未登记: $(basename "$patch")"
  grep -Fq "$base" docs/customizations/README.md || fail "docs/customizations/README.md 未登记: $base"
done

mapfile -t DOC_FILES < <(find docs/customizations -maxdepth 1 -type f -name '[0-9][0-9][0-9]-*.md' | sort)
for doc in "${DOC_FILES[@]}"; do
  base="$(basename "$doc" .md)"
  patch="patches/${base}.patch"
  [[ -f "$patch" ]] || fail "缺少二开补丁文件: $patch"
done

info "检查当前未提交源码改动是否已同步 patch"
mapfile -t CHANGED_FILES < <(
  {
    git diff --name-only
    git diff --cached --name-only
    git ls-files --others --exclude-standard
  } | sort -u
)

if [[ "${#CHANGED_FILES[@]}" -gt 0 ]]; then
  patch_changed=0
  source_changed=0
  for file in "${CHANGED_FILES[@]}"; do
    case "$file" in
      patches/*.patch)
        patch_changed=1
        ;;
      *.md|docs/*|patches/README.md|AGENTS.md|tools/skills/newapi-upstream-sync/SKILL.md|makefile|scripts/verify_patches.sh|scripts/sync_upstream_local.sh|.github/*|.spec-workflow/*)
        ;;
      *)
        source_changed=1
        ;;
    esac
  done

  if [[ "$source_changed" == "1" && "$patch_changed" != "1" && "$ALLOW_UNPATCHED_CHANGES" != "1" ]]; then
    fail "检测到源码改动但没有 patch 文件改动。若这是非二开改动，请设置 ALLOW_UNPATCHED_CHANGES=1；否则请先更新 patches/NNN-*.patch"
  fi
fi

if ! git rev-parse --verify "$PATCH_BASE_REF" >/dev/null 2>&1; then
  fail "找不到 patch 基准引用: $PATCH_BASE_REF。请先执行 git fetch $UPSTREAM_REMOTE $UPSTREAM_BRANCH"
fi

tmp_dir="$(mktemp -d /tmp/newapi-verify-patches-XXXXXX)"
cleanup() {
  git worktree remove --force "$tmp_dir" >/dev/null 2>&1 || true
}
trap cleanup EXIT

info "在干净基准 ${PATCH_BASE_REF} 上按顺序验证 patch 可应用"
git worktree add --detach "$tmp_dir" "$PATCH_BASE_REF" >/dev/null

for patch in "${PATCH_FILES[@]}"; do
  patch_abs="${REPO_ROOT}/${patch}"
  patch_name="$(basename "$patch")"
  info "验证 ${patch_name}"
  (
    cd "$tmp_dir"
    if git apply --check "$patch_abs" 2>/dev/null; then
      git apply "$patch_abs"
    elif git apply --check --ignore-whitespace "$patch_abs" 2>/dev/null; then
      git apply --ignore-whitespace "$patch_abs"
    elif git apply --check --3way "$patch_abs" 2>/dev/null; then
      git apply --3way "$patch_abs"
    else
      git apply --stat "$patch_abs" || true
      git apply --check "$patch_abs"
    fi
  )
done

info "二开 patch 校验通过"
