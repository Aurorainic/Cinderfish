# 第十八波施工契约：运维可视化（2026-11-06）

`AdminObjectStorage.vue` 与 `AdminQueue.vue` 是占位页。对象存储没有可写的管理端点，投递队列也不是
Bull/Redis；本波按本实例的实际结构做两个只读页。

## 1. 端点（Cinderfish 自有，非 Misskey 同名）

| 端点 | 返回 |
|------|------|
| `POST /api/admin/object-storage` | `{ backend, localPath, s3: { endpointHost, bucket, region, useSSL, publicBaseUrl, hasCredentials } \| null }` |
| `POST /api/admin/deliver-queue` | `{ pending, dueNow, failed, oldestNextAt, recent: [{ id, host, attempts, nextAttemptAt, lastError }] }` |

实现：`internal/httpapi/admin_infra.go`（新）。

- object-storage 只读，且**绝不返回 access key / secret key**；endpoint 只保留主机部分。
- deliver-queue 的 `pending` 是 `attempts < 12`（worker 会取用的行）、`dueNow` 是其中
  `next_attempt_at <= now()`、`failed` 是 `attempts >= 12`（已不再重试）；`recent` 取 id 最大的 20 行。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `AdminObjectStorage.vue` | 只读卡片：后端、本地路径或 S3 的主机/桶/区域/HTTPS/公开基础 URL/凭据是否配置；说明改 `config.toml` 需重启 |
| `AdminQueue.vue` | 三个计数（待投递/已到期/超重试上限）、最早待投递时间、最近 20 条（主机、尝试次数、下次尝试、最后错误）、刷新 |
| `AdminIndex.vue` | 管理组新增「投递队列」，内容组新增「对象存储」 |

两个页面都没有编辑、重试、清空按钮——对应的后端能力不存在，画出来就是假控件。

## 3. 验收

`go vet`、`go test ./...`（新增两个用例：只读配置不含密钥、空队列形状与权限）、`vue-tsc`、
`check:port`、`pnpm build` 全绿；真实二进制上 seed 一条投递行后走查两页。证据见 HANDOVER §12AH.3。
