# 同步上游最新版并保留本地魔改：实施计划

> 此文件保留执行前的计划清单，未在实施过程中逐项回填。实际集成位置、提交 SHA、补丁重放结果、Docker 矩阵和双前端浏览器验证见 [verification.md](verification.md)。该集成仍在独立分支等待审核，未合并 `main`。

## 1. 建立隔离环境

- [ ] 记录原工作区 `git status`、当前 `main` SHA 和目标上游 SHA。
- [ ] 确认用户现有删除/新增文件保持在原工作区，不进入集成分支。
- [ ] 从当前 `main` 创建 `sync/upstream-7c28993f` 分支和独立 worktree。
- [ ] 在 worktree 中确认工作区干净，并记录回滚检查点。

## 2. 完成本地差异归类

- [ ] 对照 `22e509c1..main` 的全部 155 个文件与 9 个补丁覆盖集合。
- [ ] 将业务代码归入 `001-009`；将 Trellis、知识图谱、接口文档等归为项目元数据/文档。
- [ ] 把 `video_duration.go` 补丁遗漏登记到 `004`。
- [ ] 复核 `docker-image-alpha.yml`、`docker-image-arm64.yml` 是否仍需要；保留则归入 `005`，淘汰则在 `005` 文档说明原因。
- [ ] 若发现无法归类的业务差异，停止实施并更新 PRD。

## 3. 合并固定上游提交

- [ ] 执行无自动提交 merge，将 `7c28993f` 合入集成分支。
- [ ] 冲突文件采用“上游架构 + 定制行为”原则逐个解决。
- [ ] 重点处理 `.github/workflows/*`、`.gitignore`、`AGENTS.md`、`common/init.go`、`makefile`、`model/redemption.go`、测试文件和前端目录迁移。
- [ ] 保留上游双前端、授权系统、系统任务、优雅停机、数据库类型抽象和安全修复。
- [ ] 继续删除 Electron workflow，保护所有项目身份与归属信息。
- [ ] 创建中文上游合并提交。

## 4. 恢复 001-003 后端定制

- [ ] 基于上游兑换事务重做 `001`，保留 API Key 兑换、当前 key 补额和充值记录。
- [ ] 在任务模型保留独立 `token_id` 索引列，任务提交同时写查询归属和私有计费归属。
- [ ] 恢复 `/api/token/redeem`、`/api/task/token/self` 的鉴权与路由顺序。
- [ ] 将上游失败退款的 token 恢复改为 `TASK_REFUND_RESTORE_TOKEN_QUOTA` 控制，默认关闭；不得影响差额结算。
- [ ] 将金额脱敏接入目标上游当前 OpenAI、Claude、异步任务错误边界。
- [ ] 运行 001-003 的定向测试并提交。

## 5. 恢复 004/007/008/009 视频定制

- [ ] 扩展目标上游 `TaskSubmitReq`，保留 `content`、`files` 等请求字段，遵守显式零值规则。
- [ ] 在 Sora 适配器中建立单一参考视频检测入口，并加通道/模型守卫。
- [ ] 恢复默认双倍与可选精确时长计费；复用目标上游受保护下载/SSRF 能力。
- [ ] 恢复 Seedance 顶层视频字段双倍计费，证明计费字段确实透传。
- [ ] 恢复 `seedance-asset` 校验、倍率短路、扩展响应保留和 API Key 删除流程。
- [ ] 恢复 `/api/task/token/asset/delete` 的鉴权、路由顺序和跨 key 隔离。
- [ ] 恢复 unknown/缺失状态归一化、错误优先、结果 URL 成功判定和内容直链。
- [ ] 增加组合测试，覆盖 004/007 不重复计费、008 不叠加倍率、009 不提前退款。
- [ ] 运行 Sora/relay/controller 定向测试并提交。

## 6. 适配 006 双前端锁屏

- [ ] 后端向 `default` 与 `classic` 两份 index 注入同一安全 JSON 配置。
- [ ] 提取共享的锁屏存储、TTL、密码指纹和校验逻辑。
- [ ] 在 `web/default` 增加符合新设计系统的锁屏 gate 和公告加载。
- [ ] 在 `web/classic` 迁移现有 Semi UI 锁屏 gate。
- [ ] 覆盖空密码、特殊字符、两份 index、TTL、密码变化和 localStorage 不可用场景。
- [ ] 分别执行两套前端的 lint/typecheck/build 并提交。

## 7. 重建维护流程与 9 个补丁

- [ ] 以 `7c28993f` 为默认新基准，按编号顺序重建 `patches/001-009.patch`。
- [ ] 更新 9 份定制文档、两份定制 README、同步 skill、makefile 和脚本中的基准/命令。
- [ ] 增强 `scripts/verify_patches.sh`：应用后在临时 worktree 编译并执行定制定向测试。
- [ ] 增加补丁文件覆盖/最终树一致性检查，确保新增源文件不遗漏。
- [ ] 保持每个验证子命令有不超过 120 秒的 timeout。
- [ ] 运行 `make verify-patches` 并提交。

## 8. 静态检查与单元测试

- [ ] 运行 `gofmt` 检查、`go vet`、`go build ./...`。
- [ ] 分包运行 `go test ./...`，每条命令设置 `timeout 120s`，避免单个后台任务无限等待。
- [ ] 运行定制文档列出的全部最小回归命令。
- [ ] 在 `web/` 使用 Bun 安装锁定依赖。
- [ ] 运行 `default` 的 lint、typecheck、build 和 `classic` 的 lint、build、i18n lint。
- [ ] 检查前后端翻译键、受保护版权头和品牌信息未回退。

## 9. 构建 Docker 验证工具

- [ ] 构建当前版本基线镜像和候选镜像，使用独立标签。
- [ ] 新增或更新本地验证脚本/Compose override，禁止复用生产容器名和 volume。
- [ ] 准备代表性旧数据 fixture：用户、token、兑换码、渠道、普通视频任务、资产任务、日志和订阅记录。
- [ ] 准备 mock 上游：视频失败、unknown -> success、资产创建/删除、参考视频 Range/完整下载。
- [ ] 清理逻辑使用 trap，只删除本任务创建的容器、网络和 volume。

## 10. Docker 数据库与升级矩阵

- [ ] SQLite + Redis：空库启动、旧镜像写入、候选镜像升级、数据回读。
- [ ] MySQL 5.7.x + Redis：空库与升级、Token key 列迁移、`tasks.token_id` 索引、数据回读。
- [ ] PostgreSQL 9.6 + Redis：空库与升级、列类型迁移、数据回读。
- [ ] PostgreSQL 15 + Redis + ClickHouse 24.8：日志表、写入、查询和 TTL。
- [ ] 每个组合检查 `/api/status`、迁移日志、容器重启和无 panic/SQL 错误。

## 11. Docker 功能黑盒与浏览器验证

- [ ] 001：API Key 兑换与按 key 任务隔离。
- [ ] 002：开关关闭/开启两轮失败退款，校验钱包、key 额度、任务终态和 mock 命中数。
- [ ] 003：同步与异步错误响应中的金额脱敏。
- [ ] 004/007：参考视频双倍、精确时长、多视频、非视频字段和非白名单模型。
- [ ] 008：资产提交、查询、跨 key 隔离、成功删除、上游业务失败不标记删除。
- [ ] 009：unknown 继续轮询、错误失败、结果 URL 成功、content 直链。
- [ ] 使用浏览器自动化分别验证 `default`/`classic` 的锁屏、公告、错误密码、正确密码、缓存与密码变化。
- [ ] 不加载任何真实供应商凭证；输出真实渠道人工验收清单。

## 12. 收尾检查

- [ ] 运行 `graphify update .`，确认知识图谱更新成功。
- [ ] 再次执行 `make verify-patches`、关键测试和 Docker 健康检查。
- [ ] 审计 `git diff upstream/main...HEAD`，确认只有 9 组定制、明确项目元数据和验证工具差异。
- [ ] 编写升级/备份/回滚说明，强调数据库失败回滚需要恢复快照。
- [ ] 按主题创建中文提交，不推送、不合并 `main`。
- [ ] 向用户报告集成分支、worktree、提交 SHA、通过/失败验证和剩余风险。

## 验证命令基线

以下命令会按实际目录调整，但每条测试命令都使用不超过 120 秒的 timeout：

```bash
timeout 120s make verify-patches
timeout 120s go test ./controller -run '^(TestTokenRedeem|TestGetUserTokenTask|.*Asset.*)$' -count=1
timeout 120s go test ./service -run '^(TestRefundTaskQuota|TestCASGuarded|TestUpdateVideoTasks_FailureRefund)' -count=1
timeout 120s go test ./common ./types ./service -run 'MaskBilling|TaskError' -count=1
timeout 120s go test ./relay/channel/task/sora ./relay/common ./relay -count=1
timeout 120s go test . -run TestInjectFrontendLockPassword -count=1
timeout 120s go build ./...
timeout 120s bun --cwd web/default run build:check
timeout 120s bun --cwd web/classic run build
```

## 回滚点

- 合并前：删除独立 worktree/集成分支即可，原工作区不变。
- 每组定制提交后：回退该主题提交，不改写 `main` 历史。
- Docker 验证：删除任务专属 Compose project 和 volume。
- 生产手册：旧镜像 + 升级前数据库快照；禁止仅回滚镜像后继续使用已迁移数据库。
