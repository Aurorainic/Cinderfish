# 第三十六波施工契约：公告已读状态（2026-11-06）

公告页此前明确写「已读状态还没有服务端记录」。本波加迁移、端点与页面标记。

## 1. 数据模型（迁移 v17）

```sql
CREATE TABLE announcement_reads (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    announcement_id TEXT NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, announcement_id)
);
```

## 2. 端点与打包

- `POST /api/announcements`：带有效 token 时按 `announcement_reads` 填充 `isRead`；匿名恒 false；
- `POST /api/i/read-announcement`（需登录）`{announcementId}` → 幂等 upsert → `204`；
- `MeDetailed.hasUnreadAnnouncement` = 有效未过期公告中用户未读的数量 > 0，在 `i` 与 `i/update`
  的打包里填充。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `api/endpoints.ts` | `i/read-announcement` |
| `pages/Announcements.vue` | 打开/加载更多后逐条标记已读；未读公告左侧强调线；全部读完后清 `hasUnreadAnnouncement` |

## 4. 有意的限制

1. 页面一打开就把当前页标记已读，而不是滚动到视口才标记。
2. 修改公告内容不会重置已读状态（id 不变）。
3. 没有「全部标为已读」按钮。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
