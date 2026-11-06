# 第十九波施工契约：实例身份、合规与统计（2026-11-06）

关于页的「统计接口尚未提供」与只能改五个字段的实例设置都留了缺口。本波补齐实例身份/合规字段、
实例规则与公开 `stats`。

## 1. 运行时设置扩展

`instance_meta` 新增键：`tosUrl`、`privacyPolicyUrl`、`repositoryUrl`、`feedbackUrl`、
`inquiryUrl`、`impressumUrl`、`serverRules`（JSON 字符串数组）。

- `admin/meta` 返回这些字段（`tosUrl` 空时是 `null`），`serverRules` 恒为数组；
- `admin/update-meta` 接受它们：URL 为空或 `http(s)`、≤300；规则去空行、每条 ≤300、最多 20 条；
- `meta` 同步返回，填掉此前写死的空串与空数组。

## 2. 公开统计

`POST /api/stats`（新，无需登录）：

```json
{ "usersCount", "notesCount", "originalNotesCount", "originalUsersCount",
  "reactionsCount", "followingCount", "instancesCount", "driveFiles", "driveUsage" }
```

`usersCount` 是未冻结的本地账号；`instancesCount` 是 `users.host` 去重数；`driveUsage` 是
`drive_files.size` 之和。全部单条 SQL 子查询完成。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `AdminSettings.vue` | 新增六个 URL 字段与「实例规则（每行一条）」文本域 |
| `About.vue` | 合规链接列表（条款/隐私/源码/反馈/咨询/版本说明）、实例规则列表、真实统计数字（替换原占位说明） |
| `api/types.ts`、`endpoints.ts` | `AdminInstanceMeta` 扩展、`InstanceStats`、`stats` |

## 4. 有意的限制

1. 规则只在关于页展示，不进入注册流程（没有“同意规则”勾选）。
2. 统计是即时 `count(*)`，没有缓存；个人实例规模可接受。
3. `instancesCount` 统计的是库里有账号的远程实例，不是完整的联邦实例清单。
4. 仍然没有 `charts/*` 历史图表。

## 5. 验收

`go vet`、`go test ./...`（新增合规字段与统计用例）、`vue-tsc`、`check:port`、`pnpm build` 全绿；
真实二进制上管理页保存条款与规则、关于页展示链接/规则/统计。证据见 HANDOVER §12AI.3。
