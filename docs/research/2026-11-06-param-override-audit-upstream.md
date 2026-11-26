# 上游参数覆盖审计 `po` 的后续变化

查证时间：2026-11-06 23:31（Asia/Shanghai）。仅调研官方来源，未同步、合并、推送或修改业务代码。

截至最新已发布 **v1.0.0-rc.48** 和本次固定的 `main`，参数覆盖审计仍保存完整操作值，未发现独立关闭 `po` 全文、截断、摘要或减少其记录范围的实现。后续确有审计范围扩充和性能优化，但不能据此预期升级后数据库备份因 `po` 变小。[启用条件][audit-enable]、[值格式化][audit-value]、[写入字段][po-write]、[发布版本][release]

## 版本与查证范围

- 本地锁定上游基线：[7c28993f6bd9e92616f3f578212577f8b7c40b45](https://github.com/QuantumNous/new-api/commit/7c28993f6bd9e92616f3f578212577f8b7c40b45)，2026-07-12 00:24:17 +08:00；本地 HEAD 为 `9f60002e165c2bba824d4e5286989c7b07f0eeb9`。
- 最新已发布版本：[v1.0.0-rc.48][release]，2026-11-04 15:40:15 UTC 发布，对应 [706144769e5bed5a47b78721e3bb11da3cf4c0b9](https://github.com/QuantumNous/new-api/commit/706144769e5bed5a47b78721e3bb11da3cf4c0b9)。
- 调研固定 `main`：[e03439033238722c0b8d4437a052445f3aeeed83](https://github.com/QuantumNous/new-api/commit/e03439033238722c0b8d4437a052445f3aeeed83)。以下判断不依赖后续移动的分支引用。
- 使用已认证 `gh api` 读取上述两个版本的相关源码，并查询基线后 `relay/common/override.go`、`service/log_info_generate.go`、`model/log.go` 的提交列表；对下表相关提交检查 diff、关联 PR 和合并状态。用 GitHub Compare 验证下表四项均为 rc.48 的祖先，均返回 `behind_by: 0`。
- PR/issue 检索覆盖 `param override`、`ParamOverrideAudit`、`audit`、`sensitive`、`po + truncate/截断`、`参数覆盖 + 日志/数据库`、`log + retention`、`日志 + 膨胀` 等组合。检索无命中不代表所有历史讨论中绝无相关表述；源码判断以固定提交为准。

## 已发布的相关调整

日期统一使用提交时间 UTC；以下均已进入 rc.48，也包含于本次固定 `main`。

| 日期 | 提交、PR | 实际调整及对体积的意义 |
| --- | --- | --- |
| 2026-08-10 | [eab18a835791](https://github.com/QuantumNous/new-api/commit/eab18a83579187f880139894dd9e7f06d1a492ce)、[#6641](https://github.com/QuantumNous/new-api/pull/6641) | 同步覆盖后的推理强度到日志，并新增 `reasoning`、`reasoning_effort`、`output_config`、`generationConfig.thinkingConfig`、`generation_config.thinking_config` 审计路径前缀。PR 正文明确写有“参数覆盖推理强度增加审计”；这是扩大审计范围。 |
| 2026-09-03 | [057f71c2336c](https://github.com/QuantumNous/new-api/commit/057f71c2336c3981187b732a9d06f65490e9a946) | 日志元数据改用 `LogOther` 区分权限；`other["po"] = ...` 改成 `other.SetPublic("po", ...)`。不是删除或截断 `po`；关联 PR 查询返回空列表。 |
| 2026-10-27 | [01da3a55f838](https://github.com/QuantumNous/new-api/commit/01da3a55f838cfa21010b05c7451c0a42cf5a173)、[#8061](https://github.com/QuantumNous/new-api/pull/8061) | 缓存参数覆盖使用的已编译正则，降低重复编译开销；没有修改审计值格式化或持久化策略。 |
| 2026-11-01 | [70e6af642982](https://github.com/QuantumNous/new-api/commit/70e6af6429824f99de98e96e65254c8c2ef7ee0b)、[#8148](https://github.com/QuantumNous/new-api/pull/8148) | 参数覆盖在同一个 `ojson.Doc` 上编辑，最后生成一次请求体，减少请求处理中的复制和重编码；审计调用仍保留，`po` 全文存储没有减量。 |

祖先关系复核示例：[eab18a8 → rc.48](https://github.com/QuantumNous/new-api/compare/eab18a83579187f880139894dd9e7f06d1a492ce...706144769e5bed5a47b78721e3bb11da3cf4c0b9)、[057f71c → rc.48](https://github.com/QuantumNous/new-api/compare/057f71c2336c3981187b732a9d06f65490e9a946...706144769e5bed5a47b78721e3bb11da3cf4c0b9)、[01da3a5 → rc.48](https://github.com/QuantumNous/new-api/compare/01da3a55f838cfa21010b05c7451c0a42cf5a173...706144769e5bed5a47b78721e3bb11da3cf4c0b9)、[70e6af6 → rc.48](https://github.com/QuantumNous/new-api/compare/70e6af6429824f99de98e96e65254c8c2ef7ee0b...706144769e5bed5a47b78721e3bb11da3cf4c0b9)。

## rc.48 与固定 main 的实际行为

两份源码在以下关键逻辑上一致；链接固定到已发布的 rc.48。

1. `DebugEnabled` 会开启审计；非 Debug 时，参数覆盖触及指定敏感路径也会开启。没有独立的 `po` 开关。只关闭 Debug 不能阻止敏感路径的审计。[白名单][audit-paths]、[启用判断][audit-enable]
2. 审计值为字符串时原样返回，其他值完整 JSON 序列化；`prepend` 等操作把该值放进审计文本，没有长度上限、摘要或截断。[值格式化][audit-value]、[操作文本][audit-prepend]
3. 非空审计直接写入 `other.SetPublic("po", relayInfo.ParamOverrideAudit)`；`SetPublic` 没有限长，`JSONString` 完整序列化，消费日志再将其入库。[字段写入][po-write]、[SetPublic][set-public]、[序列化][other-json]、[入库][consume-store]
4. `po` 不在用户日志投影删除的权限字段列表中。权限投影是读取结果的格式化，不是删除数据库里的原值。[日志投影][projection]
5. `LogConsumeEnabled` 是整条消费日志的开关，不能用来只关闭 `po` 全文；它会同时影响消费日志记录。[消费日志入口][consume-enabled]
6. 日志清理任务是手动启动的 `non-scheduled` 任务；新增日志设置中的 `MaxRangeDays` / `max_range_days` 限制查询、导出跨度，不是日志保存期限或定时清理。定时任务注册也未包含日志清理。[清理任务](https://github.com/QuantumNous/new-api/blob/e03439033238722c0b8d4437a052445f3aeeed83/service/system_task.go#L76)、[定时任务注册](https://github.com/QuantumNous/new-api/blob/e03439033238722c0b8d4437a052445f3aeeed83/controller/system_task_handlers.go#L19)、[日志设置](https://github.com/QuantumNous/new-api/blob/e03439033238722c0b8d4437a052445f3aeeed83/setting/operation_setting/log_setting.go)

## 未合并提案与讨论

- [#6566](https://github.com/QuantumNous/new-api/pull/6566) 仍为 open，`merged_at: null`；正文也是同步推理强度并增加审计范围，不是减小 `po`。
- [#5371](https://github.com/QuantumNous/new-api/pull/5371) 已关闭、`merged_at: null`；提议的 `LogRetentionDays` 是阻止删除保留窗口内日志的保护，不是自动清理。不能把此 PR 当作已发布配置。
- [#6400](https://github.com/QuantumNous/new-api/pull/6400) 提到自动清理，但已关闭、`merged_at: null`；不能从标题推断已合并。
- [#5342](https://github.com/QuantumNous/new-api/issues/5342) 在基线前已提出数据库日志堆积；关闭前的[回复](https://github.com/QuantumNous/new-api/issues/5342#issuecomment-4808760067)建议最新版及迁移日志到 ClickHouse。该回复不构成 `po` 截断或自动清理已实现的证据。
- 本次检索未找到明确针对 `po` 体积的已合并修复或专用减量提案；[`"po" + "truncate"`](https://github.com/QuantumNous/new-api/issues?q=%22po%22+%22truncate%22)、[`"po" + "截断"`](https://github.com/QuantumNous/new-api/issues?q=%22po%22+%22%E6%88%AA%E6%96%AD%22)在查证时均为零命中。此结论限于上述源码、提交及关键词检索范围。

## 对升级与既有数据的影响

单纯升级到 rc.48，或升级到本次固定 `main`，不能解决 `po` 全文带来的增量；推理路径审计扩大后，部分请求还可能新增审计记录。性能优化主要改变请求处理成本，不会把 `po` 改成摘要。[#6641](https://github.com/QuantumNous/new-api/pull/6641)、[#8061](https://github.com/QuantumNous/new-api/pull/8061)、[#8148](https://github.com/QuantumNous/new-api/pull/8148)、[当前写入][po-write]

上述调整没有历史 `po` 回写、缩短或删除流程；读取日志的权限投影也不修改持久化原值。因此已有 `logs.other` 不会因这些升级自动缩小，包含这些数据的备份仍保留原有负担。减少既有数据需要另行制定并执行日志清理或数据迁移方案，本次没有执行此类操作。[权限投影][projection]、[消费日志入库][consume-store]、[手动清理任务](https://github.com/QuantumNous/new-api/blob/e03439033238722c0b8d4437a052445f3aeeed83/service/system_task.go#L76)

[release]: https://github.com/QuantumNous/new-api/releases/tag/v1.0.0-rc.48
[audit-paths]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/relay/common/override.go#L48
[audit-enable]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/relay/common/override.go#L346
[audit-value]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/relay/common/override.go#L421
[audit-prepend]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/relay/common/override.go#L467
[po-write]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/service/log_info_generate.go#L148
[set-public]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/model/log_other.go#L62
[other-json]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/model/log_other.go#L173
[projection]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/model/log_other.go#L247
[consume-enabled]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/model/log.go#L339
[consume-store]: https://github.com/QuantumNous/new-api/blob/706144769e5bed5a47b78721e3bb11da3cf4c0b9/model/log.go#L384
