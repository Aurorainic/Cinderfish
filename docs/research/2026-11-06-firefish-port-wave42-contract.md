# 第四十二波施工契约：按活动作者签名（2026-11-06）

`docs/research/api-contract/meta-federation.md` 的「已知限制」第 8 条记着一件事：出站请求固定用
第一个管理员（没有管理员时用第一个本地用户）的密钥签名。活动里声明的 `actor` 却是真正的作者。

对端会校验签名密钥属于活动声明的 actor（Mastodon 在 `ActivityPub::ProcessCollectionService`
里直接拒绝，Misskey 也会把 keyId 的 actor 与活动的 actor 比对）。所以多用户实例上，只有
管理员的帖子能被接受，其余会被 401 或静默丢弃。

## 1. 改动

| 位置 | 变化 |
|------|------|
| `federation.Client.KeySource` | 签名改为 `func(ctx, userID string) (keyID, priv, err)`；`userID` 为空表示「无用户上下文」 |
| `federation.Client.PostActivity` | 新增 `asUserID` 参数，透传给 `sign` |
| `federation.Deliverer.deliverOne` | 用队列行里已有的 `dl.UserID` 作为签名者 |
| `federation.Inbox.accept` | 用**接受者**的 id 签名，不再是管理员 |
| `httpapi.AttachFederation` | key source 按 `userID` 取该用户的密钥；`userID` 为空时回落到 `signingUserID`（实例密钥） |

投递队列本来就有 `user_id` 列，所以不需要迁移：它记录的是活动作者，正好是签名者。

## 2. 为什么保留实例密钥

普通对象抓取（`FetchObject`）没有用户上下文，也不代表某个账号发言，用实例密钥即可；这也是
Misskey 的 instance actor 的做法。回落只发生在这条路径上。

## 3. 有意的限制

1. 实例密钥仍是「第一个管理员」，没有独立的 instance actor 文档（`/actor`）。
2. 没有改 `KeyID` 的格式，仍是 `<actor>#main-key`。
3. 未新增测试；`EnsureLocalKey` 会为第一次发言的用户即时生成密钥对。

## 4. 验收

```
gofmt -l internal/ cmd/   # 空
go vet ./...              # exit 0（含测试文件编译）
```

未执行 `go test`、数据库测试与浏览器走查。
