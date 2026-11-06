# API 对接契约：drive-media

本文覆盖 Cinderfish 后端中 `drive-media` 命名空间下的全部端点：云盘文件、云盘文件夹、媒体文件直读、
以及公开的自定义表情列表。所有结论都来自本 worktree（`.worktrees/frontend-ui-redesign`）里的实际
handler 与服务层代码，不是从 Misskey 文档反推；与 Misskey `c05e844` 源码不一致的地方单独标出。

阅读方式：先看 `## 通用约定` 与 `## 共享对象形状`（分页、鉴权、错误信封、`DriveFile` /
`DriveFolder` / `EmojiSimple` 三个打包器的完整字段），再看具体端点；每个端点只描述自己额外的字段与错误。

> 命名空间边界：`admin/emoji/*`（`admin/emoji/list`、`add`、`delete`）虽然与 `POST /api/emojis` 共用
> `internal/httpapi/emojis.go` 与 `internal/media/emoji.go`，但在文档分工上属于 `admin-emoji-registry`
> 命名空间，本文只在“共享对象形状”与“已知缺口”里提到它，不重复契约细节。
> `hashtags/*` 属于 `timelines-notifications` 命名空间；本文只记录一个事实：`hashtags/show` **未实现**。

## 端点总览

| 方法与路径 | 用途 | 鉴权 | Handler |
|---|---|---|---|
| `POST /api/drive` | 容量、用量与文件数（`capacity=0` 表示未配置配额） | 必须登录 | `internal/httpapi/drive_usage.go:11` |
| `POST /api/drive/files` | 列出调用者云盘里的文件（按文件夹分层，`folderId` 缺省为根） | 必须登录 | `internal/httpapi/drive.go:103` |
| `POST /api/drive/files/create` | `multipart/form-data` 上传一个文件 | 必须登录（token 只能走请求头） | `internal/httpapi/drive.go:13` |
| `POST /api/drive/files/show` | 按 id 读取单个文件的元数据 | **匿名可读** | `internal/httpapi/drive.go:152` |
| `POST /api/drive/files/update` | 改名 / 描述 / 敏感标记 / 移动到文件夹 | 必须登录 | `internal/httpapi/drive.go:214` |
| `POST /api/drive/files/delete` | 删除自己的一个文件及其存储对象 | 必须登录 | `internal/httpapi/drive.go:181` |
| `POST /api/drive/folders` | 列出某文件夹下的子文件夹（缺省为根） | 必须登录 | `internal/httpapi/drive_folders.go:51` |
| `POST /api/drive/folders/create` | 新建文件夹 | 必须登录 | `internal/httpapi/drive_folders.go:98` |
| `POST /api/drive/folders/show` | 读取单个文件夹（含子项计数） | 必须登录 | `internal/httpapi/drive_folders.go:134` |
| `POST /api/drive/folders/update` | 改名 / 移动文件夹 | 必须登录 | `internal/httpapi/drive_folders.go:162` |
| `POST /api/drive/folders/delete` | 删除文件夹，子项上移到其父级 | 必须登录 | `internal/httpapi/drive_folders.go:211` |
| `POST /api/emojis` | 本实例自定义表情清单（`EmojiSimple`） | **匿名可读** | `internal/httpapi/emojis.go:53` |
| `GET /media/{path...}` | 直接读取媒体字节（经当前存储后端，见下） | **匿名可读** | `internal/httpapi/drive.go:215` |

路由注册位置：`internal/httpapi/server.go:187`–`196`（drive 与 media）、`internal/httpapi/server.go:147`
（`/api/emojis`）。全部为 `POST`，唯一例外是 `GET /media/{path...}`；没有任何 `GET /api/drive/*` 别名。

## 通用约定

### 传输与编码

- 所有 `/api/*` 成功响应由 `writeJSON`（`internal/httpapi/server.go:235`）写出：`Content-Type:
  application/json; charset=utf-8`，`json.Encoder` 且 `SetEscapeHTML(false)`（`<`、`>`、`&` 不转义）。
- 请求体为 JSON 时最多读取 8 MiB（`decodeOptionalJSON`，`internal/httpapi/ping.go:23`）；空 body 合法，
  按“所有字段缺省”处理；JSON 语法错误 → `400 INVALID_PARAM`（`info.reason = "body is not valid JSON"`）。
- 未在 handler 结构中声明的 JSON 字段被静默忽略，不会报错（例如 `drive/files` 的 `sort`、`type`）。

### 鉴权

`Server.authenticate`（`internal/httpapi/auth.go:30`）按固定顺序取 token：

1. 请求体 JSON 的 `i` 字段（字符串）；
2. `Authorization` 请求头，前缀 `Bearer `（大小写不敏感，`internal/httpapi/auth.go:38`）。

`i` 优先于请求头。token 解析规则（`internal/account/tokens.go:88`）：长度 16 的 token 视为原生用户
token（查 `users.token`），其余查 `access_tokens`（miAuth / OAuth 签发的 32 字符 token）。

- 两个来源都没有 token → handler 一律按“未登录”处理，返回 `401 AUTHENTICATION_FAILED`，并带
  `WWW-Authenticate: Bearer realm="Misskey", error="invalid_token", ...`（`internal/httpapi/server.go:304`）。
- token 无效或账号已被封禁（`is_suspended`）→ 同样 `401 AUTHENTICATION_FAILED`，封禁不单独区分
  （`internal/httpapi/auth.go:52`）。
- **本命名空间不校验 scope**：`read:drive` / `write:drive` 之类的 Misskey 权限位在 HTTP API 上没有被检查，
  任何有效 token 都能调用（只有 `/streaming` 检查 `read:account`，见 `internal/httpapi/streaming_adapters.go:30`）。
- **`POST /api/drive/files/create` 是例外**：它在 `ParseMultipartForm` 之前调用 `authenticate(..., nil)`
  （`internal/httpapi/drive.go:22`），因此**看不到 multipart 表单里的 `i` 字段**，token 必须放在
  `Authorization` 请求头。前端上传必须显式设置请求头。
- 匿名可读的端点：`POST /api/drive/files/show`、`POST /api/emojis`、`GET /media/{path...}`。

未接数据库 / 未接存储的实例（`httpapi.NewServer` 或 `SetStorage` 未调用）上，本命名空间的 handler 会先
返回 `503 INTERNAL_ERROR`（`{"code":"INTERNAL_ERROR","message":"Not available."}`），而不是 401：

- drive 文件相关：`s.drive == nil`（`internal/httpapi/drive.go:14`）；
- folder 相关：`requireFolders` → `s.folders == nil`（`internal/httpapi/drive_folders.go:31`）；
- `POST /api/emojis`：`s.emojis == nil`（`internal/httpapi/emojis.go:54`）；
- `GET /media/...`：`s.storage == nil` 时用的是 **404**，但 code 仍是 `INTERNAL_ERROR`
  （`internal/httpapi/drive.go:216`）。

### 错误信封

所有错误（含 404 兜底路由）都是同一个 JSON 形状（`internal/httpapi/server.go:269`）：

```json
{"error": {"message": "...", "code": "...", "id": "uuid-like", "kind": "client|server|permission", "info": "可选，仅 INVALID_PARAM 等带"}}
```

- `message`：人读字符串；`code`：机器判定用；`id`：Misskey 的错误 UUID，前端通常不依赖，但可用于对齐；
  `kind`：`client` / `server` / `permission`；`info` 只在 `HasInfo` 为真时出现（一般是 `INVALID_PARAM` 的
  原因字符串）。
- 未匹配的路径或方法落到 `notFoundHandler`（`internal/httpapi/server.go:327`），返回
  `404 NO_SUCH_ENDPOINT`，`message` 形如 `No such endpoint: GET /api/drive/files`。因为注册了 `/` 兜底，
  **用 GET 调 POST-only 端点不会得到 405，而是这个 JSON 404**（实测 Go `ServeMux` 落到兜底）。

本命名空间会出现的错误码：

| HTTP | code | 触发条件 | 定义 |
|---|---|---|---|
| 400 | `INVALID_PARAM` | JSON 解析失败、multipart 解析失败、缺 `file` 字段、名称超长、文件夹环、folder 参数指向不存在的文件夹（上传/列表路径） | `internal/httpapi/server.go:290` |
| 400 | `FILE_TOO_LARGE` | 上传内容超过 20 MiB | `internal/httpapi/drive.go:61` |
| 400 | `NO_SUCH_FOLDER` | `checkOwnFolder`：上传或列文件时给的 `folderId` 不存在 | `internal/httpapi/drive.go:83` |
| 403 | `ACCESS_DENIED` | `checkOwnFolder`：`folderId` 属于别人 | `internal/httpapi/drive.go:94` |
| 404 | `NO_SUCH_FOLDER` | folder 系列端点（create/show/update/delete）：文件夹不存在或不属于调用者 | `internal/httpapi/drive_folders.go:42` |
| 404 | `NO_SUCH_FILE` | 文件不存在、删除别人的文件、媒体 key 不存在 | `internal/httpapi/drive.go:172`、`:205`、`:235` |
| 401 | `AUTHENTICATION_FAILED` | 缺 token / token 无效 / 账号封禁 | `internal/httpapi/server.go:304` |
| 500 | `INTERNAL_ERROR` | 数据库或存储写入失败 | `internal/httpapi/streaming_publish.go:123`（`writeInternalServerError`） |
| 503 | `INTERNAL_ERROR` | 服务未接线（无 DB / 无存储 / 无表情服务） | 各 handler 顶部 |

注意两套 `NO_SUCH_FOLDER` 的 HTTP 状态不一致：上传/列表路径是 **400**，folder 系列端点是 **404**，二者
`id` 相同（`d74ab9eb-bb09-4bba-bf24-fb58f761e1e9`，见 `internal/httpapi/drive_folders.go:13`）。
`NO_SUCH_FILE` 统一使用 `b6992544-63e7-67f0-fa7f-32444b1b5306`，这个 id 在 Misskey 里其实是
`admin/emoji/add` 的 `NO_SUCH_FILE`，与上游 `drive/files/show`、`drive/files/delete` 的 id 不同。

### 分页游标（drive 文件与文件夹共用）

- `limit`：`int`，缺省/`<=0` 取 **10**，上限 **100**（超出被截到 100，不报错）。
  `internal/media/drive.go:182`、`internal/media/folder.go:92`。
- `sinceId`：只返回 `id > sinceId` 的行（更新的）；`untilId`：只返回 `id < untilId` 的行（更旧的）。
  比较是**字符串比较**（`id > $n` / `id < $n`），依赖 id 的时间可排序性。
- 排序恒为 `ORDER BY id DESC`（新的在前）；**没有** `sort` / `type` 参数，也不支持 `sinceDate` /
  `untilDate`（`drive/files` 连字段都不解析；`drive/folders` 解析后显式丢弃，见
  `internal/httpapi/drive_folders.go:75`）。
- id 由 `internal/idgen` 生成：10 位 base36 毫秒时间戳 + 8 位 base36 随机串，共 18 字符，字符串序即时间序。
  前端把列表最后一项的 `id` 传给 `untilId` 即可翻页（Misskey 客户端就是这么做的）。
- 列表端点**不返回总数、不返回是否有下一页**；`len(items) < limit` 才可判定到底。

### 时间格式（不统一，前端解析要兼容）

| 对象 | 字段 | 实际格式 | 出处 |
|---|---|---|---|
| `DriveFile` | `createdAt` | `2006-01-02T15:04:05.000Z`（UTC，毫秒，固定三位小数） | `internal/httpapi/drive.go:252` |
| `DriveFolder` | `createdAt` | `time.RFC3339Nano`（UTC，纳秒，可能带 7–9 位小数） | `internal/httpapi/drive_folders.go:19` |
| admin emoji | `createdAt` | `time.RFC3339Nano` | `internal/httpapi/emojis.go:47` |

`new Date(str)` 三者都能解析，但字符串**不保证**逐字节相等，前端不要做字符串相等比较。

### CORS

`/api/*` 与 `/media/*` 没有任何 `Access-Control-Allow-*` 响应头（只有 ActivityPub 的
`internal/httpapi/activitypub.go:49` 起有 `*`）。前端必须与后端同源，或走同源反向代理 / 开发服务器代理
（`/api`、`/media` 转发到 3000），不能从另一个源直接 `fetch`。

## 共享对象形状

### `DriveFile`（`packDriveFile`，`internal/httpapi/drive.go:249`）

上传、列出、展示、删除前后返回的都是这一个形状；`notes/*` 的 `files` 数组也复用它
（`internal/httpapi/notes_pack.go:40`）。

| 字段 | 类型 | 实际取值 | 说明 |
|---|---|---|---|
| `id` | string | 18 字符 id | 文件 id，分页游标用它 |
| `createdAt` | string | `...000Z` | 见上表 |
| `name` | string | 文件名 | 已去路径成分；缺省回退 `"file"` |
| `type` | string | MIME | 数据库列 `content_type` |
| `size` | number | 字节数 | `int64`，上限 20 MiB |
| `isSensitive` | boolean | **恒 `false`** | 上传接口不接受 `isSensitive`，库列从未被写入 |
| `url` | string | `"/media/ab/cd/<sha256>"` 或 S3 公网 URL | 本地后端是**相对路径**，前端需拼 origin |
| `thumbnailUrl` | null | **恒 null** | 没有缩略图生成 |
| `webpublicUrl` | null | **恒 null** | 没有 webp 处理 |
| `comment` | null | **恒 null** | 上传接口不接受 `comment` |
| `folderId` | string \| null | 所属文件夹 | 根目录为 `null` |
| `blurhash` | null | **恒 null** | 未计算 |

**未返回**（Misskey `DriveFile` 有、这里没有）：`md5`、`properties`（宽高/朝向等）、`folder`（嵌套
`DriveFolder`）、`userId`、`user`。前端不要读这些字段；列表里也没有 `folder`，要面包屑只能逐级调
`drive/folders/show`。

关于 `url`：

- 本地后端：`LocalStorage.URL` = `"/media/" + key`（`internal/media/local.go:77`），前缀在
  `cmd/cinderfish/main.go:196` 写死为 `/media`。
- S3 后端：`PublicBaseURL` 非空时为 `<PublicBaseURL>/<key>`，否则为
  `{http|https}://<endpoint>/<bucket>/<key>`（`internal/media/s3.go:105`）。`cmd/cinderfish/main.go:185`
  构造 `S3Options` 时**没有**设置 `Presign`，所以当前部署不会出现会过期的签名 URL；如果以后打开
  `Presign`，`url` 每次请求都会变化且有 TTL，前端必须每次重新取元数据而不是缓存 URL。

### `DriveFolder`（`packDriveFolder`，`internal/httpapi/drive_folders.go:16`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 文件夹 id |
| `createdAt` | string | `RFC3339Nano` |
| `name` | string | 名称；创建/改名时按**字节**长度校验上限 200，创建时空名回退 `"Untitled"` |
| `parentId` | string \| null | 父文件夹，根为 `null` |
| `foldersCount` | number | **直接**子文件夹数 |
| `filesCount` | number | **直接**文件数 |
| `parent` | null | **恒 null**，包括 `folders/show`（Misskey 的 detail 打包会填嵌套父级） |

计数只统计一层（`internal/media/folder.go:223` 的 `folderSelect`），不是递归总量。

### `EmojiSimple`（`packEmojiSimple`，`internal/httpapi/emojis.go:24`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string | shortcode，匹配 `^[a-zA-Z0-9_]+$` |
| `category` | string \| null | 分类名 |
| `aliases` | string[] | 别名，无别名时为 `[]`（不是 `null`） |
| `url` | string | 注册时固化的图片 URL（本地后端是 `/media/...` 相对路径） |
| `localOnly` | boolean | 本地表情恒 `true`（`internal/media/emoji.go:64`） |
| `isSensitive` | boolean | 本地注册恒 `false`（`admin/emoji/add` 不读取该参数） |

与 Misskey `EmojiSimple` 的差异：Misskey 只在为真时输出 `localOnly` / `isSensitive`，并可能带
`roleIdsThatCanBeUsedThisEmojiAsReaction`；这里两个布尔**恒输出**，且**没有** `roleIds...` 字段。

## POST /api/drive

需登录。返回：

```json
{ "capacity": 0, "usage": 123456, "filesCount": 12 }
```

`usage` = `sum(drive_files.size)`，`filesCount` 是 Misskey 没有的额外字段（设置页显示用）。
Cinderfish 没有实例配额，`capacity` 恒为 `0`，客户端应读作“无限制”。

## POST /api/drive/files

- 路径与方法：`POST /api/drive/files`（无别名，`internal/httpapi/server.go:193`）。
- 鉴权：必须登录，未登录 `401`。

请求体（JSON，全部可选）：

| 字段 | 类型 | 必填 | 默认 | 校验 / 语义 |
|---|---|---|---|---|
| `limit` | int | 否 | 10 | `<=0` 视为 10，`>100` 截到 100 |
| `sinceId` | string | 否 | — | 只取 `id > sinceId` |
| `untilId` | string | 否 | — | 只取 `id < untilId` |
| `folderId` | string \| null | 否 | null（根） | 缺省或 `null` → 只列根文件（`folder_id IS NULL`）；非空 → 必须是自己的文件夹 |

未实现的 Misskey 参数：`sinceDate`、`untilDate`、`type`（MIME 过滤）、`sort`（按名/大小/时间）——传了也
被静默忽略。

响应：`200`，`DriveFile` 数组，按 `id` 降序；无内容时是 `[]` 而不是 `null`
（`out := make([]any, 0, len(files))`，`internal/httpapi/drive.go:144`）。

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_PARAM` | body 不是合法 JSON |
| 400 | `NO_SUCH_FOLDER` | `folderId` 不为空但不存在（包括传 `""`：会走 `GetFolder("")` 并失败） |
| 403 | `ACCESS_DENIED` | `folderId` 存在但属于别的用户 |
| 401 | `AUTHENTICATION_FAILED` | 未登录 / token 无效 / 封禁 |
| 500 | `INTERNAL_ERROR` | 查询失败 |
| 503 | `INTERNAL_ERROR` | 服务未接线（`s.drive == nil`，即未接存储或未接 DB） |

要点与差异：

- 只列**当前一层**；前端做“全部文件”视图必须递归文件夹或自行按 `folderId` 分组。
- `folderId: ""` 与“不传”语义不同：不传是根目录，传空串会 400 `NO_SUCH_FOLDER`。
- 无类型/时间过滤、无排序参数，全部被忽略，不会报错——前端若依赖 `sort` 会静默失效。
- 引用：`internal/httpapi/drive.go:103`–`149`；分页实现 `internal/media/drive.go:181`。

## POST /api/drive/files/create

- 路径与方法：`POST /api/drive/files/create`（无别名，`internal/httpapi/server.go:187`）。
- 鉴权：必须登录，且 **token 必须在 `Authorization: Bearer` 头里**（见通用约定）。

请求：`multipart/form-data`（`ParseMultipartForm(32 << 20)`，内存里最多 32 MiB，超出部分落临时文件）。

| 表单字段 | 类型 | 必填 | 默认 | 校验 / 语义 |
|---|---|---|---|---|
| `file` | 文件 | 是 | — | 缺失 → `400 INVALID_PARAM`，`info.reason = "no file field"`；文件名经 `media.SanitizeName` 去路径 |
| `folderId` | text | 否 | 空 = 根 | 空串视为根；非空则必须是自己的、存在的文件夹 |

被忽略的 Misskey 参数：`name`（重命名）、`comment`、`isSensitive`、`force`。请求里给了也不会生效。

内容处理（`internal/media/drive.go:66`）：

- 文件名：`path.Base` 去掉路径、`TrimSpace`；为空、`.`、`..`、`/` 回退 `"file"`。**没有**上游的名称
  校验（长度 ≤ 200、禁止 `\`、`/`、`..`）——这里直接剥掉路径成分，所以不会报 `INVALID_FILE_NAME`。
- MIME：取 multipart 分片的 `Content-Type`；为空时按扩展名推断；再不行 `application/octet-stream`。
  **没有**类型白名单（上游会拒 `UNALLOWED_FILE_TYPE`）。
- 大小上限：`DefaultMaxFileSize = 20 MiB`（`internal/media/drive.go:23`）。`SetMaxFileSize` 存在但
  `cmd/cinderfish/main.go` 从未调用，配置里也没有对应项，所以**客户端无法通过配置调大**。
- 存储 key：`sha256` 摘要的前 2/4 位作目录 + 完整摘要（`internal/media/drive.go:96`）。相同字节的上传
  共用同一个存储对象，但**每次上传仍各插一行** `drive_files`。

响应：`200`，单个 `DriveFile`。

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_PARAM` | 不是 multipart、缺 `file` 字段、multipart 解析失败 |
| 400 | `FILE_TOO_LARGE` | 内容超过 20 MiB（上游用 413 `MAX_FILE_SIZE_EXCEEDED`） |
| 400 | `NO_SUCH_FOLDER` | `folderId` 不存在 |
| 403 | `ACCESS_DENIED` | `folderId` 属于别人 |
| 401 | `AUTHENTICATION_FAILED` | 缺/错 token（含“token 只放在表单 `i` 里”的情况） |
| 500 | `INTERNAL_ERROR` | 存储写入或入库失败 |
| 503 | `INTERNAL_ERROR` | 未接存储 |

要点与差异：

- **返回的文件 `isSensitive` 恒 `false`**，`comment`/`thumbnailUrl`/`webpublicUrl`/`blurhash` 恒 `null`。
- 没有配额（`NO_FREE_SPACE`）、没有内容审核（`INAPPROPRIATE`）、没有 `force` 语义。
- **去重是双向的**：同内容共享一个对象；`drive/files/delete` 删任一行都会删掉共享对象，导致另一行
  （可能属于别的用户）的 `url` 变成 404。这一点与 Misskey 的“引用计数 + force 强制新副本”不同，
  前端如果要“删了还能再传”，注意这是服务端缺陷而不是竞态。
- 引用：`internal/httpapi/drive.go:13`–`75`；`internal/media/drive.go:66`、`:23`、`:52`。

## POST /api/drive/files/show

- 路径与方法：`POST /api/drive/files/show`（无别名，`internal/httpapi/server.go:194`）。
- 鉴权：**不需要凭证**。handler 完全不调用 `authenticate`，任何人拿到 id 都能读元数据。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 |
|---|---|---|---|---|
| `fileId` | string | 名义必填，实际不校验 | — | 缺失或空串按 `""` 查库，结果是 404 |

不支持的 Misskey 用法：上游 `paramDef` 是 `anyOf(fileId | url)`，这里**只认 `fileId`**，`url` 会被忽略。

响应：`200`，单个 `DriveFile`（形状同上）。

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | `NO_SUCH_FILE` | id 不存在 / 空 id / 已删除 |
| 400 | `INVALID_PARAM` | body 不是合法 JSON |
| 503 | `INTERNAL_ERROR` | 未接存储 |

要点与差异：

- 上游要求登录，且非管理员只能看自己的文件（否则 `ACCESS_DENIED`）；这里**既不要登录也不校验归属**，
  是一个隐私面（所有文件名、大小、类型、URL 对所有知道 id 的人可见）。
- 错误体里的 `id` 固定为 `b6992544-63e7-67f0-fa7f-32444b1b5306`；上游该端点用 `067bc436-2718-4795-b0fb-ecbe43949e31`。
- 引用：`internal/httpapi/drive.go:152`–`178`；服务层 `internal/media/drive.go:131`。

## POST /api/drive/files/delete

- 路径与方法：`POST /api/drive/files/delete`（无别名，`internal/httpapi/server.go:195`）。
- 鉴权：必须登录，只能删自己的文件。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 |
|---|---|---|---|---|
| `fileId` | string | 是（缺失不报 `INVALID_PARAM`） | — | 缺失/空 → 404 `NO_SUCH_FILE` |

响应：`200`，空对象 `{}`（上游是 `204 No Content`）。

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | `NO_SUCH_FILE` | 文件不存在、**是别人的文件**、或存储对象删除失败 |
| 400 | `INVALID_PARAM` | body 不是合法 JSON |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 503 | `INTERNAL_ERROR` | 未接存储 |

要点与差异：

- 删别人的文件返回 `404 NO_SUCH_FILE` 而不是上游的 `403 ACCESS_DENIED`（不泄露存在性）。
- 顺序是“先删库行，再删存储对象”（`internal/media/drive.go:238`）；如果存储删除失败，行已经没了，
  但接口仍返回 404 `NO_SUCH_FILE`，前端重试也只会得到 404——可以按“已删除”处理。
- 共享对象问题见 `drive/files/create` 的说明。
- 文件被笔记引用时不会级联清理 `notes.file_ids`；笔记打包会跳过查不到的文件（`files` 数组变短），
  不会报错。前端渲染历史笔记时可能遇到 `fileIds` 比 `files` 多。
- 引用：`internal/httpapi/drive.go:181`–`212`；`internal/media/drive.go:230`。

## POST /api/drive/files/update

- 路径与方法：`POST /api/drive/files/update`（无别名）。
- 鉴权：必须登录；只能改自己的文件。
- 请求体（所有字段可选，不出现 = 不改）：

| 字段 | 类型 | 校验 |
|---|---|---|
| `fileId` | string | 缺失 → 400 `INVALID_PARAM` |
| `name` | string \| null | trim 后非空、≤ 255 字符 |
| `isSensitive` | boolean | — |
| `comment` | string \| null | ≤ 512 字符；空串清空 |
| `folderId` | string \| null | 必须是自己的文件夹；`null`/`""` 移回根目录 |

响应：`DriveFile`（`comment` 现在是字符串或 `null`，不再是常量 `null`）。

| HTTP | code | 触发 |
|---|---|---|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `INVALID_PARAM` | 缺 `fileId`、名称/描述非法 |
| 400 | `NO_SUCH_FOLDER` | `folderId` 不存在或属于别人 |
| 404 | `NO_SUCH_FILE` | 文件不存在或属于别人 |

要点：名称/描述的校验在服务层（`internal/media/drive.go` 的 `FileUpdate`）；文件夹归属在更新前
单独查询。`comment` 存在 `drive_files.comment`（迁移 v16），`packDriveFile` 空值打包为 `null`。

## POST /api/drive/folders

- 路径与方法：`POST /api/drive/folders`（无别名，`internal/httpapi/server.go:188`）。
- 鉴权：必须登录。

请求体（全部可选）：

| 字段 | 类型 | 必填 | 默认 | 校验 / 语义 |
|---|---|---|---|---|
| `limit` | int | 否 | 10 | `<=0` → 10，`>100` → 100 |
| `sinceId` | string | 否 | — | `id > sinceId` |
| `untilId` | string | 否 | — | `id < untilId` |
| `folderId` | string \| null | 否 | null（根） | 缺省/`null` → 根下的子文件夹；非空 → 必须是自己的 |
| `sinceDate` | int | 否 | — | **解析后丢弃**（`internal/httpapi/drive_folders.go:75`） |
| `untilDate` | int | 否 | — | 同上 |

响应：`200`，`DriveFolder` 数组，`id` 降序，空结果是 `[]`。

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_PARAM` | body 不是合法 JSON |
| 400 | `NO_SUCH_FOLDER` | `folderId` 非空但不存在（含 `""`） |
| 403 | `ACCESS_DENIED` | `folderId` 属于别人 |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 500 | `INTERNAL_ERROR` | 查询失败 |
| 503 | `INTERNAL_ERROR` | 未接 DB |

要点：

- 只列**直接子文件夹**，不递归；`foldersCount` / `filesCount` 已在每项里给出，可直接做“有内容”标记。
- 每项的 `parent` 恒 `null`；要显示祖先路径需要顺着 `parentId` 调 `drive/folders/show`。
- 引用：`internal/httpapi/drive_folders.go:51`–`95`；服务层 `internal/media/folder.go:91`。

## POST /api/drive/folders/create

- 路径与方法：`POST /api/drive/folders/create`（无别名，`internal/httpapi/server.go:189`）。
- 鉴权：必须登录。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 / 语义 |
|---|---|---|---|---|
| `name` | string | 否 | 空串 → `"Untitled"` | 长度按**字节**计 `> 200` → `400 INVALID_PARAM`（`len(body.Name)`） |
| `parentId` | string \| null | 否 | null（根） | 非空时必须存在且属于调用者 |

响应：`200`，单个 `DriveFolder`（新建时 `foldersCount` / `filesCount` 为 0，`parent` 为 `null`）。

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_PARAM` | body 非法 JSON；`name` 超过 200 字节 |
| 404 | `NO_SUCH_FOLDER` | `parentId` 不存在或属于别人 |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 500 | `INTERNAL_ERROR` | 入库失败 |
| 503 | `INTERNAL_ERROR` | 未接 DB |

要点与差异：

- 名称长度用字节数而非字符数（上游按 JS string length 计）；纯中文名约 67 字就会触顶。前端如需本地
  校验，用 UTF-8 字节长度而不是 `String.length`。
- 空名不报错，落库为 `"Untitled"`（上游默认值相同）。
- `parentId: ""` 不等于“根”：它会被当成一个真实 id 去查，查不到 → 404 `NO_SUCH_FOLDER`。建在根下请
  **省略** `parentId` 或传 `null`（与 `folders/update` 里 `""` 表示“移到根”的语义不同）。
- 没有向 Streaming 的 `drive` 频道发 `folderCreated` 事件（本实现完全没有 drive 频道，见“已知缺口”）。
- 引用：`internal/httpapi/drive_folders.go:98`–`131`；`internal/media/folder.go:46`。

## POST /api/drive/folders/show

- 路径与方法：`POST /api/drive/folders/show`（无别名，`internal/httpapi/server.go:190`）。
- 鉴权：必须登录，只能读自己的文件夹。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 |
|---|---|---|---|---|
| `folderId` | string | 是 | — | 缺失/空 → 查库失败 → 404 `NO_SUCH_FOLDER`（不是 `INVALID_PARAM`） |

响应：`200`，单个 `DriveFolder`，`foldersCount` / `filesCount` 为该层直接子项数。

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | `NO_SUCH_FOLDER` | 不存在或不属于调用者（两者不区分） |
| 400 | `INVALID_PARAM` | body 非法 JSON |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 503 | `INTERNAL_ERROR` | 未接 DB |

要点与差异：上游 `show` 用 `detail: true` 打包会带嵌套 `parent` 对象；这里 `parent` **恒为 `null`**，
前端必须自己按 `parentId` 再查（面包屑 N+1）。

引用：`internal/httpapi/drive_folders.go:134`–`157`；`internal/media/folder.go:77`。

## POST /api/drive/folders/update

- 路径与方法：`POST /api/drive/folders/update`（无别名，`internal/httpapi/server.go:191`）。
- 鉴权：必须登录，只能改自己的文件夹。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 / 语义 |
|---|---|---|---|---|
| `folderId` | string | 是 | — | 缺失/空/不存在/非本人 → 404 `NO_SUCH_FOLDER` |
| `name` | string \| null | 否 | 不改 | 字段类型是 `*string`：`null` 与缺省等价（不改）；`""` 把名字清成空串；长度 `> 200` 字节 → 400 |
| `parentId` | string \| null | 否 | 不改 | **缺省 = 不动**；`null` = 移到根；`""` 也视为移到根；非空 = 移到该父级 |

“缺省 vs 显式 null”由 `nullableField` 区分（`internal/httpapi/i_update.go:23`）。

响应：`200`，更新后的单个 `DriveFolder`。两个字段都没给时直接返回当前行（不报错、不发 UPDATE）。

| HTTP | code | 触发条件 |
|---|---|---|
| 400 | `INVALID_PARAM` | body 非法 JSON；`name` 超长；**移动到自己或自己的后代**（`info = "a folder cannot be moved into itself"`） |
| 404 | `NO_SUCH_FOLDER` | `folderId` 或新 `parentId` 不存在 / 非本人 |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 500 | `INTERNAL_ERROR` | 更新失败 |
| 503 | `INTERNAL_ERROR` | 未接 DB |

要点与差异：

- 环检测是沿新父级向上走（`internal/media/folder.go:158`），比“只判自指”更严格；移动整棵子树到自身
  之下会被 400 拒绝。
- **`name` 与 `parentId` 的“null 语义”不同**：`parentId` 用 `nullableField` 能区分“缺省”和“显式
  null”，所以 `null` = 移到根；`name` 是普通 `*string`，`null` 和缺省一样表示“不改”，只有 `""`
  才会把名字清空。两者都不给时直接返回当前行（不报错、不发 UPDATE）。清空后的名字不会再回退
  `"Untitled"`（回退只发生在 create）。
- 没有 `folderCreated` / `folderUpdated` 之类的流事件。
- 引用：`internal/httpapi/drive_folders.go:162`–`207`；`internal/media/folder.go:136`。

## POST /api/drive/folders/delete

- 路径与方法：`POST /api/drive/folders/delete`（无别名，`internal/httpapi/server.go:192`）。
- 鉴权：必须登录，只能删自己的文件夹。

请求体：

| 字段 | 类型 | 必填 | 默认 | 校验 |
|---|---|---|---|---|
| `folderId` | string | 是 | — | 缺失/空 → 404 `NO_SUCH_FOLDER` |

响应：`200`，空对象 `{}`（上游 `204`）。

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | `NO_SUCH_FOLDER` | 不存在或不属于调用者（不区分） |
| 400 | `INVALID_PARAM` | body 非法 JSON |
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 500 | `INTERNAL_ERROR` | 事务失败 |
| 503 | `INTERNAL_ERROR` | 未接 DB |

要点：

- 语义与 Misskey 一致：一个事务里先把 `drive_files.folder_id` 和子文件夹的 `parent_id` 改成被删文件夹的
  `parent_id`（根就是 `NULL`），再删行（`internal/media/folder.go:211`–`218`）。**不删内容**。
- 前端删完应重新拉父级列表 + 受影响文件列表。
- 引用：`internal/httpapi/drive_folders.go:211`–`238`。

## POST /api/emojis

- 路径与方法：`POST /api/emojis`（无别名，`internal/httpapi/server.go:147`）。
- 鉴权：**不需要凭证**（handler 不调用 `authenticate`）。

请求体：无参数被读取（Misskey 上游 `paramDef` 也是空对象）。传 `i` 或任何字段都不会影响结果。

响应：`200`

```json
{"emojis": [{"name": "...", "category": "cats", "aliases": ["..."], "url": "/media/...", "localOnly": true, "isSensitive": false}]}
```

- 顶层是对象，数组在 `emojis` 键下（不是直接返回数组）。
- 只含**本地**表情（`host IS NULL`），按 `category ASC NULLS FIRST, name ASC` 排序
  （`internal/media/emoji.go:81`）。
- 无分页，一次返回全部；实例表情多时响应会很大。
- `url` 是注册当时固化的地址：本地后端为相对路径 `/media/...`，前端需拼 origin。

| HTTP | code | 触发条件 |
|---|---|---|
| 500 | `INTERNAL_ERROR` | 查询失败 |
| 503 | `INTERNAL_ERROR` | 未接表情服务（`s.emojis == nil`） |

要点与差异：

- 上游该端点 `allowGet: true` 且带 `cacheSec: 3600`；这里**只注册了 POST**，`GET /api/emojis` 会落到
  兜底 404 `NO_SUCH_ENDPOINT`，且没有任何 `Cache-Control`。
- 表情图片如果是从 S3 迁移到本地（或反向）导致 `storage` 前缀变化，`url` 会失效：注册时把
  `drive_file.url` 拷进了 `emojis.url`（`internal/httpapi/emojis.go:157`），数据库迁移不会改写这一列。
- 表情注册/删除入口属于 `admin-emoji-registry` 命名空间（`POST /api/admin/emoji/add` 等，见
  `internal/httpapi/emojis.go:113`、`:173`）。
- 引用：`internal/httpapi/emojis.go:53`–`72`；`internal/media/emoji.go:81`。

## GET /media/{path...}

- 路径与方法：`GET /media/{path...}`（`internal/httpapi/server.go:196`）；`HEAD` 由 Go `ServeMux`
  按 GET 匹配，同样可用。
- 鉴权：**匿名**。这是 `DriveFile.url` 指向的地址。

请求：路径的剩余部分就是存储 key，例如 `/media/ab/cd/<64 位 sha256>`。没有查询参数被读取
（不支持 `?thumbnail`、`?webpublic` 之类的 Misskey 变体）。

响应：

- 成功：`200`，响应体是原始字节；`Content-Type` **恒为** `application/octet-stream`
  （`internal/httpapi/drive.go:243`），不反映真实 MIME。
- handler 不设置 `Content-Length`（`io.Copy` 直写）；超过 Go 内部 2 KiB 首包缓冲的响应会走 chunked，
  很小的文件可能被自动补上长度。没有 `ETag`、`Last-Modified`、`Cache-Control`，也**不支持 `Range`**
  （不返回 `Accept-Ranges`）。后果：`<video>` / `<audio>` 无法拖动进度条；没有验证器，服务端无法回答
  304，缓存完全交给浏览器的启发式策略。

| HTTP | code | 触发条件 |
|---|---|---|
| 404 | `NO_SUCH_FILE` | key 为空（`/media/`）或对象在当前存储后端里不存在 |
| 404 | `INTERNAL_ERROR` | 未接存储（`s.storage == nil`，`internal/httpapi/drive.go:216`） |
| 307 | — | `GET /media`（无尾斜杠）被 `ServeMux` 重定向到 `/media/`，响应体是 HTML 而不是 JSON |

要点与差异：

- 这条路由从**当前配置的 Storage** 读对象（`internal/httpapi/drive.go:233`）：本地后端直接开文件；
  S3 后端会把对象从 bucket 下载后经应用转发（多一跳，且依赖应用侧的访问密钥）。S3 部署下
  `DriveFile.url` 通常直接指向 bucket，并不会走这条路由，两条路径的可用性需要分别验证。
- 越权/穿越已防护：`LocalStorage.path` 用 `filepath.Clean("/"+key)` 归一化并拒绝逃出根目录
  （`internal/media/local.go:82`）；key 是内容哈希，不可枚举。
- 没有尺寸变换（无缩略图 / 无 webp），也没有防盗链。
- `/media` 前缀被显式排除在 SPA 兜底之外（`internal/httpapi/web.go:47`），不会被前端路由吞掉。
- 引用：`internal/httpapi/drive.go:215`–`246`；`internal/media/local.go:57`、`:77`、`:82`。

## 已知缺口与 Misskey 差异

按“写前端时会撞到”的顺序排列。每一条都能在代码里定位，不是猜测。

### 云盘文件

1. **没有缩略图 / webpublic**：`thumbnailUrl`、`webpublicUrl`、`blurhash` 恒 `null`，`thumbnail_key`
   列从未被写入。前端只能用 `url` 原图做网格，注意流量。
2. **没有 `md5` / `properties`**：`DriveFile` 缺这两个字段（`properties.width/height` 常被前端用于
   占位防抖动，这里拿不到，需要在 `<img>` `load` 后自行测量）。另外库列名叫 `md5`，实际存的是
   **SHA-256** 摘要（`internal/media/drive.go:95`），即便将来输出也不是真 MD5。
3. **`isSensitive` 不可设置**：`drive/files/create` 不接受该参数，`is_sensitive` 永远 `false`。
4. **`comment` 不可设置**，恒 `null`。
5. **缺少端点**：`drive`（容量/用量）、`drive/stream`、`drive/files/update`（改名/移动/标记敏感）、
   `drive/files/find`、`drive/files/find-by-hash`、`drive/files/upload-from-url`、
   `drive/files/attached-notes`、`drive/files/attached-chat-messages`、`drive/files/move-bulk`、
   `drive/files/check-existence`、`drive/folders/find` 均未实现
   （对照 `docs/research/2026-11-04-misskey-api-coverage.md` 的 `drive/*` 缺口 16 项）。
   前端的“剩余空间”条、跨文件夹移动、按哈希去重提示都要先降级。
6. **没有配额**：上游的 `NO_FREE_SPACE`、`MAX_FILE_SIZE_EXCEEDED`、`INAPPROPRIATE`、
   `UNALLOWED_FILE_TYPE`、`INVALID_FILE_NAME` 全都不存在；对客户端来说唯一的上传失败原因是
   `FILE_TOO_LARGE`（400）与存储错误（500）。
7. **上传大小固定 20 MiB**，无配置项（`SetMaxFileSize` 无调用方）。
8. **`drive/files` 无 `type` / `sort` / 日期过滤**，静默忽略。
9. **`drive/files/show` 匿名可读且不校验归属**（上游需要登录 + 本人/管理员）。
10. **删除别人的文件返回 404 而非 403**；存储删除失败也返回 404。
11. **内容去重会误删**：同内容共享一个存储对象，删任一行会删掉对象，其他行 `url` 失效。

### 文件夹

12. `DriveFolder.parent` **恒 `null`**，`folders/show` 也不填（上游 detail 会填递归父级）。面包屑要多次
    请求。
13. 名称长度按**字节**判断 200，不是字符数；超长错误是 `INVALID_PARAM` 而非上游的 schema 校验。
14. 无 `drive/folders` 的递归/全树接口，只能逐层列。
15. `sinceDate` / `untilDate` 解析后丢弃（签名不支持日期游标）。

### 表情

16. **`hashtags/show` 未实现**（`internal/httpapi/hashtags.go` 只有 `trend`，属
    `timelines-notifications` 命名空间）——标签页若要“某标签的详细统计”只能用 `hashtags/trend` 或本地统计。
17. `POST /api/emojis` 无 `GET`、无分页、无缓存头；`EmojiSimple` 无
    `roleIdsThatCanBeUsedThisEmojiAsReaction`。
18. 表情条目没有 `license` 字段；`admin/emoji/add` 会忽略 `isSensitive`、`localOnly`、
    `license`、`roleIdsThatCanBeUsedThisEmojiAsReaction`（`internal/httpapi/emojis.go:157` 固定传
    `false`），所以本地表情恒 `localOnly: true`、`isSensitive: false`。
19. 表情不使用 `webpublicUrl`，直接引用上传文件的 `url`。

### 媒体服务与传输

20. **`Content-Type` 恒 `application/octet-stream`**，且没有 `ETag` / `Cache-Control` / `Range`，
    handler 也不设 `Content-Length`。`<img>` 一般仍能按内容嗅探解码，但视频不可 seek、音频体验差，
    且缺少缓存验证器，前端最好自己按 `url` 做本地缓存。
21. **媒体地址与后端耦合**：本地后端是相对路径 `/media/...`（同源），S3 后端是绝对 URL（跨源、
    可能是 bucket 域名）。前端拼接图片地址时要处理“已是绝对 URL”和“相对路径”两种情况。
22. **`/api` 与 `/media` 没有 CORS 头**，跨源前端不可用。
23. **没有 drive 相关的 Streaming 事件**：Misskey 的 `drive` 频道（`fileCreated`、`fileDeleted`、
    `folderCreated`、`folderUpdated`、`folderDeleted` 等）在本实现中不存在；`internal/streaming` 只做了
    main / homeTimeline / localTimeline / hybridTimeline / globalTimeline 五个频道，源码注释里明确把
    `drive` 列为“故意缺席”（`internal/streaming/channel_main.go:13`）。上传/删除后需要自己刷新列表。
24. `POST /api/drive/files/create` 只接受 `Authorization` 头，表单 `i` 无效——这是与 Misskey 客户端习惯
    最容易踩的差异。
