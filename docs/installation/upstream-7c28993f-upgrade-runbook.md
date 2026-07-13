# 上游 7c28993f 同步版生产升级与回滚手册

本文用于把当前生产版本升级到集成分支 `sync/upstream-7c28993f`。目标上游固定为 `7c28993f6bd9e92616f3f578212577f8b7c40b45`，并保留项目登记的 9 组本地定制。

本文不授权生产部署。执行人必须先在生产数据副本上完成迁移演练，并由业务负责人批准切流窗口。

## 1. 已验证范围与已知限制

本地已使用同一运行时候选镜像验证：

- SQLite、MySQL 5.7、PostgreSQL 9.6、PostgreSQL 15 空库启动及 Redis 连接。
- 当前版本写入代表性用户、Token 和额度数据后，由候选版本复用同一数据卷升级并回读。
- PostgreSQL 15 配合 ClickHouse 24.8 写入审计日志，日志 TTL 为 30 天。
- `tasks.token_id` 列和 `idx_tasks_token_id` 索引在三种主数据库中均存在。
- 9 组定制的本地 mock 黑盒回归，以及 default/classic 双前端锁屏回归。

本地运行时候选镜像为 `newapi-sync:7c28993f-runtime`，镜像 ID 为：

```text
sha256:e65b13bd0c3088212c9d66171610b9e7f26ec186bdfb5d412343ebfad2218853
```

该镜像由已验证的双前端产物和本地 Go 二进制封装进项目相同 Debian 运行时基底。生产 `Dockerfile` 的冷构建在当前仅有 legacy builder 的主机上未能在单次 120 秒门限内完成，因此生产发布前还必须在正式 CI/buildx 环境完成一次从源码冷构建、记录镜像 digest，并对该 digest 重跑本文的副本演练。不得把“运行时镜像已通过”误写成“生产 Dockerfile 冷构建已通过”。

## 2. 强制前置条件

升级前必须同时满足：

- 固定候选镜像 digest，禁止使用会漂移的 `latest` 标签。
- 记录旧镜像 digest、当前环境变量、Compose/编排配置和数据库版本。
- 确认主数据库、日志数据库和持久卷有足够的备份空间。
- 在生产快照副本上完成一次候选版本启动、迁移、重启和数据回读。
- 准备旧镜像、数据库恢复命令和独立的回滚验证入口。
- 设置维护窗口或先摘除写流量，保证最终备份期间没有新充值、兑换、任务或额度变更。
- 不向验证容器注入真实供应商凭证；需要真实渠道验收时使用专用低额度渠道并单独批准。

## 3. 数据备份

### 3.1 SQLite

SQLite 必须在停止所有应用副本后备份。复制数据库文件时同时检查同目录的 `-wal` 和 `-shm` 文件，优先使用 SQLite 在线备份命令生成单文件快照：

```bash
sqlite3 /data/new-api.db ".backup '/backup/new-api-pre-7c28993f.db'"
sqlite3 /backup/new-api-pre-7c28993f.db "PRAGMA integrity_check;"
```

完整记录 `/data` 持久卷的挂载路径和文件校验和。

### 3.2 MySQL 5.7

MySQL 5.7 服务必须显式使用 `utf8mb4`，本地验证使用：

```text
--character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
```

升级前创建一致性逻辑备份，并验证可恢复到独立实例：

```bash
mysqldump --single-transaction --routines --triggers --events \
  --hex-blob --default-character-set=utf8mb4 \
  -h DB_HOST -u DB_USER -p DB_NAME > new-api-pre-7c28993f.sql
```

恢复演练后确认数据库、表和连接字符集均为 `utf8mb4`。如果现网不是 `utf8mb4`，本次升级窗口不得顺带执行字符集迁移，应先单独评估和演练。

### 3.3 PostgreSQL

PostgreSQL 使用自定义格式备份，并恢复到与生产主版本相同的独立实例：

```bash
pg_dump -Fc -h DB_HOST -U DB_USER -d DB_NAME \
  -f new-api-pre-7c28993f.dump
pg_restore --list new-api-pre-7c28993f.dump >/dev/null
```

PostgreSQL 9.6 已停止上游维护，查询 `information_schema` 约束信息明显慢于新版本。应把迁移到受支持的 PostgreSQL 版本作为独立后续任务，不要与本次应用升级合并执行。

### 3.4 ClickHouse 与 Redis

- ClickHouse 如果属于审计保留范围，升级前按现有备份方案创建快照并记录 `LOG_SQL_CLICKHOUSE_TTL_DAYS`。本地已验证 TTL 为 30 天。
- Redis 在本项目中不是主业务数据的替代品，但仍需记录版本、连接参数和持久化策略。回滚时不得恢复一个晚于主数据库快照、且包含不兼容业务状态的 Redis 快照。

## 4. 副本演练

1. 从最终生产备份恢复一套隔离数据库，使用与生产相同的数据库主版本和配置。
2. 使用旧镜像启动并读取管理员、用户数、Token 数、可用额度和近期任务数，保存基线。
3. 停止旧镜像，使用候选镜像连接同一副本数据库和 Redis。
4. 等待 `/api/status` 成功，检查启动日志不存在 panic、迁移失败或 SQL 错误。
5. 重启候选容器，再次确认健康和数据回读。
6. 验证 default/classic 两套前端、管理员登录、Token 列表、任务列表和 `/v1/models`。`/v1/models` 不应产生供应商费用。
7. 使用测试账号和本地 mock 复跑 9 组定制定向验收，不使用生产用户或真实计费渠道。

数据库结构检查：

```sql
-- SQLite
PRAGMA table_info(tasks);
PRAGMA index_list(tasks);

-- MySQL
SHOW COLUMNS FROM tasks LIKE 'token_id';
SHOW INDEX FROM tasks WHERE Key_name = 'idx_tasks_token_id';

-- PostgreSQL
SELECT column_name FROM information_schema.columns
WHERE table_name = 'tasks' AND column_name = 'token_id';
SELECT indexname FROM pg_indexes
WHERE tablename = 'tasks' AND indexname = 'idx_tasks_token_id';
```

## 5. 健康门限与切流规则

普通数据库组合使用至少 120 秒的部署健康等待。PostgreSQL 9.6 必须使用至少 180 秒的启动宽限：

```yaml
healthcheck:
  test: ["CMD-SHELL", "wget -q -O - http://localhost:3000/api/status | grep -o '\"success\":\\s*true' || exit 1"]
  start_period: 180s
  interval: 10s
  timeout: 5s
  retries: 6
```

本地观测结果：

- PostgreSQL 9.6 空库或普通重启约 97 秒。
- 旧数据升级到候选版本约 139.473 秒。
- 120 秒时应用尚未 ready，但最终正常启动、状态码为 200，且数据完整。

因此，PostgreSQL 9.6 在 120 秒未 ready 时不得立即判定迁移失败，也绝对不得提前切流。必须继续隔离等待到 180 秒门限，并检查日志。180 秒仍未健康，或期间出现 panic/SQL 错误，按停止条件回滚。

所有数据库都遵守同一切流条件：只有 `/api/status` 成功、结构检查通过、基线数据一致、容器重启后仍健康，才允许把流量切到候选版本。

## 6. 生产切换步骤

1. 摘除旧副本流量，等待正在处理的请求结束，停止新的充值、兑换和任务写入。
2. 创建最终一致性数据库快照，记录备份时间、校验和和旧镜像 digest。
3. 只启动一个候选副本，不连接外部流量。
4. 按第 5 节等待健康，并检查完整启动日志和数据库结构。
5. 对管理员、用户、Token、额度和任务数量做升级前后比对。
6. 完成无费用烟雾检查：两套前端、管理员登录、Token 查询、任务查询、`/v1/models`。
7. 逐步加入流量；先观察错误率、登录、任务提交/轮询和额度变更，再扩容其余副本。
8. 在观察窗口结束前保留旧镜像和升级前数据库快照，不执行清理。

## 7. 立即停止条件

出现任一情况都不得切流，已经切流则立即摘除候选副本：

- 启动日志出现 panic、数据库迁移失败、重复 SQL 错误或连接配置错误。
- 普通数据库 120 秒仍未健康；PostgreSQL 9.6 在 180 秒仍未健康。
- `tasks.token_id` 或 `idx_tasks_token_id` 缺失。
- 管理员、用户、Token、额度、任务等基线数据不一致。
- 候选容器首次成功但重启后失败。
- default 或 classic 前端无法访问，或配置了锁屏密码却可以绕过锁屏。
- 充值、兑换、失败退款、任务归属或视频计费出现与定制文档不一致的结果。
- 必须临时修改生产数据才能让升级继续。

## 8. 回滚步骤

候选版本启动时会执行 GORM `AutoMigrate`，并可能修改表结构。即使尚未切流，只要候选版本已经连接生产数据库，就不能假定数据库仍与旧镜像完全兼容。

标准回滚顺序：

1. 摘除并停止全部候选副本，阻止继续写入。
2. 保存候选启动日志、数据库错误和失败时间点，供事后分析。
3. 删除或隔离迁移后的数据库实例，恢复升级前最终一致性快照。
4. 如 ClickHouse 结构或 TTL 也发生变更且审计日志必须保持一致，同时恢复对应快照。
5. 使用记录的旧镜像 digest 连接恢复后的数据库和原配置启动。
6. 验证 `/api/status`、管理员登录、Token、额度、任务和前端，再恢复流量。
7. 比对维护窗口内是否存在未进入最终快照的外部支付回调或供应商任务，按业务台账人工处理。

禁止只把镜像标签改回旧版本后继续使用已经迁移的数据库。数据库迁移失败或行为不兼容时，必须恢复数据库快照与旧镜像这一整套状态。

## 9. 发布记录

每次演练和生产切换至少记录：

- 集成分支提交 SHA、候选镜像 digest、旧镜像 digest。
- 主数据库、Redis、ClickHouse 的版本和配置摘要。
- 备份文件、校验和、恢复演练结果和负责人。
- 应用首次健康耗时、重启健康耗时和完整迁移日志位置。
- 结构检查、数据基线、9 组定制验收和双前端验收结果。
- 是否触发停止条件、是否回滚，以及实际恢复点。
