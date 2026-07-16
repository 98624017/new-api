# 同步验证记录

## 集成位置

- 分支：`sync/upstream-7c28993f`
- Worktree：`/home/feng/project/new-api-sync-7c28993f`
- 提交：`d698c94b186e61111c5fa371408aae3113d54295`
- 状态：未推送，未合并 `main`，等待用户审核。

## 2026-07-16 复核结果

### 补丁与定向回归

在集成 worktree 执行 `make verify-patches` 并通过。该命令在干净的
`7c28993f` 基线上顺序重放 001-009 补丁，检查重放树与集成树一致，并通过：

- shared/default 前端锁屏单测；
- default 与 classic 前端构建；
- Go 全量编译；
- 001-009 定制定向 Go 测试。

完整命令输出保存在本机：
`/tmp/newapi-sync-audit-20260716/verify-patches.log`。

### Docker 数据库矩阵

使用仅带 `newapi-audit-20260716-*` 前缀的容器、网络和数据目录执行，结束后已清理。

| 场景 | 结果 | 覆盖内容 |
| --- | --- | --- |
| SQLite + Redis | 通过 | 基线镜像创建持久数据目录，候选镜像复用该目录启动并重启健康。 |
| MySQL 5.7 + Redis | 通过 | 基线建库、候选升级、`tasks.token_id` 与 `idx_tasks_token_id`、候选重启健康。 |
| PostgreSQL 9.6 + Redis | 通过 | 基线建库、候选升级、`tasks.token_id` 与 `idx_tasks_token_id`、候选重启健康。 |
| PostgreSQL 15 + Redis + ClickHouse 24.8 | 通过 | 候选启动、主库结构、ClickHouse `logs` 表与 30 天 TTL、候选重启健康。 |

首次 MySQL 与 ClickHouse 启动曾因数据库 TCP 服务尚未稳定而失败；改为实际 SQL/HTTP 就绪探测并额外等待后复跑通过。该过程及最终结果完整保存在：
`/tmp/newapi-sync-audit-20260716/docker-matrix.log`。

### 双前端锁屏

使用浏览器自动化在候选运行时镜像中分别验证 `classic` 和 `default` 前端：

- 均显示锁屏输入框；
- 错误密码保持锁定；
- 正确密码成功解锁。

截图保存在：

- `/tmp/newapi-sync-audit-20260716/classic-lock.png`
- `/tmp/newapi-sync-audit-20260716/default-lock.png`

## 剩余验证边界

本次 Docker 矩阵验证了数据库启动、迁移、关键结构、日志 TTL 和重启健康。代表性业务数据回读以及视频 mock 黑盒链路未在容器矩阵中单独重跑，仍由 `make verify-patches` 中的定向回归覆盖。生产升级前仍必须按升级手册在生产数据副本上执行备份和迁移演练。
