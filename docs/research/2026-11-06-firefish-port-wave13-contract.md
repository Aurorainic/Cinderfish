# 第十三波施工契约：运行时实例设置（2026-11-06）

实例名称、简介、维护者与注册模式此前只能改 `config.toml` 再重启。本波把这五项落进 `instance_meta`，
`meta`、nodeinfo 与注册校验都读运行时值，并补管理页。

## 1. 存储与读取

`instance_meta`（v1 已有）新增键：`instanceName`、`instanceDescription`、`maintainerName`、
`maintainerEmail`、`registrationMode`。

- `liveInstanceSettings(ctx)`：读这些键，缺省回退配置（名称 Cinderfish、简介空、注册模式取
  `config.registration_mode`）。
- `liveRegistrationMode(ctx)`：供 signup 使用；切换立即生效，不需要重启。
- `meta`：名称/简介/维护者/邮箱改用运行时值，`disableRegistration` 由运行时模式推导。
- `nodeinfo` 2.0/2.1：`openRegistrations` / `disableRegistration` 同样改用运行时值。

## 2. 端点

| 端点 | 权限 | 行为 |
|------|------|------|
| `POST /api/admin/meta` | 管理员 | 返回 `{name, description, maintainerName, maintainerEmail, registrationMode}` |
| `POST /api/admin/update-meta` | 管理员 | 接受上述五个字段，`null`/缺省表示不变；其它字段忽略 |

校验：名称必填且 ≤100；简介 ≤500；维护者 ≤100；邮箱可为空，否则需含 `@` 且 ≤254；
注册模式只能是 `open`/`invite`/`admin`。

## 3. 有意的限制

1. **只覆盖五个字段**：数据库、媒体、联邦策略、端口等仍在 `config.toml`；页面明确说明。
2. **忽略未列出的字段**：上游 `admin/update-meta` 的其它字段（captcha、自定义 CSS、服务条款等）没有
   实现，静默忽略并在文档与页面上写明，而不是假装保存。
3. **没有审计日志**：谁在何时改了实例设置没有记录。
4. **注册模式不校验邀请码库存**：`invite` 模式依赖 `registration_tickets`，没有邀请码时 signup 失败。

## 4. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/AdminSettings.vue` | 占位页改为表单：名称、简介、维护者、邮箱、注册模式（三选一），保存后提示刷新公开页 |
| `web/src/pages/AdminIndex.vue` | 管理组新增「实例设置」 |
| `web/src/api/endpoints.ts`、`types.ts` | `adminMeta`、`adminUpdateMeta` 与 `AdminInstanceMeta` |

## 5. 验收

`go vet`、`go test ./...`（新增 3 个用例：设置回读、运行时注册模式、校验与权限）、`vue-tsc`、
`check:port`、`pnpm build` 全绿；真实二进制走查改名/改简介、切到「仅管理员创建」后 signup 400、
切回开放注册后 signup 200。证据见 HANDOVER §12AC.3。
