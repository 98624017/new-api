# Technical Design

## Boundary

Use the existing `model.formatUserLogs` response sanitizer as the server-side boundary. It already runs for `GetLogByTokenId` (the API-key endpoint) and `GetUserLogs`, while administrator `GetAllLogs` does not call it. Extend the existing redaction list with the audit-only `po` key.

The frontend has a single parser, `parseCreditLogOther`, before both refund classification and detail rendering. Remove `po` from the parsed record there so dashboard rendering has a second independent guard without changing the API response contract used for billing fields.

## Data Flow

```text
ChannelSetting.SystemPrompt
  -> relay param override audit (RelayInfo.ParamOverrideAudit)
  -> other.po in consume log
  -> model.GetLogByTokenId
  -> formatUserLogs: delete po
  -> /api/log/token response
  -> task-dashboard parseCreditLogOther: delete po
  -> CreditRecordsPage detail badges
```

`po` is an internal parameter-override audit record, not a billing input. Removing the whole key is deterministic and avoids trying to parse or redact arbitrary nested prompt text. Public billing fields (`model_price`, ratios, cache counters, request path, refund markers) remain unchanged.

## Compatibility

- Existing rows are protected at read time; no migration or cleanup job is required.
- Admin log views retain `po` for internal diagnosis because only user-facing formatters are changed.
- The dashboard remains compatible with older backend responses that still contain `po`.
- The relay continues to inject system prompts upstream; only the user-facing log projection changes.

## Risks and Rollback

- Risk: a future internal audit key may be renamed and require a follow-up redaction entry. Keep `po` as a named key rather than broad content heuristics.
- Rollback is a two-file code revert plus removal of the frontend guard; no data or schema rollback is needed.
- Local customization bookkeeping must add `013-hide-credit-system-prompts` documentation and a replayable patch, then register it in both customization indexes.
