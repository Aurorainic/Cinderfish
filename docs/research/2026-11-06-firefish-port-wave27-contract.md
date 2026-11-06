# 第二十七波施工契约：实例信息页（2026-11-06）

`/instance-info/:host` 此前是 `TODO: port`。本波实现 `federation/show-instance` 与页面。

## 1. 端点

`POST /api/federation/show-instance`（需登录）请求 `{host}`，返回 `FederationInstance | null`：

- 本地主机：名称/简介来自运行时设置，`softwareName = "cinderfish"`，`softwareVersion = null`，
  `openRegistrations` 由注册模式推导，用户/帖子数为实时计数；
- 其它主机：只统计库内已收录的 `users.host` / `notes.user_host`；未知主机返回 `null`；
- 远端软件字段全部 `null`，状态标志全部 `false`；
- **不发网络请求**，不做 nodeinfo 抓取（避免 SSRF）。

`host` 会去掉 `http(s)://`、路径与末尾斜杠并转小写。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `FederationInstance`、`federation/show-instance` |
| `pages/InstanceInfo.vue` | 主机输入、查询、软件/注册/计数展示；未知主机与 401 分别提示 |

## 3. 有意的限制

1. 远端实例信息不联网校验，只反映本库已收录的数据。
2. `softwareName`/`softwareVersion`/`latestStatus` 对远端恒为 `null`。
3. 不做实例屏蔽/暂停状态的展示（那两个标志恒 `false`）。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
