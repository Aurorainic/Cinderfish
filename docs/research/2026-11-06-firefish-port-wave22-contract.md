# 第二十二波施工契约：资料扩展字段（2026-11-06）

资料设置页此前只支持 name/description/avatar 与两个可见性开关；所在地、生日、语言、附加信息
与一批布尔开关没有写入路径。本波把它们写进 `user_profiles.settings` 并回读。

## 1. `i/update` 新增字段

| 字段 | 校验 | 存储 |
|------|------|------|
| `location` | ≤ 50 字符 | `settings.location` |
| `birthday` | `YYYY-MM-DD` 或空 | `settings.birthday` |
| `lang` | ≤ 16，字母/数字/`-` | `settings.lang` |
| `fields` | ≤ 16 条，`{name,value}`；name ≤ 255、value ≤ 1024；空 name 丢弃 | `settings.fields` |
| `noCrawle`、`preventAiLearning`、`isExplorable`、`hideOnlineStatus`、`publicReactions`、`alwaysMarkNsfw`、`autoSensitive`、`carefulBot`、`autoAcceptFollowed`、`injectFeaturedNote` | boolean | `settings.<字段名>` |

语义沿用现有约定：字段不出现 = 不改动；出现 `null`/`""` = 清空。布尔值用 `*bool` 区分“未提供”
与 `false`。日期在 `settings` 里以字符串保存，不引入新列。

## 2. 打包

- `UserDetailed`：`location`/`birthday`/`lang` 空值打包为 `null`，`fields` 打包为
  `[{name,value}]`，`publicReactions` 读真实值。
- `MeDetailed`：`noCrawle`、`preventAiLearning`、`isExplorable`、`hideOnlineStatus`、
  `alwaysMarkNsfw`、`autoSensitive`、`carefulBot`、`autoAcceptFollowed`、`injectFeaturedNote`
  从 profile 读取；无 profile 行时 `isExplorable` 仍为 `true`，其余为 `false`。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`account/types.ts` | `ProfileField`、`UserDetailed` 四个字段、`ProfileUpdateRequest` 扩展 |
| `pages/SettingsProfile.vue` | 位置、生日（date 输入）、语言、附加信息列表（增删，最多 16 项），随保存一起提交 |
| `pages/SettingsOther.vue` | “在时间线显示推荐笔记”改为真实开关，调用 `i/update { injectFeaturedNote }` |

## 4. 有意的限制

1. 仍然没有横幅（`bannerId`）、锁推（`isLocked`）、关注审批、`isBot`/`isCat` 与发言猫语。
2. 没有邮箱修改与 `i/update-email`。
3. `fields` 不做 URL 验证（Misskey 只对带 `rel=me` 的链接做前端提示）。
4. 布尔与元数据全部存在 `settings` JSONB，不是独立列；直接查库时需要解 JSON。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查；保存与回读链路留待统一验证。
