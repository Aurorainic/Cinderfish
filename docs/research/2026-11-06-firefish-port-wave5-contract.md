# 第五波施工契约：静音/屏蔽、令牌管理、无障碍、多账号与补齐 widgets（2026-11-06）

本文是第五波前端移植的施工依据。前四波的契约见
`2026-11-05-firefish-port-wave2-contract.md`、`wave3`、`2026-11-06-firefish-port-wave4-contract.md`。
本波只有一次逻辑提交，按「后端 → 设置页 → widgets → 接线」的顺序落地。

## 1. 本波目标

补上第四波遗留的一批「有真实后端依赖、但还没做」的用户面功能，并明确拒绝没有渲染方的假控件。

| 目标 | 结果 |
|------|------|
| 静音 / 屏蔽 | 6 个端点 + 设置页；时间线真正过滤；`users/show` 带 `isMuted`/`isBlocked` |
| 实例静音 | `i/update` 接受 `mutedInstances`，存 `user_profiles.settings`，时间线按主机过滤 |
| 应用授权 | `i/apps` / `i/revoke-token` + 设置页，用户可以自己撤销第三方令牌 |
| 多账号 | 设置页列出本浏览器保存的账号，可切换、移除、新增 |
| 无障碍 | 只接 6 个有消费方的 store 键 |
| widgets | 新增 `button`、`calendar`、`unix-clock`、`slideshow` |

## 2. 后端

### 2.1 mute / block

`blocks`、`mutes` 两表在 v1 schema 已存在（第 1 版迁移），本波不需要新迁移。

| 层 | 文件 | 内容 |
|----|------|------|
| 存储 | `internal/account/relations.go`（新） | `Mute`/`Unmute`/`IsMuted`、`Block`/`Unblock`/`IsBlocked`、`Mutes`/`Blocks` 列表（id 倒序、`sinceId`/`untilId`） |
| 端点 | `internal/httpapi/relations.go`（新） | `mute/create`、`mute/delete`、`mute/list`、`blocking/create`、`blocking/delete`、`blocking/list` |
| 注册 | `internal/httpapi/server.go` | 6 条 POST 路由 |
| 资料 | `internal/httpapi/account.go` | 登录查看 `users/show` 时补 `isMuted` / `isBlocked` |
| 时间线 | `internal/note/timeline.go` | 所有时间线排除被静音作者、双向屏蔽关系、被静音实例 |

行为对齐 Misskey：

- 屏蔽会在同一事务里删除双向 `followings` 与 `follow_requests`；
- `blocking/create` 返回 `UserDetailedNotMe`，其余写端点返回 204；
- 解除不存在的关系是 no-op，方便客户端重试；
- 错误 id 从 `tmp/refcheck/firefish-full/packages/backend/src/server/api/endpoints/{mute,blocking}/*.ts` 复制。

### 2.2 实例静音

`user_profiles.settings` 是 v1 就有的 JSONB，本波用它存 `mutedInstances`：

- `i/update` 增加 `mutedInstances: string[] | null`；解析时转小写、去空行、去重，拒绝带 scheme/path/空格的项，最多 200 条；
- `account.Profile` 增加 `MutedInstances`，`i` 与 `i/update` 的响应回读；
- 时间线条件：`NOT EXISTS (… up.settings ? 'mutedInstances' AND (up.settings->'mutedInstances') ? n.user_host)`，
  没有该键时不产生过滤（不会因 NULL 把整页清空）。

### 2.3 应用授权

| 层 | 文件 | 内容 |
|----|------|------|
| 存储 | `internal/account/tokens.go` | `Apps()`（按 `sort` 白名单排序）、`RevokeOwned()`（带 `user_id` 约束） |
| 端点 | `internal/httpapi/apps.go`（新） | `i/apps`、`i/revoke-token` |
| 注册 | `internal/httpapi/server.go` | 2 条 POST 路由 |

`i/apps` 只列 `access_tokens` 里的第三方令牌，原生 token 与会话不出现；`i/revoke-token` 删除他人令牌或
不存在的 id 都是静默 no-op（与 Misskey 一致）。

## 3. 前端

### 3.1 设置页

| 页面 | 数据源 | 说明 |
|------|--------|------|
| `SettingsMuteBlock.vue` | `mute/list`、`blocking/list`、`users/show`、`mute/*`、`blocking/*` | 两个名单切换；用 `@username` 添加；逐行解除 |
| `SettingsInstanceMute.vue` | `i/update`（`mutedInstances`） | 每行一个主机，保存后回读 |
| `SettingsApps.vue` | `i/apps`、`i/revoke-token` | 授权时间/最近使用/权限展开，撤销前确认 |
| `SettingsAccounts.vue` | 账号层 `readStoredAccounts` / `fetchMe` / `switchAccount` / `signIn` | 当前账号标记、切换、移除、就地登录新增 |
| `SettingsAccessibility.vue` | `defaultStore` | 只画有消费方的 6 个键 |

无障碍页刻意省略的键（无消费方）：`useBlurEffect`、`useBlurEffectForModal`、`loadRawImages`、
`imageNewTab`、`disableShowingAnimatedImages`、`iconSet`。页面底部有说明，避免「开关存在但没效果」。

### 3.2 widgets

| 文件 | 与上游的差异 |
|------|--------------|
| `web/src/widgets/button.vue` | 上游点击执行 AiScript；本移植改为打开 `url`，只允许 http(s)，配置项去掉 `script` |
| `web/src/widgets/calendar.vue` | 去掉生日/新年问候（没有 `birthday` 字段），保留日期与今天/本月/今年进度条 |
| `web/src/widgets/unix-clock.vue` | 直译，保留毫秒闪烁与透明背景 |
| `web/src/widgets/slideshow.vue` | 上游用文件夹选择器；本移植取网盘最新 24 张图片轮播，`folderId` 不提供 |

`widgets/index.ts` 的 loader 表从 12 项增到 16 项；`_widgets.*` 的 i18n 键上游文件里已经存在。

### 3.3 明确不做

| 项 | 原因 |
|----|------|
| `SettingsStatusbar.vue` | 没有状态栏渲染组件与挂载点，做了也只是存一个永不显示的列表 |
| `SettingsCustomKatexMacro.vue` | `katex` 不是依赖，`MkFormula` 已经按纯文本降级 |
| `rss` / `rss-ticker` widget | 需要服务端代理绕过 CORS，`/media` 不是通用代理 |
| `server-metric` / `online-users` widget | 需要 Streaming 的 `stats` 通道与连接计数，当前没有 |
| `SettingsEmail`、`SettingsImportExport`、`SettingsWebhook`、`SettingsMigration` 等 | 对应端点不存在（见 API 台账） |

## 4. 验收口径

本波按用户要求**只写代码，不跑测试与浏览器走查**。提交前只做编译级检查：

```
gofmt -l internal/ cmd/
go build ./...
cd web && pnpm exec vue-tsc --noEmit
cd web && pnpm run check:port
```

真实二进制走查（静音后时间线消失、屏蔽后互相不可见、撤销令牌后请求 401、切换账号后资料变化、
四个 widgets 渲染）留到用户确认后的验证轮。
