# 第二十五波施工契约：云盘文件编辑（2026-11-06）

网盘组件此前明确注明：`drive/files/update` 不存在，文件不能重命名、加描述、标记敏感或移动文件夹，
文件磁贴也不可拖拽。本波补齐该端点，并把这些操作接进网盘界面。

## 1. 数据模型（迁移 v16）

```sql
ALTER TABLE drive_files ADD COLUMN comment TEXT;
```

`comment` 可空，读取时 `COALESCE(comment, '')`，打包空值仍为 `null`。

## 2. `POST /api/drive/files/update`

| 字段 | 类型 | 说明 |
|------|------|------|
| `fileId` | string | 必填 |
| `name` | string \| null | trim 后非空，≤ 255 字符 |
| `isSensitive` | boolean | 敏感标记 |
| `comment` | string \| null | ≤ 512 字符，空串清空 |
| `folderId` | string \| null | 自己的文件夹；`null`/`""` 移回根目录 |

- 仅作者可改：他人文件按 404 `NO_SUCH_FILE` 处理（不泄露存在性）。
- 文件夹不存在或属于别人 → 400 `NO_SUCH_FOLDER`。
- 响应是完整的 `DriveFile`。
- 实现：`internal/media/drive.go` 的 `FileUpdate`/`Update`，HTTP 在 `internal/httpapi/drive.go`。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `DriveFile.comment` 改为 `string \| null`；新增 `drive/files/update` |
| `components/MkDrive.file.vue` | 菜单新增重命名/描述/敏感开关；磁贴可拖拽；敏感角标 |
| `components/MkDrive.folder.vue` | 接收拖入的文件，调用 update 并发出 `file-moved` |
| `components/MkDrive.vue` | `updateItem` 原地替换磁贴，`file-moved` 时移除旧磁贴 |

## 4. 有意的限制

1. 面包屑（breadcrumb）仍不接受文件拖放，只能在网格里拖到文件夹磁贴。
2. 发帖框的附件列表仍只有“移除”，重命名/描述在网盘页完成。
3. 没有 `drive/files/upload-from-url`、去重查询与批量移动。
4. 文件移动不改 `notes.file_ids`，已发笔记的附件归属不受影响（本来也不需要）。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查；重命名/描述/敏感/拖拽移动留待统一验证。
