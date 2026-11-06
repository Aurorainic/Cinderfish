# 第四十五波施工契约：管理台文件详情（2026-11-06）

`/admin/file/:fileId` 一直是个 17 行占位页，而后端也没有「按 id 看任意用户文件」的端点——
管理台的文件列表只能看不能点。本波补端点与页面。

## 1. 端点

`POST /api/admin/drive/show-file`（管理员）

| 参数 | 校验 |
|------|------|
| `fileId` | 必填；不存在 → 404 `NO_SUCH_FILE` |

返回单条 `DriveFile`（与列表同一 `packDriveFile`）+ `user`（`UserLite`）；上传者账号已不存在时
`user` 为 `null`。非管理员 403。

上游还返回上传 IP 与原始请求，本实现不存这两项数据，所以不返回。

## 2. 页面

`pages/AdminFile.vue`：

- 预览按内容类型分流：`image/*` 图片（敏感文件加模糊）、`video/*` 视频、`audio/*` 音频，
  其它显示占位块；
- 字段：名称（可复制）、id、类型、大小、上传时间、敏感标记、说明、链接；
- 上传者：用户名链接到个人页 + 账号 id；账号已删时给出说明；
- 页面底部明确写出「只读」的原因：`drive/files/delete` 只允许删除自己的文件，管理员没有删除
  他人文件的端点。

`pages/AdminFiles.vue` 每行末尾加「详情」链接进入本页。

## 3. 有意的限制

1. 没有删除或批量清理入口（无对应端点）。
2. 不显示上传 IP、原始请求、远端缓存状态。
3. 缩略图列恒为 `null`（缩略图管道仍未实现），预览直接用原图。

## 4. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

台账：137 路由 / 130 同名 / 310 未实现。未执行浏览器走查（2026-11-07 补：
`TestAdminDriveShowFile` 与四类预览走查，见 HANDOVER §12BI.4）。

关于 `user: null`：`drive_files.user_id` 是 `ON DELETE CASCADE`，账号删除还会先删云盘对象，
所以正常流程构造不出「文件在、用户不在」。该分支仅保留为防御代码，页面文案已改为
「账号删除会连同云盘文件一起删除」。
