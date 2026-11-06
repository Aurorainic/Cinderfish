# 第三十一波施工契约：实例图标、横幅与主题色（2026-11-06）

公开 `meta` 的 `iconUrl`/`bannerUrl`/`themeColor` 一直是硬编码常量，管理页也没有入口。本波把
它们做成运行时设置，无迁移、无新路由。

## 1. 存储与端点

`instance_meta` 新键 `iconUrl`、`bannerUrl`、`themeColor`。

- `admin/meta` 返回三者（空值 `null`）；
- `admin/update-meta`：URL 空或 `http(s)` ≤300；`themeColor` 空或 `#rrggbb`；
- 公开 `meta`：`iconUrl`/`bannerUrl` 空串，`themeColor` 未设置时回落 `#86b300`。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `AdminInstanceMeta` 与 update-meta 请求字段 |
| `pages/AdminSettings.vue` | 图标 URL、横幅 URL、主题色三个输入，随现有保存一起提交 |

消费方本来就在：`MkInstanceTicker`、关于页与 favicon 读 `instance.iconUrl`，欢迎页读横幅。

## 3. 有意的限制

1. 只接受 URL，不做文件上传/裁剪（实例图标要先上传到网盘再贴 URL）。
2. 主题色只写入 `meta`，前端主题仍由用户主题设置决定。
3. `backgroundImageUrl`、`logoImageUrl` 等其余图片字段仍为常量。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
