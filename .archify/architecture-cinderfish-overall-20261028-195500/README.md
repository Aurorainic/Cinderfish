# 项目整体结构图（随进度更新）

- 产出：`cinderfish-overall.html`
- 源：`candidate.json`（Archify architecture schema v1）
- 绘制时基线：`db44d20` 之后的 M1–M4 完成状态

## 为什么分成两个区域

这张图表达的是**项目整体**，但 M5（联邦）、M6（前端）、M7（打磨）尚无可运行代码。如果把规划中的模块画成和已实现模块一样，图就会撒谎。因此：

- **已实现（M1–M4）**区域：`cmd/cinderfish`、`httpapi`、`account/auth`、`note`、`streaming`、`media`、`config/idgen`、PostgreSQL、媒体后端——都有 `go list ./...` 可验证的包。
- **规划中（M5–M7）**区域：`federation`、`deliver`、`queue`、`web`——**目录尚不存在**。指向它们的连线一律用虚线。
- 状态靠**区域边界**区分，不靠颜色（架构 schema 的 component type 没有"未实现"这一类）。

## 每完成一个里程碑怎么改

1. 把对应节点从"规划中"区域移到"已实现"区域：改 `boundaries[].wraps`，不要只改位置。
2. 若该里程碑引入了新包，补一个组件节点，并在 `cards` 里写清证据（文件 + 行号）。
3. 新模块一旦有真实调用关系，把虚线改成实线（去掉 `variant: "dashed"`）。
4. 跑一次校验：

```bash
node ~/.agents/skills/archify/bin/archify.mjs finalize architecture \
  <本目录>/candidate.json <本目录>/cinderfish-overall.html \
  --quality showcase --json
```

## 已知约束

- **没有做过浏览器视觉校验**：本机无 Chrome/Chromium，`browser-check` 闸门被跳过。schema 校验、交付、provenance 检查均通过。
- **没有仓库溯源链接**：仓库无 `origin` 远程，Archify 的 `--repo-root` 证据校验强制要求 origin 存在。因此候选里不含 `meta.repository`，证据写在卡片文字里。若将来配置了远程，应改用子带 `sources` 的可点击溯源。
- 画布 1310×776，满足 Archify 的宽画布契约（宽高比 ≥ 1.55）。纵向再增加行会跌破可读性下限，需要同时增加横向内容或拆分图。
