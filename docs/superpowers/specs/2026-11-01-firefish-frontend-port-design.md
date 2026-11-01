# Cinderfish Firefish 前端整体移植设计（2026-11-01）

状态：待用户批准。批准前不写产品代码、不安装依赖、不替换现有前端入口。

## 1. 意图与原因

Cinderfish 的目标不是继续把当前界面逐步修饰成“像 Firefish”，而是将 Firefish 的前端产品结构、壳、Deck、信息密度、主题语义和交互模型作为主基线，改写为适配 Cinderfish 的 Vue 3 + TypeScript + Vite 前端，并接入现有 Go 服务。

后端继续以 Misskey 原生 API 兼容为契约，ActivityPub 负责联邦通信。Firefish 前端只作为前端体验与组件结构来源，不作为后端实现来源。

## 2. 目标

1. 建立可运行的 Firefish 前端移植基线，而不是继续在当前“AI 味过重的 Mastodon”界面上局部换肤。
2. 保留 Firefish/Iceshrimp 风格的应用壳、宽屏侧边系统、移动端 Deck/单列自适应、时间线密度、Note 交互、主题系统和设置体验。
3. 将 Firefish 的 API、认证、流式、媒体、MFM 和数据类型依赖替换为 Cinderfish 的 Misskey API client。
4. 保持现有 Go 单二进制部署方式：后端托管前端静态产物，开发时允许 Vite 独立运行。
5. 让当前已有的本地账户、时间线、通知、发帖、反应、媒体、设置和 ActivityPub 基础能力在新前端中继续可用。
6. 为后续外站用户关注本站用户的联邦能力保留清晰的前后端边界。

## 3. 非目标

1. 不将 Misskey 前端作为视觉、布局或组件参考。
2. 不提供 Mastodon API 兼容层。
3. 不在本轮把 Firefish 后端整体搬到 Go，也不引入 Node 服务端作为生产依赖。
4. 不承诺一次性达到 Firefish 的全部功能覆盖；Firefish 中依赖其后端专属端点的功能必须标记为未接入、降级或移除。
5. 不为了保持 Firefish 原始代码兼容而保留与 Cinderfish 不匹配的旧数据层、构建链或大型运行时依赖。
6. 不在前端移植任务中顺手重写 ActivityPub；外站关注链路作为单独的后端里程碑。

## 4. 反目标

以下情况视为失败，即使页面能够启动：

- 只复制颜色和圆角，仍保留当前页面的卡片流、稀疏布局和割裂的交互密度。
- 前端同时维护一套 Firefish API 和一套 Cinderfish API，导致页面功能分裂。
- 引入 Firefish 原有不可维护或已停止维护的构建链，破坏 Go 静态部署和现有开发体验。
- 新前端无法在现有 Misskey API 下完成登录、时间线读取、发帖、反应、通知和登出。
- 移动端出现横向溢出、Deck 挤压或关键操作不可达。
- 为了视觉复制而绕过 API 权限、笔记可见性或联邦策略。

## 5. 约束

- 生产后端：Go 1.26 单二进制。
- 前端目标技术栈：Vue 3 + TypeScript + Vite；优先复用现有依赖，不默认新增 UI 框架或状态框架。
- 现有后端 API：Misskey 原生 RPC 风格，Streaming over WebSocket，ActivityPub 端点继续由 Go 提供。
- 主题：保留现有友善的主题系统和自动跟随系统机制；Firefish 主题 token 需要映射到 Cinderfish token，而不是把主题能力删掉。
- 参考源码：Firefish 是主要前端参考；Iceshrimp 用于补充布局和交互证据；Misskey 只用于 API 契约核对。
- 参考源码放在 `tmp/refcheck/`，不进入 Git；源码许可证、NOTICE 和 AGPL 义务必须在移植决策中单独登记。
- 现有 worktree 为 `.worktrees/frontend-ui-redesign`，所有变更在该 worktree 进行，不直接改 main，不推送远端。
- 数据库 schema 不因前端移植而修改；需要新 API 或联邦数据时另开后端任务。

## 6. 方案取舍

### 方案 A：保留当前前端渐进改造

保留现有路由、组件和 stores，逐页改成 Firefish 风格。风险低、可以持续交付，但旧的组件边界、页面信息架构和交互模型会持续影响结果，难以得到真正的 Firefish 前端体验。本次不采用。

### 方案 B：以 Firefish 前端为整体移植基线

先取得可运行的 Firefish 前端，冻结页面、壳、组件和数据流映射，再将 API、认证、流式、媒体、MFM 和类型依赖替换为 Cinderfish 实现。可以删除当前前端的旧结构，结果最接近目标；代价是迁移量大，需要阶段性保留回滚点和明确的兼容矩阵。本次采用。

### 方案 C：直接运行 Firefish 前端并给 Go 后端加 Firefish 专属兼容层

短期视觉相似度最高，但会把 Firefish 后端契约、端点命名和数据语义引入 Cinderfish，形成第二套后端兼容面，扩大长期维护成本，也违背 Misskey API 是唯一后端兼容边界的决定。本次不采用。

## 7. 目标架构

```text
Firefish 风格 Vue 3/Vite 前端
  ├─ AppShell / universal sidebar / mobile navigation
  ├─ Deck / column registry / page windows
  ├─ views + note/feed/composer/profile/settings components
  ├─ Cinderfish API client
  │    ├─ POST /api/<endpoint> Misskey RPC
  │    ├─ /streaming WebSocket
  │    ├─ /media/*
  │    └─ ActivityPub 页面和远程对象链接
  ├─ theme tokens + prefers-color-scheme
  └─ browser storage for session, theme, deck preferences

Go Cinderfish 单体
  ├─ Misskey-compatible API
  ├─ Streaming
  ├─ media
  ├─ ActivityPub / WebFinger / inbox / outbox
  └─ static web hosting
```

前端与后端只通过明确的 API client、Streaming client 和媒体 URL 交互。页面组件不得直接拼接 RPC 请求；所有端点调用、错误信封、认证 token 和重连策略集中在 client/store 层。

## 8. 迁移范围

### 8.1 前端基础

- 复制 Firefish 的目录和组件分区作为迁移参考，按 Vue 3 + Vite 的现有工程约定落地。
- 重建入口、路由、认证守卫、全局错误处理、加载状态和通知提示。
- 重建 universal 壳、宽屏侧边系统、主列、右侧 widget 区、移动端导航和 Deck 列容器。
- 保留并扩展 `data-theme`、CSS token、浅色/深色和 `prefers-color-scheme` 自动跟随。

### 8.2 数据与协议适配

- 以 Misskey 源码的 `paramDef`、`res` 和 `api.types.ts` 为端点依据，建立 Cinderfish 类型和请求映射。
- 先覆盖登录、当前用户、用户资料、时间线、笔记详情、发帖、回复、转贴、反应、通知、媒体、设置、登出。
- 统一处理 Misskey 错误 envelope、token、游标、分页和 WebSocket 事件。
- 将 Firefish 特有字段映射到现有 Cinderfish 字段；无对应字段时提供明确的空值或能力开关，不伪造数据。
- MFM、emoji、CW、可见性、localOnly、引用和媒体附件以 Misskey/Cinderfish 语义为准。

### 8.3 功能能力矩阵

建立 `docs/research/` 下的能力矩阵，至少列出：

- Firefish 页面/组件
- 所需 API 端点和响应字段
- Cinderfish 当前状态：已支持、需适配、需新增后端、暂不支持
- 降级行为和验收场景

首批必须可验收的用户路径：注册/登录、主页时间线、本地/社交/全局切换、发帖、回复、转贴、反应、通知、个人资料、设置、主题切换、移动端导航、宽屏 Deck。

## 9. ActivityPub 后端边界

“外站用户关注本站用户”不作为前端移植的隐含前提，而作为后端联邦专项。该专项需要补齐并单独测试：

- WebFinger 和远程 Actor 解析
- 入站 `Follow` 的验证、关系持久化和 `Accept`/`Reject`
- `Undo(Follow)` 和关系状态变更
- 本站用户的 followers/following 集合更新
- 外站用户资料缓存、签名验证、重试和联邦策略拦截
- 对外站关注者投递本站公开内容及必要的 `Create/Update/Delete`

前端只消费统一的关系状态和通知数据，不直接理解 ActivityPub 活动格式。

## 10. 分阶段交付

1. **基线与可运行参考**：在 `tmp/refcheck/` 获取 Firefish/Iceshrimp/Misskey；记录版本、许可证和启动方式；确认 Firefish 前端是否能在本机启动。
2. **新前端骨架**：建立 Firefish 风格入口、AppShell、主题、路由、认证、API client 和最小页面；保留旧入口可回滚。
3. **核心社交闭环**：时间线、Note、Composer、通知、Profile、媒体和 Streaming。
4. **Deck 与响应式**：宽视图侧边系统、主列/右栏、移动单列与 Deck 列管理；验证 380/600/840/1440px。
5. **设置和能力收口**：主题自动跟随、账号/隐私设置、错误/空/加载态、无后端能力的降级和清理旧架构。
6. **联邦专项**：独立实现并验证外站 Follow 链路，不与前端迁移提交混合。
7. **回归与决策点**：完成真实浏览器验证、Go/前端门禁和能力矩阵更新；若整体移植成本不可接受，依据每阶段提交回退到最近稳定点，而不是保留双前端长期并行。

## 11. 许可证与源码治理

参考源码只放入 `.gitignore` 覆盖的 `tmp/refcheck/`，不提交构建产物或未审查的第三方源文件。实现前记录 Firefish/Iceshrimp 的许可证和具体复用边界；若复制或改写受 AGPL 约束的代码，Cinderfish 的分发许可证义务必须与项目现有 AGPL 决策一致，并在文档中保留来源记录。

## 12. 验证

- 参考前端能启动，且能够通过配置指向本地 Cinderfish API，或明确记录阻塞原因。
- Cinderfish 前端 `pnpm typecheck`、`pnpm build` 通过。
- 后端 `gofmt -l internal/ cmd/` 为空，`go vet ./...` 和 `go test ./...` 通过；配置数据库时再跑数据库集成测试。
- 浏览器真实验收：登录、时间线、发帖、回复、反应、通知、主题切换、宽屏侧栏/Deck、移动端无横向溢出。
- API 行为验证：请求/响应字段、错误 envelope、游标和 Streaming 事件与 Misskey 契约一致。
- 联邦专项另行验证远程 Follow、Accept、Undo 和公开内容投递。
- `git diff --check` 通过，每个阶段都有独立提交，方便整体回退或选择性保留。

## 13. 留给实现者的决策

- Firefish 当前可用分支/提交以及是否需要以 Iceshrimp Vue fork 替代不可运行的 Firefish 构建链。
- 具体的前端目录重命名和保留/删除策略。
- Firefish API 依赖的逐项替换顺序。
- 首批未支持页面的降级方式：隐藏、只读还是明确提示。
- 是否在迁移期间保留 feature flag 以便浏览器对比旧入口和新入口。
- ActivityPub Follow 专项的数据库扩展是否需要新的不可变 migration。
