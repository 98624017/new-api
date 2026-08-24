# 013-hide-credit-system-prompts

适配上游基线：`7c28993f6bd9e92616f3f578212577f8b7c40b45`。

## 1. 背景

渠道参数覆盖审计字段 `other.po` 会记录参数覆盖操作。某些 LLM 请求的覆盖内容包含 NewAPI 注入的 system message；该字段通过 API key 自助日志接口返回时会泄露渠道核心提示词。

## 2. 目标

- 普通用户和 API key 自助日志不返回 `other.po`。
- 保留额度、token、退款和公开计费元数据。
- dashboard 在前端再次过滤 `po`，兼容后端旧响应或缓存数据。
- 管理员日志仍可查看完整参数覆盖审计，方便排障。

## 3. 实现范围

- `model.formatUserLogs` 在普通用户日志序列化时删除 `po`；该函数覆盖 `/api/log/token` 和用户自查日志，管理员 `GetAllLogs` 不受影响。
- `task-dashboard/src/components/CreditRecordsPage.tsx` 的 `parseCreditLogOther` 丢弃 `po` 后再交给退款识别和详情渲染。
- 后端和 dashboard 各有一条回归测试，验证 `po` 被删除且普通计费字段保留。

## 4. 兼容性与风险

- 只在读取/展示边界过滤，不修改数据库历史记录，不改变 relay 注入逻辑，不需要迁移。
- `po` 是内部审计字段，客户不应依赖；管理员查询行为保持不变。
- 若未来新增同类敏感审计键，需要在用户日志格式化器和 dashboard 解析器同步加入明确的删除规则。

## 5. 回归验证

```bash
go test ./model -run '^TestFormatUserLogs' -count=1
bun run test
bun run build
make verify-patches
```
