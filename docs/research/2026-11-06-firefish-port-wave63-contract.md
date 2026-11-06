# 第六十三波施工契约（真实二进制 + 无头浏览器走查与两个缺陷修复）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12CB
验证环境：

- 二进制：`/tmp/cf-w63`（本仓库 `cmd/cinderfish` 当前提交编译）
- 配置：`/tmp/w63.toml`，`127.0.0.1:3999`，domain `localhost:3999`，库 `cinderfish_test`（schema v21），
  federation public，media 本地 `/tmp/w63-media`，前端 `web/dist` 为当前构建产物
- 浏览器：Helium 0.20.1.1（Chromium 156），CDP 驱动，截图在 `/tmp/w63-shots`（不入库）

## 1. 走查清单与结果

| 页面/流程 | 结果 |
|-----------|------|
| 欢迎页 → 登录（真实表单）→ 首页时间线 | **发现缺陷**：登录后时间线 401「发生了一个错误」，刷新才恢复（见 §2.1） |
| `/settings/migration` 表单渲染 | 正常；「迁移到」「从旧账号迁移」两段与说明齐全 |
| 迁移到本地账号 `bob` + 确认对话框「好」 | 服务端拒绝并回显 `User is not remote. You can only migrate to other instances.` |
| 旧账号别名保存 → 重新载入 | **发现缺陷**：保存落库但解析不回昵称（见 §2.2）；修复后输入框回填 `@w63bob` |
| `/oauth/authorize`（PKCE） | 同意页显示应用主机 `w63app.example`、权限「读取账号与内容 / 以你的名义发帖」；接受后浏览器跳向回调域，库里出现授权码（scopes `["read","write"]`） |
| `/auth/<token>` | 显示应用名 `W63 Walkthrough App` 与两项权限；接受后 `auth_sessions.user_id` 置位，`access_tokens` 生成 app 绑定令牌（2 个 scope），浏览器跳向 `callback?token=…` |
| `/authorize_interaction?uri=…` | 远端不可达时显示「无法解析这个链接」与「在原实例查看」 |
| `/tags/w63go` | 头部「被 2 人提到 · 全部时间」，三条带标签笔记按时间列出 |
| `/my/notifications` | 显示「关注了您」与两条「发布了新笔记」（`type: note`）通知 |
| `/settings/custom-katex-macro` | 输入 `\newcommand{\R}{\mathbb{R}}` → 保存 → localStorage `customKaTeXMacro` 与 `customKaTeXMacroParsed`（`{"R":{"args":0,...}}`）正确，页面显示「已保存」 |
| `/settings/integration`、`/settings/plugin`、`/settings/plugin/install`、`/signup-complete/abc` | 四个说明页文案与「这里已经有 / 上游有、这里没有」结构正常 |
| 移动端（390×844）：migration、tag、notifications、plugin、oauth、katex、integration | `scrollWidth - clientWidth = 0`，无横向溢出；无 JS 异常 |

所有页面均未出现 `Runtime.exceptionThrown` 或 console error。

## 2. 发现并修复的缺陷

### 2.1 登录后必须刷新才能用

`web/src/api/token.ts` 在首次调用时把 token 读入内存并缓存；`account/me.ts` 的 `mirrorToken`
只写 localStorage，从不更新这份缓存。于是「先以游客打开、再登录」的页面在整个生命周期里继续发匿名
请求：`notes/timeline` 与 `i/registry/get-all` 都拿到 401，时间线停在错误态。

修法：`mirrorToken` 改为调用 `setToken()`，内存副本与 localStorage 同步。

### 2.2 本地账号的 Actor URI 解析不了

`i/known-as` / `i/move` 存的是 Actor URI，迁移页随后用 `users/show` 的 `userId` 把它们换回昵称；
`users/show` 的 URI 回退只查 `users.uri`，而**本地账号这一列永远为空**（其 URI 由 baseURL 推导）。
结果是别名保存成功、页面却回填空白，`movedToUri` 对本地目标同样解析不出。

修法：新增 `Server.userByAPID`：先查 `users.uri`，再匹配 `<baseURL>/users/<id>` 取本地 id；
`users/show` 的单值与 `userIds` 批量、以及 `packMovedTo` 都改用它。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `web/src/account/me.ts` | `mirrorToken` 走 `setToken` |
| `internal/httpapi/users_pack.go` | 新增 `userByAPID`，`packMovedTo` 改用 |
| `internal/httpapi/account.go` | `users/show` 单值与批量改用 `userByAPID` |
| `internal/httpapi/migration_test.go` | 增加「按 Actor URI 回查本地账号」断言 |

## 4. 仍未验证

- `MkMoved` 横幅：需要真实远端账号把自己标到本实例的本地账号上，单实例无法构造。
- 双因素登录对话框、关注者移除菜单、远端举报的管理台展示没有走查。
- 真实联邦（远端实例）仍未联调，全部联邦验证仍是双实例测试。
