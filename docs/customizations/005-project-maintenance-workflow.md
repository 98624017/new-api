# 005-project-maintenance-workflow

## 1. 背景

当前项目除了业务二开外，还维护了一组项目级工程化差异，用于稳定同步上游、验证本地补丁、保留本地构建和回归测试流程。

这些差异不属于单个业务接口，但属于“当前项目基准 + 本地补丁 = 当前现状代码”的必要组成部分，因此单独登记为项目维护补丁。

## 2. 目标

- 保留本地上游同步脚本和 GitHub Actions。
- 保留二开补丁校验脚本。
- 保留本地 README / AGENTS / makefile 中的项目维护说明。
- 保留 multipart 请求体回归修复及测试。
- 避免这些维护性差异散落在工作区、无法通过补丁重放。

不解决：

- 不改变业务接口行为。
- 不新增新的上游同步策略。
- 不替代 `001-004` 的业务补丁文档。

## 3. 影响范围

- `.github/workflows/docker-image-manual-ghcr.yml`
  - 保留手动 GHCR Docker 构建流程。
- `.github/workflows/sync-upstream.yml`
  - 保留上游同步 workflow。
  - workflow 运行时先把当前分支的 `patches/*.patch` 复制到临时目录，再从 `upstream/<branch>` 创建同步分支并应用临时目录中的补丁，避免在已打补丁分支上重复应用补丁。
- `.gitignore`
  - 保留本地生成物忽略规则，包含 graphify 输出、`.tmp-newapi-verify` 和 `meituapi/` 等本地验证/素材产物。
- `AGENTS.md`
  - 保留项目内 agent 工作约定。
- `README.md` / `README.zh_CN.md`
  - 保留本地维护说明入口。
- `makefile`
  - 保留 `verify-patches` 等本地维护命令。
- `relay/common/relay_utils.go`
  - 保留 multipart 请求体处理回归修复。
- `relay/common/relay_utils_test.go`
  - 覆盖 multipart 请求体回归测试。
- `scripts/sync_upstream_local.sh`
  - 本地上游同步脚本。
- `scripts/verify_patches.sh`
  - 二开补丁配对和可重放校验脚本。
- `tools/skills/newapi-upstream-sync/SKILL.md`
  - 本地上游同步 skill 说明。

## 4. 风险点

- 该补丁覆盖项目维护文件，后续上游同步时容易与 CI、README、构建脚本变更冲突。
- `scripts/verify_patches.sh` 默认基准锁定为当前项目使用的原版 new-api commit，切换上游基准时需显式设置 `PATCH_BASE_REF`。
- multipart 回归修复位于通用 relay 工具函数，需避免影响非 multipart 请求体处理。
- GitHub Actions 上游同步 workflow 依赖当前分支的 `patches/*.patch` 作为临时补丁源；同步分支切到 upstream 后，不能再从工作区 `patches/` 读取补丁。

## 5. 测试方案

最小验证命令：

```bash
make verify-patches
go test ./relay/common -run TestValidateBasicTaskRequest_MultipartWithMetadata -count=1
```

完整验证：

```bash
make verify-patches
go test ./relay/common -count=1
```

## 6. 升级关注点

- 上游若重构 Docker workflow，需要手动复核本地 GHCR workflow 是否仍需要保留。
- Docker workflow 中固定 SHA 的第三方 Action 需要定期复核；`cosign-installer` 应保持在支持当前 GitHub runner 和 cosign release 下载重试的版本，避免单架构签名安装失败导致多架构 manifest 不更新。
- 上游同步 workflow 若调整分支创建方式，需要确认同步分支仍基于 upstream 干净分支，而不是当前已打补丁分支。
- 上游若重构 relay request body 读取逻辑，需要重新确认 multipart 回归测试仍覆盖真实风险。
- 上游同步基准更新时，应同步更新 `scripts/verify_patches.sh` 的默认 `PATCH_BASE_REF_DEFAULT`。
