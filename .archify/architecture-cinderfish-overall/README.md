# 项目整体结构图（随进度更新）

- 产出：`cinderfish-overall.html`
- 源：`candidate.json`（Archify architecture schema v1）
- 绘制时基线：`60b860c`（M6 时间线与发帖之后）
- 画布：1040×812，宽画布契约不触发（宽度低于 1240）

## 目录命名

目录名**不带时间戳**，固定为 `.archify/architecture-cinderfish-overall/`。历史产物曾用过
`architecture-cinderfish-overall-20261028-195500/`，已废弃：时间戳让同一张图的每次修订都变成
一条新目录，读图时无法判断哪一份是当前版本，且 `candidate.json` 里的 `meta.output` 与各
receipt 的路径字段都要跟着改。复审证据按 `review-<n>/` 递增，这是唯一允许带序号的子目录。

## 这张图表达什么

展示 M1–M5 已实现系统的主链路：客户端与远端实例如何进入单二进制，笔记、账户、Streaming、媒体如何落到 PostgreSQL 与对象存储，以及联邦的入站验签与出站投递队列。

- **矩形边界**：`cinderfish` 单进程内的包。`main` 装配 `httpapi`，`httpapi` 挂接各领域包与联邦、Streaming。
- **边界外**：`PostgreSQL` 与 `media` 后端（local / S3）是进程外依赖；`Misskey 客户端` 与 `Fediverse 远端实例` 是协议对端。
- **虚线**：Streaming 通过 PostgreSQL `LISTEN/NOTIFY` 做跨进程中继，不是进程内直连。
- **web 节点**：Vue 3 前端，经 `httpapi` 的静态托管与 SPA fallback 由同一进程发出；协议命名空间不进 SPA。
- **M7**：打磨项（S3 迁移工具、admin 面板、性能自测、部署文档）尚未开始，状态写在 `cards` 里。

## 每完成一个里程碑怎么改

1. 只改 `candidate.json`：组件、连线、`cards` 与（必要时）边界。
2. 更新每个组件的 `sources` 行号：Archify 会用 `--repo-root` 在固定 revision 上逐条校验。
3. 更新 `meta.repository.revision` 为当前 `git rev-parse HEAD`。
4. 跑校验：

```bash
node ~/.agents/skills/archify/bin/archify.mjs finalize architecture \
  <本目录>/candidate.json <本目录>/cinderfish-overall.html \
  --repo-root <仓库根> --quality showcase \
  --out-dir <本目录>/review-<n> --json
```

已有产物时每次都用新的 `--out-dir`，避免覆盖旧的浏览器证据。

## 已知约束

- **浏览器闸门被跳过**：本机没有 Chrome/Chromium，`browser-check` 状态为 `skipped`。`validate`、`deliver` 与 provenance `check` 均通过，但不是视觉验收。
- **没有做人工视觉检查**：布局按连线关系手工放置，未在浏览器里逐条确认标签避让。
- **私有仓库链接**：`meta.repository` 指向 `https://github.com/Aurorainic/Cinderfish.git`，`sources` 是 GitHub web 链接；未登录访问会 404，这是私有所致。
