# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

<!--
Document your project's quality standards here.

Questions to answer:
- What patterns are forbidden?
- What linting rules do you enforce?
- What are your testing requirements?
- What code review standards apply?
-->

(To be filled by the team)

---

## Forbidden Patterns

<!-- Patterns that should never be used and why -->

(To be filled by the team)

---

## Required Patterns

### Scenario: Task Adaptor Channel-Specific Request And Billing Contracts

#### 1. Scope / Trigger

- Trigger: a task adaptor supports more than one upstream channel or request shape, especially when billing depends on fields from the client request body.
- Applies to task relay adapters under `relay/channel/task/**`.

#### 2. Signatures

- Channel identity must come from `relay/common.RelayInfo.ChannelMeta.ChannelType` during adaptor initialization.
- Billing entry point: `EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64`.
- Request conversion entry point: `BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error)`.

#### 3. Contracts

- Every channel-specific request-body format needs an explicit channel guard before reading format-only fields.
- If a field can trigger extra billing, the same channel path must also forward or otherwise consume that field.
- Environment-gated billing rules must use explicit model allowlists and must be reloaded after environment initialization.

#### 4. Validation & Error Matrix

- Allowlisted model + forwarded reference-video field -> return the configured `OtherRatios` entry.
- Allowlisted model + field that belongs to another channel's request format -> do not charge from that field.
- Non-allowlisted model + reference-video field -> preserve baseline billing behavior.
- Legacy request format with existing video metadata -> preserve legacy video-input billing behavior.

#### 5. Good/Base/Bad Cases

- Good: the adaptor path that actually forwards OpenAI Videos top-level `input_video` / `files` fields also owns the billing detector for those fields.
- Base: a legacy provider-specific adaptor keeps its legacy request conversion and legacy video-input ratio behavior.
- Bad: a shared `EstimateBilling` checks OpenAI Videos top-level fields without confirming the current channel actually forwards those fields.

#### 6. Tests Required

- Positive unit test for allowlisted model + channel-supported reference-video field.
- Negative unit test for non-video reference fields.
- Negative unit test for non-allowlisted models.
- Regression test proving another channel handled by the same adaptor is not charged for fields it does not forward.
- Request-body conversion test proving the charged fields are present in the upstream request for that channel.

#### 7. Wrong vs Correct

##### Wrong

```go
if IsAllowlisted(modelName) && hasTopLevelReferenceVideo(c) {
	return map[string]float64{"video_input": 2}
}
```

##### Correct

```go
hasReferenceVideo := adaptor.usesOpenAIVideosRequest() && hasTopLevelReferenceVideo(c)
if IsAllowlisted(modelName) && hasReferenceVideo {
	return map[string]float64{"video_input": 2}
}
```

---

### Scenario: Legacy Task Result URL Classification In Frontend Views

#### 1. Scope / Trigger

- Trigger: a frontend view consumes `dto.TaskDto.status`, `fail_reason`, or `result_url` to render task results or failure states.
- Applies to task logs, task detail dialogs, result links, and any destructive/error styling derived from these fields.

#### 2. Signatures

- Backend compatibility source: `func (t *model.Task) GetResultURL() string`.
- DTO projection: `func relay.TaskModel2Dto(task *model.Task) *dto.TaskDto`.
- Default frontend classifier: `getTaskFailureReason(status: string, failReason?: string, resultUrl?: string): string`.

#### 3. Contracts

- New tasks store their result URL in `Task.PrivateData.ResultURL`; legacy tasks may store a successful video URL in `Task.FailReason`.
- `TaskModel2Dto` uses `GetResultURL()`, so a legacy successful task can legitimately return the same non-empty value in both `fail_reason` and `result_url`.
- When `status === SUCCESS` and the trimmed values are equal, frontend views must treat the value as a result URL: hide the failure section and do not apply failure styling.
- Failed tasks must keep displaying `fail_reason`, even if it equals `result_url`; successful tasks with distinct non-empty values must also preserve the distinct failure/warning text.
- Result links and video proxy behavior remain available when the duplicate legacy failure value is hidden.

#### 4. Validation & Error Matrix

- `SUCCESS` + equal non-empty trimmed values -> no visible failure reason and no destructive detail-entry style.
- `SUCCESS` + distinct non-empty values -> show `fail_reason`; keep `result_url` as a result.
- `FAILURE` + equal values -> show `fail_reason` because status takes precedence over legacy-result compatibility.
- Any status + empty/whitespace-only `fail_reason` -> no visible failure reason.

#### 5. Good/Base/Bad Cases

- Good: derive failure text once with `getTaskFailureReason` and reuse it for both detail content and failure styling.
- Base: a modern successful task has `result_url` populated and an empty `fail_reason`; render only the result.
- Bad: use `Boolean(task.fail_reason)` as the failure predicate, because legacy successful video tasks then appear failed.

#### 6. Tests Required

- Unit test that `SUCCESS` with equal URLs, including whitespace differences, returns no failure text.
- Regression test that `FAILURE` with equal values still returns the complete failure text.
- Unit test that `SUCCESS` with distinct failure and result values preserves the failure text.
- UI consumers must use the same classifier for the failure section and destructive styling.

#### 7. Wrong vs Correct

##### Wrong

```ts
const isFailed = Boolean(task.fail_reason?.trim())
```

##### Correct

```ts
const failureReason = getTaskFailureReason(
  task.status,
  task.fail_reason,
  task.result_url
)
const isFailed = Boolean(failureReason)
```

---

## Testing Requirements

<!-- What level of testing is expected -->

(To be filled by the team)

### Scenario: Replayable Customization Chain And Embedded Dual Frontends

#### 1. Scope / Trigger

- Trigger: a local customization changes code, tests, frontend dependencies, embedded frontend assets, or cache initialization.
- Applies to `patches/*.patch`, `scripts/verify_patches.sh`, `web/default`, `web/classic`, `web/shared`, and Go builds that embed frontend `dist` directories.

#### 2. Signatures

- Full gate: `make verify-patches`.
- Locked upstream input: `PATCH_BASE_REF`, defaulted in `scripts/verify_patches.sh`.
- Frontend install: `bun install --cwd web --frozen-lockfile`.
- Required frontend outputs: `web/default/dist` and `web/classic/dist` before `go build ./...`.
- Shared lock cache key: `FRONTEND_LOCK_STORAGE_KEY` from `web/shared/frontend-lock`.

#### 3. Contracts

- Applying `patches/001-*.patch` through `patches/009-*.patch` to `PATCH_BASE_REF` must reproduce every patch-owned path byte for byte from the integration tree.
- Patch application alone is not success. The replay tree must install locked Bun dependencies, test shared frontend state, build both frontends, compile Go, and run customization regressions.
- `web/classic` must explicitly pin `date-fns@2.30.0` and `date-fns-tz@1.3.8` while `web/default` uses its newer dependency line. Do not rely on Bun workspace hoisting to select a compatible transitive version.
- Both frontend builds may run in parallel, but `go build ./...` must wait for both because Go `embed` consumes both output trees.
- Parallel Go regression groups must not execute the same package in two processes.
- Default frontend cache cleanup must preserve `FRONTEND_LOCK_STORAGE_KEY`; default and classic share one unlock record across frontend switches.

#### 4. Validation & Error Matrix

- Source or test change without its customization patch -> fail before replay.
- Patch applies but a new source file is absent from the patch -> final-tree comparison or build must fail.
- Classic resolves `date-fns-tz@1.3.8` against `date-fns@4` -> classic build must fail; restore the explicit classic pin and lockfile.
- Either frontend `dist` is missing -> Go build is invalid and must not start.
- A frontend build or any parallel regression exits non-zero -> wait for all started jobs, then fail the gate.
- Default cache initialization deletes the shared lock key -> the cache regression must fail because switching frontends would require an unexpected second unlock.

#### 5. Good/Base/Bad Cases

- Good: update code, its numbered patch, and customization documentation; replay from the locked upstream base; compare final paths; build and test the replay tree.
- Base: documentation-only task delivery outside a customization may remain outside the numbered patch chain, but must not change runtime behavior.
- Bad: accept `git apply --check` as sufficient evidence, build Go before frontend assets exist, or let two parallel commands test the same Go package.

#### 6. Tests Required

- Run `make verify-patches` under an outer 120-second timeout.
- Assert all numbered patches apply in order and replay-owned paths equal the integration tree.
- Run `web/shared/frontend-lock.test.ts` and `web/default/src/lib/frontend-cache.test.ts`; assert the shared unlock survives default cache initialization.
- Build `web/default` and `web/classic`, then run `go build ./...`.
- Run the numbered customization regression groups and assert every background job exit status is collected.

#### 7. Wrong vs Correct

##### Wrong

```bash
git apply --check patches/*.patch
go build ./...
```

```ts
const PRESERVED_LOCAL_STORAGE_KEYS = new Set(['user', 'uid'])
```

##### Correct

```bash
timeout 120s make verify-patches
```

```ts
import { FRONTEND_LOCK_STORAGE_KEY } from '../../../shared/frontend-lock'

const PRESERVED_LOCAL_STORAGE_KEYS = new Set([
  'user',
  'uid',
  FRONTEND_LOCK_STORAGE_KEY,
])
```

---

## Code Review Checklist

<!-- What reviewers should check -->

(To be filled by the team)
