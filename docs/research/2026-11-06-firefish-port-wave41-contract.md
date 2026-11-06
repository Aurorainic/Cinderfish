# 第四十一波施工契约：编辑的联邦 Update（2026-11-06）

第四十波修好出站投递后，`notes/edit` 成了同一条链路上剩下的窟窿：编辑只改本地库、只推
Streaming，远端看到的还是旧正文；反过来，对端编辑过的帖子在我们这边也永远停在旧内容——
`inbox.update` 只处理 actor，收到笔记对象直接回 `skip: cannot update Note`。本波两边都补。

## 1. 出站

`federation.Renderer.RenderUpdate` 把渲染好的笔记包成 `Update`：

```json
{ "@context": "...", "id": "<Create id>#updates/<毫秒>", "type": "Update",
  "actor": "<actor>", "published": "...", "object": { ...note... }, "to": [...], "cc": [...] }
```

id 带编辑时间戳，避免对端按活动 id 去重时把第二次编辑当成重复投递丢掉。`to`/`cc` 从笔记
对象继承，和 `Create` 一致。

`Server.DeliverUpdate` 用与 `DeliverNote` 相同的 `deliveryInboxes` 投递，纯转贴跳过（它没有
自己的内容可改），`localOnly` 与远端作者同样跳过。调用点在 `handleNotesEdit` 成功之后。

注意：Mastodon 忽略 status 的 `Update`，会一直留旧副本——这是对端行为，不是本实现的缺陷，
所以前端仍然显示本地「已编辑」标记。

## 2. 入站

`inbox.update` 按对象类型分流：

- actor 文档 → 走既有 `upsertActor`，行为不变；
- 笔记对象（`Note`/`Question`/`Article`/`Page`）→ `updateNote`：
  1. 必须有 `id`；`attributedTo` 存在时必须等于投递方 actor；id 的主机必须等于 actor 主机
     （与 `create` 相同的两条防伪检查）；
  2. 取 `updated` 作编辑时间，缺失则用当前时间；
  3. 有 `content` 就更新正文（HTML 反转义 + 去首尾空白），有非空 `summary` 就更新 CW，
     `sensitive` 直接跟随；
  4. 按 `ap_id` 定位并更新 `notes`，行不存在或已软删除则 `skip: note unknown`。

**只动内容**：可见性与受众不变。对端不一定重发 `to`/`cc`，按缺失值重算会把 followers 笔记
误判成公开或反向隐藏它。

## 3. 有意的限制

1. 远端笔记的编辑标签、标签/提及/表情的重新解析没有做——只更新正文、CW、敏感标记与
   `updated_at`。
2. 入站编辑不推 Streaming：没有既成的「笔记被编辑」事件，客户端刷新后才看到新内容。
3. Mastodon 侧忽略 status `Update`，双向都不完全对称。
4. 没有为 Update 写新测试（本阶段不跑测试）。

## 4. 验收

```
gofmt -l internal/ cmd/   # 空
go vet ./...              # exit 0
```

前端本波未改动。未执行 `go test`、数据库测试与浏览器走查。
