# 017 OpenAI 图片生成宽高比

OpenAI 类型的渠道在处理 `POST /v1/images/generations` 时会重新生成发往上游的 JSON。部分兼容模型使用 `size` 指定分辨率、使用 `aspect_ratio` 指定宽高比。例如，`seedream-5.5` 的请求可以同时携带 `"size":"2K"` 和 `"aspect_ratio":"9:16"`。

补丁让 OpenAI 渠道在关闭请求体透传时也转发请求中显式提供的 `aspect_ratio`。省略该字段时沿用上游默认行为。支持哪些比例由上游模型决定，渠道的参数覆盖仍在生成上游 JSON 后执行。

同步上游时按序重放 `patches/017-openai-image-aspect-ratio.patch`。手动恢复时，在 `relay/channel/openai/adaptor.go` 的 OpenAI 图片生成分支选取该字段，由 `dto/openai_image.go` 写入上游 JSON。将 `relay/channel/openai/image_generation_test.go` 的回归用例加入 `scripts/verify_patches.sh`。

```bash
go test ./relay/channel/openai -run '^TestImageGenerationAspectRatio$' -count=1
make verify-patches
```
