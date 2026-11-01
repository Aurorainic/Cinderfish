# Firefish 前端整体移植：参考源码与许可证登记（2026-11-01）

用途：固定本轮「以 Firefish 前端为整体移植基线」用到的全部参考源码版本、来源 URL、许可证、
稀疏范围、可构建性判定与**实际复制边界**，让后续任务不必重新猜测「哪一版、能不能抄、抄了要做什么」。

上游设计文档：`docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md`（§11 要求本登记）。
配套文档：`2026-11-01-firefish-port-capability-matrix.md`（页面到端点）、
`2026-11-01-runtime-adaptation-contract.md`（运行时 shim 契约）。

## 1. 角色边界（不可混用）

| 项目 | 角色 | 可用处 | 不可用处 |
|------|------|--------|----------|
| Misskey | 后端 API 契约的权威来源 | `paramDef`/`res`/`errors`、字段名、枚举、错误信封、Streaming 帧格式 | 界面、组件结构、布局、视觉语言 |
| Firefish（原版） | 前端体验的**主**参考 | 壳、Deck、页面信息架构、Note 部件拆分、主题语义粒度、交互模式 | 后端行为 |
| Iceshrimp Vue fork | 前端体验的**补充**参考 | 组件镜像、紧凑布局、断点取舍、日期分隔列表 | 后端行为 |
| Iceshrimp.NET | 仅布局/交互证据 | 断点与服务端渲染布局的取舍 | 代码复制（EUPL，见 §4.3） |

这条边界是 2026-10-30 用户校正的结果，记录在 `2026-10-29-frontend-redesign-reference-analysis.md`
开头；本轮整体移植不改变它，只把「参考」升级为「移植基线」。

## 2. 固定版本（2026-11-01 复核）

| 项目 | 仓库 URL | 固定版本 | 上游提交日期 | 本地范围（稀疏 checkout 或快照） | 许可证 |
|------|----------|----------|--------------|--------------|--------|
| Firefish（原版） | `https://codeberg.org/firefish/firefish.git` | `02c31e0a2804de48c9d73028df525e197d8df622` | 2024-12-11 | `packages/client/src`、`packages/sw/src` | AGPL-3.0（根 `LICENSE` 为 AGPL v3 全文） |
| Iceshrimp Vue fork | `https://codeberg.org/banaanihillo/iceshrimp.git` | `1fa829ed10f42f1511a033f5c22d01ec9efe5a41`（`main`，2025-07-25） | 2025-07-25 | `packages/`（`client`、`sw`、`backend`、`iceshrimp-sdk`；本地快照无根清单） | AGPL-3.0（上游根 `LICENSE` 为 AGPL v3 全文，2026-11-01 复核） |
| Iceshrimp.NET | `https://codeberg.org/iceshrimp/Iceshrimp.NET.git` | `2c61359de29fe3c9ea6a3b31268ceb2ce0e996a7` | 2026-10-27 | `Iceshrimp.Frontend` | EUPL-1.2（根 `LICENSE`） |
| Misskey | `https://github.com/misskey-dev/misskey.git` | `c05e8443a9d1222a467617ed6019784a1f1a53e7` | 2026-10-29 | `packages/backend/src`、`packages/misskey-js/src`、`packages/frontend/src` | AGPL-3.0-only（`LICENSE` + 源文件 SPDX 头） |

补充事实（本轮实测，非引用）：

- Firefish 校验：`packages/client/src` 下 611 个文件、`pages/` 187 个 `.vue`、`components/` 200 个 `.vue`、
  `components/` 根级 144 个 `.vue`、`pages/settings/` 39 个 `.vue`、`themes/` 30 个 `.json5`、
  `widgets/index.ts` 注册 22 个 `Mkw*` 组件；客户端共引用 **157 个不同的 `os.api` 端点**（另有 8 处
  `apiGet`）。端点统计的三种口径见能力矩阵 §1，能力矩阵按 277 个（含 `api(`/`endpoint =`/`fetch` 形式）对账。
- Iceshrimp Vue fork 校验：`packages/client/src` 下 589 个文件、`pages/` 185 个 `.vue`、
  `components/` 188 个 `.vue`；`packages/` 下同时带 `backend`、`client`、`sw`、`iceshrimp-sdk`，
  全量 2012 个文件。**更正一处旧记录**：`2026-10-31-iceshrimp-architecture-translation.md:16` 把「2012 个文件」
  挂在 `packages/client/src` 上，实测该目录是 589，2012 是 `packages/` 全量；旧笔记不改原文，以本文为准。
- Iceshrimp fork 的 `packages/iceshrimp-sdk/LICENSE` 是 **MIT**（Misskey 的 `misskey-js` 系，`syuilo` 版权），
  客户端源码则是 AGPL-3.0（见下一条）；复制 SDK 型文件与复制客户端组件要分开登记。
- `tmp/refcheck/iceshrimp-vue` **不是 git 仓库**：目录下只有 `packages/`，没有 `.git`、没有根
  `package.json`/`yarn.lock`/`LICENSE`/`COPYING`。在该目录里跑 `git remote -v` 会沿父目录解析到本项目仓库，
  输出 `Aurorainic/Cinderfish`——这是最容易误判出处的坑。版本号只能取自 Codeberg API
  （`1fa829ed10f4`，2026-11-01 复核），见 `2026-10-31-iceshrimp-architecture-translation.md:11-21`。
- Iceshrimp fork 的许可证文本同样**只能从上游取证**：`https://codeberg.org/banaanihillo/iceshrimp/raw/branch/main/LICENSE`
  于 2026-11-01 取回 34520 字节 AGPL v3 全文，`…/raw/branch/main/COPYING` 声明整体 AGPL-3.0 并列出
  `.config/`、`custom/assets/` 的 Apache-2.0 例外。复制前必须按 SHA 回上游核对，本地没有可核对的文本。
- Firefish 校验的 client 引用基线来自 `grep -rhoE "os\.api\(['\"][a-zA-Z0-9/_.-]+" packages/client/src`。

## 3. 可构建性判定（只读参考）

设计文档 §10 第 1 阶段要求「确认 Firefish 前端是否能在本机启动」。结论：**不能，两个参考前端都只作为只读参考**，
不作为本地生产依赖，也不引入它们的构建链。

### 3.1 Firefish 原版：不可构建

| 证据 | 命令/位置 | 结果 |
|------|-----------|------|
| 只稀疏取了两个源码范围 | `git -C tmp/refcheck/firefish sparse-checkout list` | `packages/client/src`、`packages/sw/src` |
| 构建入口文件缺失 | `ls packages/client/index.html` | 不存在 |
| 根构建脚本缺失 | `ls scripts/` | 不存在（`package.json:12` 的 `build` 依赖 `scripts/copy-index.mjs`） |
| workspace 依赖不可解析 | `pnpm-workspace.yaml:6` 声明 `packages/firefish-js`，`ls packages/firefish-js` | 不存在；而 `packages/client/package.json` 依赖 `firefish-js: workspace:*` |
| 工作区声明多于实际目录 | `cat pnpm-workspace.yaml` 声明 5 个包，`ls packages/` | 只有 `README.md`、`client`、`sw`；缺 `backend`、`backend-rs`、`firefish-js` |

`workspace:*` 指向一个不存在于工作区的包时，`pnpm install` 直接以
`ERR_PNPM_WORKSPACE_PKG_NOT_FOUND` 中止，因此在当前克隆上不可能产出构建产物。（本轮**没有**执行
`pnpm install`：缺少构建入口文件已经足以判定失败，跑安装只会往参考目录里堆 `node_modules`。）
本机工具链为 Node v24.22.1 / pnpm 9.9.0，与 Firefish 声明的 `packageManager: pnpm@9.9.0` 一致；
缺的不是工具链，而是源码范围。

复现三条（不必装依赖）：

```bash
cd tmp/refcheck/firefish
cat pnpm-workspace.yaml          # 声明 5 个包
ls packages                      # 只有 README.md client sw
grep -n 'firefish-js' packages/client/package.json   # "firefish-js": "workspace:*"
```

### 3.2 Iceshrimp Vue fork：不可构建（本地快照），且不作为构建基线

本地快照只有 `packages/`，没有根 `package.json`、`yarn.lock`、`.pnp.cjs`；上游用 `yarn@4.3.1` 且
`nodeLinker: pnp`，不是 pnpm 工作区。即便补上根清单，它同样是 Firefish 系构建链（Vite + 自有 `nirax`
路由 + 自有 `os` 总线 + 自身后端契约），会把 Firefish 后端契约、Service Worker、插件运行时一并拖进来，
与 `AGENTS.md`「Misskey API 是唯一后端兼容边界」冲突。它的用途固定为**组件与布局镜像**
（见 `2026-10-31-iceshrimp-architecture-translation.md`）。

### 3.3 Misskey：只读契约

`packages/frontend/src` 仍保留在稀疏范围里，但按 §1 的角色边界，它**只用于在必要时确认
`misskey-js` 类型与端点行为**，不作为任何 UI 证据。

补充：Misskey 的 pin 是 `c05e844`，不是上游 HEAD（2026-11-01 上游 `develop` 已到 `3a43e05f`，落后两个提交）。
作为契约 pin 合法；若后续任务要按上游最新行为核对字段，先确认这两个提交没有改动目标端点（本轮未逐一比对 diff）。

## 4. 许可证与复制边界

### 4.1 结论

1. Cinderfish 沿用项目的 AGPL-3.0 决定（`AGENTS.md`「Research Rules」）。**若复制或改写 Firefish /
   Iceshrimp Vue fork 的源码，Cinderfish 分发时必须整体按 AGPL-3.0 提供源码**；不得把这些代码放进
   任何非 AGPL 的产物里。
2. 参考源码只放在 `tmp/refcheck/`（`tmp/` 已被 `.gitignore` 覆盖）。**不把未审查的第三方源文件、
   构建产物、`node_modules` 提交进仓库。**
3. 复制必须以「文件级」为单位登记来源：移植文件顶部保留原文件的 SPDX/版权头，并在
   `docs/HANDOVER.md` 的移植记录里写明「来自 `<repo>@<commit>:<path>`」。
4. 本轮的移植方式是「结构移植 + 重写实现」，不是整目录拷贝：Vue 单文件组件的 `<template>`
   可以按原样迁移（AGPL 允许），`<script setup>` 中的 API、store、路由、i18n 调用一律重写到
   Cinderfish 的对应物（见运行时契约文档）。这样每处复制都能在 review 时逐条对回来源。

### 4.2 AGPL-3.0 的具体义务（Firefish、Iceshrimp Vue fork、Misskey）

| 允许 | 需要 | 禁止 |
|------|------|------|
| 阅读、改写、复制组件模板与样式、复用字段名/枚举/行为 | 分发时提供完整对应源码（AGPL-3.0）、保留版权与许可证声明、在文档中登记来源 commit | 把 AGPL 代码并入闭源产物；删掉版权头；只发布构建产物不给源码 |
| 把主题 JSON5 的**语义**（token 名、派生规则）翻译成本项目 CSS 变量 | 主题文件若被整体复制，同样受 AGPL 约束并需登记 | 把主题文件当「配置数据」绕过许可证 |

**按子包区分的例外（容易踩）**：AGPL 只覆盖各仓库的默认范围，`COPYING` 里有明确的逐目录例外：

| 仓库 | 例外 | 许可证 | 对本轮的影响 |
|------|------|--------|--------------|
| Firefish | `packages/firefish-js` | **MIT** | 抄它的类型定义（`Endpoints`、`entities`）不受 AGPL 传染。这正是最容易被顺手复制的东西——登记时按 MIT 记，不要当 AGPL |
| Firefish | `packages/macro-rs` | Apache-2.0 或 MIT（二选一） | 本轮不涉及（Rust 宏包不在稀疏范围） |
| Firefish | `.config/`、`custom/` | **Apache-2.0** | 只可能在复制配置模板时命中，本轮不复制 |
| Iceshrimp Vue fork | `packages/iceshrimp-sdk` | **MIT**（`syuilo` 版权） | 同上，SDK 型文件按 MIT 记 |
| Iceshrimp fork | `.config/`、`custom/assets/` | Apache-2.0 | 同上 |
| Misskey | `COPYING` 记录 RsaSignature2017、nsfwjs | 均 MIT | 本轮不涉及 |

结论：**「Firefish 是 AGPL」不能推成「Firefish 仓库里每个文件都是 AGPL」**。复制前先看文件所属子目录，
在移植记录里同时写子包与许可证，否则会把 MIT 代码错标成 AGPL，或反过来把 AGPL 当 MIT 用。

### 4.3 Iceshrimp.NET（EUPL-1.2）

EUPL-1.2 与 AGPL-3.0 兼容，依据是 `tmp/refcheck/iceshrimp/LICENSE:266-275` 的
「Compatible Licences」附录，其中明确列出 **GNU Affero General Public License (AGPL) v. 3**：
被覆盖作品可以并入 AGPL-3.0 分发，冲突时以兼容许可的义务为准。

但本轮**不复制代码**：它只用于布局/断点/交互取舍的证据来源，引用形式为「`Iceshrimp.Frontend/...:行号` 描述了 X」。
若将来确实要复制，需要单独立项评估许可证兼容与声明，不在本轮范围。另外本机没有 `dotnet`，
它同样不可构建（`which dotnet` 为空）。

## 5. 复现命令

```bash
export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823   # 仅 GitHub 需要

# Misskey —— API 契约
git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git tmp/refcheck/misskey
cd tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src packages/frontend/src

# Firefish —— 前端主参考（Codeberg 直连，偶发 504 时重试或改用固定 commit 的归档）
cd ../.. && git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/firefish/firefish.git tmp/refcheck/firefish
cd tmp/refcheck/firefish && git sparse-checkout set packages/client/src packages/sw/src
cd ../..

# Iceshrimp Vue fork —— 组件镜像参考。上游用 yarn 4.3.1 + PnP，本地快照没有 .git，
# 也不是 sparse checkout：它保留的是完整的 packages/（2012 个文件，无根清单），只解归档里的 packages/：
curl -L -o /tmp/iceshrimp-vue.tar.gz \
  https://codeberg.org/banaanihillo/iceshrimp/archive/1fa829ed10f42f1511a033f5c22d01ec9efe5a41.tar.gz
mkdir -p tmp/refcheck/iceshrimp-vue
tar -xzf /tmp/iceshrimp-vue.tar.gz -C tmp/refcheck/iceshrimp-vue --strip-components=1 --wildcards '*/packages/*'
# 本地快照只有 packages/，没有根 package.json/yarn.lock/LICENSE；比对时只比 packages/client 与 packages/sw
cd tmp/refcheck/iceshrimp-vue && ls packages   # README.md backend client iceshrimp-sdk meta.json sw

# Iceshrimp.NET —— 仅布局/交互证据
cd ../.. && git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/iceshrimp/Iceshrimp.NET.git tmp/refcheck/iceshrimp
cd tmp/refcheck/iceshrimp && git sparse-checkout set Iceshrimp.Frontend
```

版本核对（本轮实际执行过的）：

```bash
cd ../..   # 回到仓库根目录（上一段结束时在 tmp/refcheck/iceshrimp）
git -C tmp/refcheck/firefish  log -1 --format='%H %cI'   # 02c31e0a… 2024-12-11
git -C tmp/refcheck/misskey   log -1 --format='%H %cI'   # c05e8443… 2026-10-29（上游 develop 已到 3a43e05f）
git -C tmp/refcheck/iceshrimp log -1 --format='%H %cI'   # 2c61359d… 2026-10-27
# iceshrimp-vue 没有 .git，只能查 API（2026-11-01 实测 1fa829ed10f42f1511a033f5c22d01ec9efe5a41）
curl -s 'https://codeberg.org/api/v1/repos/banaanihillo/iceshrimp/commits?sha=main&limit=1' | head -c 300
# 许可证文本（本地没有）：
curl -s 'https://codeberg.org/banaanihillo/iceshrimp/raw/branch/main/LICENSE' | head -3   # AGPL v3
```

## 6. 本轮固定下来的决策

| 决策 | 依据 |
|------|------|
| Firefish 原版固定在 `02c31e0`，只读 | §3.1：稀疏范围缺构建入口与 workspace 包 |
| Iceshrimp Vue fork 固定在 `1fa829ed10f4`，只读 | §3.2：构建链与后端契约不引入 |
| 不把任何参考源码加入 `package.json` / workspace / 构建脚本 | 设计文档 §3.3、§4 反目标 |
| 移植按文件登记来源，模板可迁移、脚本必须重写 | §4.1 第 3、4 条 |
| 主题 token 采用「语义翻译」，不引入 JSON5 DSL | 设计文档 §5；JSON5 主题编辑器在能力矩阵里为「隐藏」 |
| 复制登记按子包区分许可证（`firefish-js`/`iceshrimp-sdk` 为 MIT） | §4.2 的例外表 |
| `iceshrimp-vue` 的版本与许可证只能从 Codeberg 取证 | §2：本地无 `.git`、无 `LICENSE` |
| Misskey 前端不进任何 UI 决策 | §1 角色边界 |
