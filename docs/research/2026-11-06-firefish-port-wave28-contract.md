# 第二十八波施工契约：网盘用量端点（2026-11-06）

网盘设置页此前靠逐页拉 `drive/files` 汇总用量，并明确写「drive 端点未提供」。本波补上
`drive` 端点与其页面接线。

## 1. 端点

`POST /api/drive`（需登录）：

```json
{ "capacity": 0, "usage": 123456, "filesCount": 12 }
```

- `usage` = `sum(drive_files.size)`，`filesCount` = 文件行数，一条 SQL 完成；
- Cinderfish 没有实例配额，`capacity` 恒为 `0`，客户端应读作“无限制”；
- `filesCount` 是 Misskey `DriveInfo` 没有的额外字段，供设置页显示。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `DriveUsage`、`drive` |
| `pages/SettingsDrive.vue` | 一次请求显示容量/用量/文件数；容量 0 显示“未设置配额（无限制）”；“默认敏感/自动敏感”改为真实开关（`i/update`） |

## 3. 有意的限制

1. 没有真实配额：容量恒 0，上传只受单文件大小限制。
2. 没有按天统计图表（chart.js 不是依赖）。
3. `filesCount` 偏离 Misskey 的响应形状（多一个字段，不影响客户端）。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
