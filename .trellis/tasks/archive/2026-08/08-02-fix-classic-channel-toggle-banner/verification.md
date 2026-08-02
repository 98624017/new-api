# 验证记录

## 定向验证

- `go test ./controller -run '^(TestUpdateChannelRejectsStatusField|TestChannelStatusValidation)$' -count=1`：通过，2 项测试。
- `bun test web/classic/src/hooks/channels/channelPagination.test.js web/classic/src/services/channel.test.js`：通过，状态合同与分页边界共 5 项测试、9 个断言。
- `bunx prettier --check ...`：本次修改的 6 个 Classic 源码/测试文件格式通过。
- `bunx eslint ...`：本次修改的 6 个 Classic 源码/测试文件无错误。
- `bun run --cwd web/classic build`：通过。
- `git diff --check`：通过。

## 补丁链门禁

`timeout 120s make verify-patches` 最终完整通过：

- 从锁定基线顺序应用 `001-011` patch
- patch 所属文件与当前集成树逐字一致
- 16 项前端定向测试通过
- Default 与 Classic 双前端构建通过
- Go 全量编译通过
- 001-011 对应的后端定向回归通过

前两次运行在 Go 冷缓存依赖下载/编译阶段达到 120 秒上限；使用门禁相同的 `/tmp` Go 缓存完成一次 `go build ./...` 预热后，原命令在规定超时内完整通过。

## Graphify

已运行 `graphify update .`。2,066 个代码文件的 AST 扫描完成，但当前 Graphify 版本在 11,287 个节点时仍尝试生成 HTML，并因节点规模上限退出；提示的 `--no-viz` 对 `update` 子命令未生效。该步骤未产生知识图谱文件差异。

## 覆盖说明

本次未启动带管理员登录态的浏览器 E2E。请求合同由新增的 Classic API 单元测试锁定，分页单元测试覆盖总数缩减与空结果边界，后端旧接口拒绝与状态值边界由控制器测试锁定，完整补丁重放与 Classic 生产构建均已通过。
