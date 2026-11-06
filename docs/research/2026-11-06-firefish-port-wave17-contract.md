# 第十七波施工契约：用户发现（2026-11-06）

发现页此前只有热门标签与时间线，缺少「找到人」的入口；后端也没有公开的 `users` 目录端点。
本波补 `POST /api/users` 与发现页的用户页签。

## 1. 端点

`POST /api/users`（公开，匿名可读；带凭据时补关注标记）：

| 参数 | 取值 | 说明 |
|------|------|------|
| `limit` / `offset` | | 默认 10 / 0，上限 100 |
| `sort` | `+follower` `-follower` `+createdAt` `-createdAt` `+updatedAt` `-updatedAt` | `+` 为倒序；`±updatedAt` 按 `created_at` |
| `state` | `all` `alive` `admin` `moderator` `adminOrModerator` | 无 moderator 标记，moderator 与 admin 等价 |
| `origin` | `combined` `local` `remote` | 缺省 local（与上游一致） |
| `hostname` | string | 精确匹配 |

冻结账号永不返回。返回 `UserDetailed[]`，带 `notesCount`/`followersCount`/`followingCount`，
已登录查看者还能拿到 `isFollowing`/`isFollowed`。非法 `state`/`origin` 返回 400。

实现：`internal/account/users_list.go`（新）、`internal/httpapi/users_list.go`（新）。

## 2. 前端

`Explore.vue` 新增「用户」页签并作为默认页签。因为 `users` 只支持 offset，这里没有复用
`MkUserList`（它走 id 游标），而是自己维护 offset 与「加载更多」：每次请求 `limit+1` 条判断是否还有
下一页，参数固定 `sort=+follower, state=all, origin=local`。上游的 featured 页签仍没有端点，保持不做。

## 3. 有意的限制

1. 没有推荐/随机排序与“仅活跃”判断：`alive` 与 `all` 相同（都排除冻结）。
2. 没有 `users/search` 之外的过滤（本波用现成的 `users/search` 不动）。
3. 远程用户在未联邦的实例上为空集，页签不做特殊提示。
4. 分页是 offset 模式：`users` 端点没有游标，用户页签自己维护 offset 并用「加载更多」追加，
   不复用走 id 游标的 `MkUserList`。

## 4. 验收

`go vet`、`go test ./...`（新增目录端点用例：冻结过滤、排序、来源、关注标记、非法参数）、
`vue-tsc`、`check:port`、`pnpm build` 全绿；真实二进制走查发现页用户页签。证据见 HANDOVER §12AG.3。
