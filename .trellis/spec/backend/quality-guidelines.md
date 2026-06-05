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

- Good: `ChannelTypeDoubaoVideo` reads OpenAI Videos top-level `input_video` / `files` fields and forwards the original top-level request body upstream.
- Base: `ChannelTypeVolcEngine` keeps legacy `metadata.content` conversion and legacy video-input ratio behavior.
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

## Testing Requirements

<!-- What level of testing is expected -->

(To be filled by the team)

---

## Code Review Checklist

<!-- What reviewers should check -->

(To be filled by the team)
