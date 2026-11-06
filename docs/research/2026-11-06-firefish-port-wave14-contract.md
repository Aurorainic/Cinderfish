# 第十四波施工契约：实例屏蔽（2026-11-06）

`AdminInstanceBlock.vue` 是占位页。本波把 Misskey 的 `meta.blockedHosts` 做成真实开关：存
`instance_meta`，`admin/meta` 回读，`admin/update-meta` 写入，并在出站与入站两个方向真正拒绝。

## 1. 策略层

`internal/federation/policy.go`：

- 新增 `blocked` / `blockedAt` 缓存（与 allowlist 同一个 10s TTL）；
- `HostBlocked(ctx, host)`：规范化后**精确匹配**（`bad.example` 不匹配 `sub.bad.example`）；
- `BlockedHosts(ctx)`：读 `instance_meta.blockedHosts`（JSONB 数组）；
- `SetBlockedHosts(ctx, hosts)`：规范化（小写/去空/去重/排序）后写库并立即失效缓存；
- `IsAllowed` 在 selfHost 之后先查屏蔽表，命中直接拒绝（出站 `client.go` 全部走这里）。

## 2. 入站

`handleInbox` 在签名验证之后、处理活动之前解析 actor 主机，命中屏蔽表返回 403。放在签名之后是
刻意的：未认证的请求不能拿来探测屏蔽名单。

## 3. 管理接口

`admin/meta` 增加 `blockedHosts`（永远是数组，空列表为 `[]`）；
`admin/update-meta` 增加 `blockedHosts: string[]`，校验：≤253 字符、不含 scheme/路径/`@`/空格、
最多 200 条；空数组清空。

## 4. 有意的限制

1. **精确匹配**：不自动包含子域；页面提示需要时逐条列出。
2. **不删除已有内容**：已入库的远端帖子/文件与已有关注关系保留，只阻止之后的收发；页面写明。
3. **不改 allowlist 语义**：入站仍不做 allowlist 检查（既有行为），只新增屏蔽检查。
4. **没有“静音实例”与转发屏蔽**：`silencedHosts` 与 `forward` 不在本波。

## 5. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/AdminInstanceBlock.vue` | 占位页改为多行文本编辑：加载、去重预览、保存、数量提示、精确匹配与“不删除已有内容”说明 |
| `web/src/pages/AdminIndex.vue` | 内容组新增「实例屏蔽」 |
| `web/src/api/types.ts` | `AdminInstanceMeta.blockedHosts` |

## 6. 验收

`go vet`、`go test ./...`（新增：策略精确匹配单测、blockedHosts 回读/校验）、`vue-tsc`、
`check:port`、`pnpm build` 全绿；真实二进制走查页面保存与持久化、移动端。证据见 HANDOVER §12AD.3。
