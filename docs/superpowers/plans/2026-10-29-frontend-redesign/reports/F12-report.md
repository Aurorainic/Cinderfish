# Task F12 报告：术语与收尾打磨

> 说明：本任务由子代理开始执行，中途 provider 额度耗尽，由主控代理接手完成并验证。

## 完成内容

1. **文案与术语 sweep**
   - 发帖器标题与占位符：`说点什么` → `发帖`，占位符改为「写点什么吧？」；回复/引用保持「写下回复…」「引用并评论…」。
   - 时间线频道名对齐 Misskey：关注 → 首页，混合 → 社交（deck 列头同名）。
   - 管理面板联邦模式说明改为「笔记/投递」措辞；ProfileView 空状态「去发现页」→「去探索页」。
2. **残留旧视觉**
   - `ProfileHeader` 横幅去掉双层 radial-gradient，改纯 `--panel-2` + `--divider` 下边框。
   - `main.css` 的 `.auth-aside` 去掉两个径向光斑，改 `--bg`；`.brand-mark` / `.instance-logo` / `UserAvatar` 金色渐变与辉光改成 `--accent` 实心 + `--on-accent` 文字。
   - `os.ts` / `PageWindow.vue` 的 `var(--shadow-sm, …)` 回退值删除（token 已存在）。
   - 全仓 `999px` 胶囊收敛为 `--radius` / `--radius-sm`；剩下的仅是小圆片/底栏指示器。
   - 空状态 `.placeholder` 图标缩小去强调色，标题统一样式。
3. **死代码清理**：`main.css` 删除 F2 搬家后失效的 `.side`、`.instance-head`、`.instance-actions`、`.link-list`、`.link-list a:hover`、`.side-foot`；同时删除已无引用的 `--nav-bg`、`--shadow`（grep 确认零引用）。
4. **Deck 打磨**：`.shell[data-shell="deck"]` 隐藏 widget 栏、列条吃满宽度；`TimelineFeed` 增加 `composer?: boolean`（默认 true），`DeckColumn` 全部传 `false`，350px 列里不再长出发帖器；侧栏新增全局「发帖」按钮，保证 deck 模式仍有发帖入口。
5. **emoji 名解析**：`lib/emoji-catalog.ts` 新增 `CHAR_BY_NAME` 索引与 `emojiChar(name)`（接受带冒号写法、大小写不敏感），`NotificationItem` 删除重复的三行遍历。
6. **头像回退字居中**：`NoteHeader` 的 `.avatar` 由 `display:block; line-height:0` 改为 `display:flex`，`UserAvatar` 回退字显式 `line-height:1`，无头像作者的首字母不再贴左上角。
7. **HANDOVER**：里程碑表新增 M6.1；§9 补 `i/update`、`i/pin`、`i/unpin`；§11 重写目录、页面清单、实时更新（引用计数订阅）与已知缺口（自定义 emoji、i18n、置顶联邦、单条草稿、deck 单层覆盖层等）。

## 验证

```
gofmt -l internal/ cmd/     # 空
go vet ./...                # 通过
CINDERFISH_TEST_DATABASE_URL=… CINDERFISH_TEST_REDIS_URL=… go test -count=1 ./...
                            # 全部 ok（account/httpapi/media/federation/streaming…）
cd web && pnpm typecheck    # 0 错误
cd web && pnpm build        # ✓ built in 503ms
```

## 遗留与判断

- 未做整页浏览器走查的明暗双主题截图核对：本轮子代理额度中断后由主控完成构建级验证，视觉走查依赖此前各任务的浏览器验证记录（F2/F5/F7/F9/F10/F11 报告）。
- `openReactionPicker` 的锚点定位、deck 覆盖层单层、草稿单条等已知限制已写入 HANDOVER §11.7，不在 F12 范围内修。
