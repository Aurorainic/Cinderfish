# 第三十五波施工契约：从 URL 上传（2026-11-06）

网盘缺少 `drive/files/upload-from-url`。本波实现带 SSRF 防护的下载上传，并在网盘菜单加入口。

## 1. 端点

`POST /api/drive/files/upload-from-url`（需登录）请求 `{url, folderId?, name?}`：

- 仅 `http(s)`；
- 目标主机的所有解析地址必须为公网地址（拒绝环回/私有/链路本地/未指定）；
- 最多 5 次重定向，每一跳重新校验；
- 30 秒超时；响应体超过单文件上限（默认 20 MiB）返回 `FILE_TOO_LARGE`；
- 非 2xx 返回 `FETCH_FAILED`；`folderId` 必须属于调用者；
- 文件名取 `name` 或 URL 路径末段，再 `SanitizeName`。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/endpoints.ts` | `drive/files/upload-from-url` |
| `components/MkDrive.vue` | 菜单新增「从 URL 上传」，输入 URL 后调用接口并将文件插到列表前 |

## 3. 有意的限制

1. 没有进度条与取消。
2. 不跟随非 http(s) 重定向；不代理 Cookie。
3. 只校验地址，不做内容类型白名单。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
