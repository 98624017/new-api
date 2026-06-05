# Seedance 视频参考双倍计费

## Goal

Seedance/Doubao 视频任务在 OpenAI Videos 风格请求体中包含视频参考素材时，按可配置模型白名单触发双倍计费，覆盖未来模型名称变化。

## Requirements

- 仅对环境变量白名单中的模型生效，未配置模型不得改变现有计费。
- 目标渠道是 Seedance/Doubao 视频任务适配器；请求体格式参考 `SEEDANCE_NEWAPI_OPENAI_VIDEOS_API.md`。
- 同一适配器需同时保留旧 Doubao/VolcEngine 渠道和新 Seedance OpenAI Videos 渠道：`VolcEngine` 继续旧 `content[]` 格式，`DoubaoVideo` 使用顶层 `files` / `input_video` 等字段。
- 需要识别顶层参考视频字段，包括 `input_video`、`video_url`、`reference_video`，以及 `files` 中明显为视频 URL 的素材。
- 已有 `metadata.content` 中 `video_url` 的视频输入计费逻辑需继续保留。
- 本次只实现“含视频参考则双倍计费”；不启用或新增按参考视频秒数精细计费。
- 环境变量应支持逗号分隔模型名，便于后续调整模型白名单。
- 作为本地定制变更，需同步对应 `docs/customizations` 与 `patches` 文件，并通过补丁校验。

## Acceptance Criteria

- [x] 配置白名单模型且请求包含顶层视频参考字段时，预扣费 OtherRatio 包含 `video_input: 2`。
- [x] 同一请求仅包含图片或音频参考时，不触发双倍计费。
- [x] 模型未在环境变量白名单中时，即使请求包含视频参考也不触发双倍计费。
- [x] 原有 `metadata.content` 视频输入逻辑不回退。
- [x] 新增或更新单元测试覆盖白名单、视频字段、非视频字段和未配置白名单场景。
- [x] 运行相关 Go 测试、补丁校验，并在修改代码后运行 `graphify update .`。`graphify` AST 提取完成，但 HTML 可视化因图节点过多失败；`--no-viz` 在当前 CLI 下仍触发相同错误。

## Notes

- 现有 Sora 实现使用 `SORA_REFERENCE_VIDEO_DOUBLE_PRICE_MODELS` 做白名单，并在默认模式下返回 `video_input: 2`。
- Doubao 适配器当前只检查 `metadata.content` 中的 `video_url`，无法覆盖文档中的顶层 `files` / `input_video` / `reference_video` 等兼容字段。
