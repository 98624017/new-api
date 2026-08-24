# Implementation Plan

1. [x] Update `model.formatUserLogs` to remove `po` alongside existing admin-only fields.
2. [x] Extend `model/log_format_test.go` with a regression case proving `po` and its nested prompt content are absent while normal billing fields remain.
3. [x] Update `task-dashboard/src/components/CreditRecordsPage.tsx` so `parseCreditLogOther` drops `po` before any UI consumer sees it.
4. [x] Extend `task-dashboard/src/components/CreditRecordsPage.test.tsx` to assert the parser removes `po` and preserves ordinary billing metadata.
5. [x] Add `docs/customizations/013-hide-credit-system-prompts.md`, `patches/013-hide-credit-system-prompts.patch`, and register the new customization in `docs/customizations/README.md` and `patches/README.md`.
6. [x] Run focused backend and dashboard tests, then attempt `graphify update .` and `make verify-patches` from `new-api`; run the dashboard test/build commands from `/root/projects/task-dashboard`.

## Validation Commands

- `go test ./model -run '^TestFormatUserLogs' -count=1`
- `bun run test`
- `bun run build`
- `graphify update .`
- `make verify-patches`
