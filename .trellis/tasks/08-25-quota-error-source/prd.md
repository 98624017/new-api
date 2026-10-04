# 区分本平台与上游额度不足报错

## Goal

让本地 API Key 额度不足与上游渠道 key 余额不足的错误文案可被人直接区分，同时保持默认行为兼容。

## Requirements

- 新增环境变量 `DISTINGUISH_LOCAL_TOKEN_QUOTA_ERROR`，默认关闭。
- 开启时，仅替换本地 token/key 预扣额度失败的用户可见文本为“本平台 API Key 额度不足，请充值或更换 Key”。
- 关闭时保持现有本地错误文本、错误码和状态码不变。
- 不修改上游渠道错误的文案、错误码、类型或状态码，不新增响应字段。

## Acceptance Criteria

- [x] 开关关闭时，本地 token 额度错误继续返回原始错误文本。
- [x] 开关开启时，本地 token 额度错误返回明确的本平台 API Key 文案。
- [x] 本地错误的错误码与 HTTP 状态码保持不变。
- [x] 相关 Go 回归测试通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
