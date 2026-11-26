# R2 最新备份获取

## 配置与访问

优先从当前项目明确指定的生产配置、备份服务配置、受保护环境变量或已认证的 Cloudflare profile 查证账户、bucket 和 prefix。只查询所需配置，输出配置项是否可用，不回显凭证。

备份实现的来源是同级 `newapibackup` 仓库：读取其 `README.md`、`internal/backup/manifest.go` 和上传逻辑。当前工作区可通过 `/root/projects/newapibackup/` 查阅；路径不存在时定位实际备份服务仓库。

该实现使用这些配置名：

| 配置 | 用途 |
| --- | --- |
| `R2_BUCKET_NAME` | 备份桶 |
| `R2_PREFIX` | 实例前缀，空值代表 bucket 根目录 |
| `R2_ACCOUNT_ID` | S3 endpoint 中的账户 ID |
| `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | S3 访问凭证 |
| `BACKUP_NAME` | manifest 中的实例名称 |

Wrangler 使用已认证 profile 或现有 `CLOUDFLARE_ACCOUNT_ID` / `CLOUDFLARE_API_TOKEN`；R2 S3 access key 不能当作 Wrangler API token 使用。已有工具可读对象时直接复用，权限不足时只询问对应实例的配置来源或已授权的认证方式，不要求用户在聊天粘贴密钥。

本项目生产入口已于 2026-10-08 通过 R2 桶列表和 manifest 实测确认：

| 配置 | 已核验值 |
| --- | --- |
| Cloudflare account ID | `b90e43483db7d824b74157de8bf5c718` |
| bucket | `xinbaoapi` |
| prefix | 空字符串（bucket 根目录） |
| latest object | `full/latest.json` |
| manifest name | `newapi` |

未提供覆盖配置时使用这个入口，并核对当前认证账户与 manifest 身份。已有环境变量或明确的新生产配置优先，配置改变时重新验证。`R2_PREFIX` 为空是有效配置，不作为缺失项。

README 的 `newapi/full/latest.json` 是示例；实测在本桶不存在，不能自动将项目名用作 prefix。当前机器的 Wrangler OAuth 已能读取该桶，即使 R2/S3 环境变量未设置也可直接沿用认证。

## 最新成功快照合同

按实际 prefix 拼接以下对象路径，去掉 prefix 首尾斜杠：

```text
<prefix>/full/latest.json
<prefix>/full/YYYY/MM/<backup-name>-backup-YYYYMMDD-HHMMSS.sql.gz
<backup-object>.json
```

prefix 为空时从 `full/` 开始。当前备份实现先上传 SQL gzip，再上传同名 manifest，最后更新 `latest.json`；因此优先读取这个发布指针，避免依赖对象列表顺序或文件名猜测。

下面是 manifest v1 的示例。

```json
{
  "schema_version": 1,
  "name": "newapi",
  "created_at": "2026-05-08T00:30:00Z",
  "object": "newapi/full/2026/05/newapi-backup-20260508-003000.sql.gz",
  "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "size": 16984003,
  "format": "plain sql gzip",
  "excluded_fields": ["public.logs.other.po"]
}
```

读取后验证 schema/version、格式、实例名称、有效带时区的时间、正整数大小、64 位十六进制 SHA-256，以及 object 位于已确认的实例 `full/` 前缀内。同名 manifest 与固定的 latest manifest 应具有一致的对象、大小和 hash。格式/版本改变时先查最新实现，不能把其他格式当作 SQL gzip。

读取该快照可选的 `excluded_fields` 字符串数组，在报告里列出声明排除的字段。当前导出器会从 `public.logs.other` 删除 `po`，因此这类快照无法判定某次请求是否有参数覆盖审计。旧 manifest 缺少此声明时，将排除范围记作“未声明”，结合当时的导出实现和可见记录判断字段覆盖。

`latest.json` 缺失或损坏时，用现有 S3 列举工具读取同一实例的完整备份和同名 manifest，分页枚举后按有效 `created_at` 选择最新可完整校验对象；记录降级选择。只有 Wrangler 而无法列举时查可用 API/连接器或报告缺少的工具能力，不虚构 `r2 object list` 命令。

快照超过配置的备份间隔时报告延迟，仍可分析该快照覆盖的数据；快照之后的事件明确不在证据范围内。不要自动触发生产备份任务。

## 下载和校验

优先使用已认证的 Wrangler；运行前查本机 `wrangler r2 object get --help`，涉及版本差异时参考官方文档：
https://developers.cloudflare.com/workers/wrangler/commands/#r2-object-get

以下 Bash 示例使用已核验的本项目默认入口，允许明确配置覆盖。认证账户先核对上述 account ID。项目要求 RTK 时保留前缀，其他环境使用实际可用工具。

```bash
NEWAPI_ANALYSIS_DIR="$(rtk proxy mktemp -d /tmp/newapi-analysis.XXXXXX)"
rtk chmod 700 "$NEWAPI_ANALYSIS_DIR"
NEWAPI_R2_BUCKET="${R2_BUCKET_NAME:-xinbaoapi}"
NEWAPI_R2_PREFIX="${R2_PREFIX:-}"
NEWAPI_R2_PREFIX="${NEWAPI_R2_PREFIX#/}"
NEWAPI_R2_PREFIX="${NEWAPI_R2_PREFIX%/}"
NEWAPI_FULL_PREFIX="${NEWAPI_R2_PREFIX:+$NEWAPI_R2_PREFIX/}full/"
rtk wrangler r2 object get \
  "${NEWAPI_R2_BUCKET}/${NEWAPI_FULL_PREFIX}latest.json" \
  --remote --cwd "$NEWAPI_ANALYSIS_DIR" \
  --file "$NEWAPI_ANALYSIS_DIR/latest.json"
```

在专用目录运行可避免加载无关项目的 Wrangler 配置。认证核验和桶探测也使用 `--cwd "$NEWAPI_ANALYSIS_DIR"`，避免在仓库生成 `.wrangler/` 缓存。需要指定 profile 时沿用已确认的 `--profile`，账户信息从既有安全环境注入。

完成前述 manifest 验证后再读取 object 并下载，固定使用本地 `backup.sql.gz` 文件名：

```bash
NEWAPI_BACKUP_OBJECT="$(rtk proxy jq -r '.object' "$NEWAPI_ANALYSIS_DIR/latest.json")"
rtk wrangler r2 object get \
  "${NEWAPI_R2_BUCKET}/${NEWAPI_BACKUP_OBJECT}.json" \
  --remote --cwd "$NEWAPI_ANALYSIS_DIR" \
  --file "$NEWAPI_ANALYSIS_DIR/object-manifest.json"
rtk wrangler r2 object get \
  "${NEWAPI_R2_BUCKET}/${NEWAPI_BACKUP_OBJECT}" \
  --remote --cwd "$NEWAPI_ANALYSIS_DIR" \
  --file "$NEWAPI_ANALYSIS_DIR/backup.sql.gz"
rtk chmod 600 "$NEWAPI_ANALYSIS_DIR/backup.sql.gz"
rtk proxy stat -c '%s' "$NEWAPI_ANALYSIS_DIR/backup.sql.gz"
rtk sha256sum "$NEWAPI_ANALYSIS_DIR/backup.sql.gz"
rtk gzip -t "$NEWAPI_ANALYSIS_DIR/backup.sql.gz"
```

把 stat 和 SHA-256 输出分别与 manifest 的 `size` / `sha256` 精确比较，再确认 gzip 校验成功。任一不一致都停止分析，重新获取或报告损坏；不能只运行命令而不比较结果。

已有 AWS CLI 和 S3 凭证时，也可使用 `aws s3 cp --endpoint-url "https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"` 读取同一对象。沿用安全环境/profile，将 R2 凭证映射到 AWS 所需环境变量而不打印值；不修改持久认证配置。

下载前检查磁盘空间，下载期间限制等待以便报告进度。已有本地缓存只有在最新指针仍指向同一 object 且大小、hash、gzip 校验一致时可复用。原始备份只保存在权限受限的临时目录。
