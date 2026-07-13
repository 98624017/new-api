#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_BRANCH="${UPSTREAM_BRANCH:-main}"
PATCH_BASE_REF_DEFAULT="${PATCH_BASE_REF_DEFAULT:-7c28993f6bd9e92616f3f578212577f8b7c40b45}"
PATCH_BASE_REF="${PATCH_BASE_REF:-${PATCH_BASE_REF_DEFAULT}}"
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
  fail "找不到 patch 基准引用: $PATCH_BASE_REF。请确认当前仓库包含项目锁定的原版 new-api 基准，或显式设置 PATCH_BASE_REF"
fi

tmp_dir="$(mktemp -d /tmp/newapi-verify-patches-XXXXXX)"
cleanup() {
  git worktree remove --force "$tmp_dir" >/dev/null 2>&1 || true
}
trap cleanup EXIT

declare -A PATCHED_PATHS=()

info "在干净基准 ${PATCH_BASE_REF} 上按顺序验证 patch 可应用"
git worktree add --detach "$tmp_dir" "$PATCH_BASE_REF" >/dev/null

for patch in "${PATCH_FILES[@]}"; do
  patch_abs="${REPO_ROOT}/${patch}"
  patch_name="$(basename "$patch")"
  info "验证 ${patch_name}"
  while IFS=$'\t' read -r _ _ path; do
    [[ -n "$path" ]] && PATCHED_PATHS["$path"]=1
  done < <(git apply --numstat "$patch_abs")
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

info "检查 patch 重放结果与当前集成树一致"
while IFS= read -r path; do
  current_path="${REPO_ROOT}/${path}"
  replay_path="${tmp_dir}/${path}"
  if [[ -e "$current_path" || -L "$current_path" ]]; then
    [[ -e "$replay_path" || -L "$replay_path" ]] || fail "重放树缺少 patch 文件: $path"
    cmp -s "$current_path" "$replay_path" || fail "重放树与当前集成树不一致: $path"
  elif [[ -e "$replay_path" || -L "$replay_path" ]]; then
    fail "当前集成树已删除文件，但重放树仍存在: $path"
  fi
done < <(printf '%s\n' "${!PATCHED_PATHS[@]}" | sort)

mkdir -p /tmp/go-build-cache /tmp/go-tmp /tmp/gomodcache

run_replay_check() {
  local label="$1"
  shift
  info "$label"
  (
    cd "$tmp_dir"
    env GOCACHE=/tmp/go-build-cache \
      GOTMPDIR=/tmp/go-tmp \
      GOMODCACHE=/tmp/gomodcache \
      timeout 120s "$@"
  )
}

command -v bun >/dev/null 2>&1 || fail "未找到 bun，无法验证双前端补丁"
run_replay_check "安装 patch 重放树的前端依赖" bun install --cwd web --frozen-lockfile
run_replay_check "验证共享锁屏状态逻辑" bun test web/shared/frontend-lock.test.ts
run_replay_check "编译 default 前端" bun run --cwd web/default build:check
run_replay_check "编译 classic 前端" bun run --cwd web/classic build
run_replay_check "编译 patch 重放后的 Go 项目" go build ./...
run_replay_check "验证 001 API Key 自助能力" \
  go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask|TestDeleteUserTokenAsset)' -count=1
run_replay_check "验证 002 失败退款开关" \
  go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded|TestUpdateVideoTasks_FailureRefund)' -count=1
run_replay_check "验证 003 金额脱敏" \
  go test ./common ./service ./types -run '(MaskBillingAmounts|MasksBillingAmounts)' -count=1
run_replay_check "验证 004/007/008/009 Sora 与 Seedance 定制" \
  go test ./relay/channel/task/sora ./controller -run '(ReferenceVideo|Seedance|Asset|Unknown|MissingVideoStatus|MissingStatus|StoredResultURL)' -count=1
run_replay_check "验证 005 multipart 回归" \
  go test ./relay/common -run '^TestValidateBasicTaskRequest_MultipartWithMetadata$' -count=1
run_replay_check "验证 006 双前端配置注入" \
  go test . -run '^TestInjectFrontendLockPassword' -count=1

info "二开 patch 重放、编译与定制回归校验通过"
