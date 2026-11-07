# Miuno 可借鉴实践调研（2026-11-07）

> 目的：对比 `/home/lainsaka/文档/Miuno`（朋友项目，基于 Misskey 源码的完全移植）与本仓库
> `Cinderfish`（Misskey API 契约 + Firefish 前端的技术栈移植），分层列出可借鉴项。
>
> 本文只记录**工程做法与可验证事实**，不复制 Miuno 的源码。Miuno 的 Go 源码与脚本带
> `SPDX-FileCopyrightText: misskey-go contributors` / `AGPL-3.0-only` 头；若未来直接复制
> 某个脚本或文件，必须保留该头并按 AGPL-3.0 处理，本仓库 `AGENTS.md:84` 的署名登记同样适用。
> 建议按本文描述自行实现，而不是搬文件。
>
> 对照基线：`docs/research/2026-11-04-misskey-api-coverage.md`、`docs/HANDOVER.md`、`AGENTS.md`。

---

## 0. 结论先行

1. **「推进很快」的实质不是提交更勤，而是单次吞吐更大。** 本项目提交 471 次、近 7 天 364 次；
   Miuno 只有 220 次、近 7 天约 212 次。但 Miuno 9 天内产出 253,135 行 Go、455 个端点注册，
   本项目内部 Go 53,951 行（生产 31,315 行）、166 个去重 `/api` 路径。
2. 吞吐来自两条我们**没有**的杠杆：
   - **前端成本归零**：Miuno 完整 fork 上游 Misskey Vue 前端（保留 29,110 次提交血缘），只做品牌化；
     本项目是「从零重建 `web/`、逐文件翻译 Firefish 客户端」（`AGENTS.md:47`）。
   - **上游源码当编译器输入，不当参考文档**：`tools/params` 直接解析上游 TS 的 `paramDef`
     生成 455 个端点的参数表，`tools/schemadump` / `make schema` 从上游迁移生成 schema，
     `federationmeta` 从上游 `meta` 生成元数据。本项目是「人读上游 → 手写 Go → 人工维护台账」。
3. **最该借的是「把契约变成可执行门禁」这一整套机器**，不是它的产品范围或代码规模。
   优先级最高的三件事：CI 跑 `go vet`/`go test`、API 覆盖率台账自动化、上游 commit 单点钉死 + 参数表生成。
4. **我们明显领先的一块别自废武功**：测试。本项目 144 个 `_test.go` / 455 个用例（生产代码 31k 行，
   测试 22k 行）；Miuno 41 个测试文件 / 117 个用例，其 ratchet floor 只有 `packages: 5 / files: 14`。
   差距在**门禁**（我们 CI 一行测试都不跑），不在测试量。
5. 需要拍板才能动的是**文档与协作工作流**（63 份 wave 契约、7,681 行 HANDOVER），
   因为 `AGENTS.md:97` 明确要求「提交前更新 HANDOVER」。见 §7。

---

## 1. 两个项目的根本差异

| 维度 | Miuno | Cinderfish |
|---|---|---|
| 定位 | Misskey 后端的 Go **逐文件转译**，联邦在内；客户端沿用上游 Vue 3 | 小型个人实例；Go 重写后端 + 从零重建 Firefish 结构前端 |
| 上游基线 | `misskey-dev/misskey@develop` `8802b29c1336…`（2026-10-20），写死在 `backend/Makefile:13` | Misskey `c05e844`（后端 API 契约）、Firefish `02c31e0` 等（`AGENTS.md:82`），写在文档里 |
| 前端 | 完整上游 fork + 就地改 372 个文件（divergence budget） | 逐文件翻译，保留 `Ported from <repo>@<commit>:<path>` 头 |
| 后端来源 | 上游 TS 业务逻辑转译，代码内 1,155 处 `Source: packages/backend/...` 指针 | 按契约重新实现，代码里只有 2 个文件提到 `paramDef` |
| 生成物 | 76 个 Go 文件 / 53,657 行（约 21%），全部标 `DO NOT EDIT` | 0 |
| 明确不做 | chat 后来停服（`0028_remove_chat.sql`）、Mastodon 层保留 | Mastodon API 彻底放弃、chat/channels/clips/flash/gallery/pages/antennas/Relay 不做 |

**判断**：Miuno 用「高保真复刻」换速度，代价是 25 万行代码的维护面、26 个迁移、372 个前端分叉文件、
以及上游升级时的合并成本。本项目用「小型实例、首版可用性」换自由，代价是 440 个端点里还有 282 个未实现。
两条路线各自成立，**不建议改路线**。

---

## 2. 数字对照（本次实跑，非估算）

| 指标 | Miuno | Cinderfish | 命令 |
|---|---|---|---|
| 首提交 / 总提交 | 2026-10-29 / **220** | 2026-10-26 / **471** | `git rev-list --count HEAD` |
| 近 7 天提交 | 208（11-01 起） | **364** | `git log --since=… --oneline \| wc -l` |
| 单日峰值 | 11-06 **152**（sqlc 按域迁移） | 11-07 **171** | `git log --format='%ad' --date=short \| sort \| uniq -c` |
| Go 文件 / 行数 | 739 / **253,135**（`backend/`） | 293 / **53,951**（`internal/`，含测试） | `find -name '*.go' \| wc -l` + `wc -l` |
| 其中生成物 | 76 文件 / 53,657 行 | 0 | `grep -rl 'DO NOT EDIT' …` |
| 端点 | **455** 处 `r.Register(api.Spec` / 455 个参数表条目 | 199 行 `HandleFunc`、**166** 个去重 `/api` 路径 + 25 非 API 路由 | `grep -c` |
| API 覆盖率 | 客户端调用的端点全被后端serve（`check-contract.sh`） | 440 中已实现同名 158 + 自有 8，**未实现 282** | coverage 台账 §1 |
| 迁移 | 26 个 `NNNN_*.sql` 文件 | 24 个版本，内嵌在 `internal/db/db.go` 的 `map[int]string` | — |
| 测试 | 41 文件 / 117 用例，ratchet floor `5 包 / 14 文件` | **144 文件 / 455 用例 / 8 benchmark** | `find -name '*_test.go'` |
| CI | `verify.yml` 4 job：`structure` / `gate`（`-race` + PG18）/ `client` / `bundle` | `publish-image.yml` **只推镜像，零 Go 检查** | — |
| 文档 | `docs/README.md` 只做索引（32 行） | HANDOVER **7,681 行 / 725 标题**，wave 契约 63 份 / 3,762 行 | `wc -l` |

---

## 3. 分层可借鉴清单

### L1 上游与契约绑定 —— 最高价值，成本最低

**Miuno 怎么做**

- 上游 commit 只写一处（`backend/Makefile:13` 的 `UPSTREAM_COMMIT`），
  `scripts/check-upstream.sh:33-63` 三处比对：Makefile 的值必须出现在 `AGENTS.md`、
  该 commit 必须是 `legacy/frontend` 的祖先（`git merge-base --is-ancestor`，**纯拷贝树会被拒**）、
  Go 与 JS 两个版本脚本必须在固定 `MIUNO_VERSION` 下回答同一个字符串。
- `backend/tools/params` 用 `tools/internal/tsmeta` 解析上游 TS 的 `paramDef` 与 `misskey-js` 类型，
  生成 `internal/server/api/params_generated.go`（455 个端点，带 `Required` / `Enum` / `MinLength` /
  `MaxLength` / `Alternatives`），运行期由 `CoerceParams()` 统一做 ajv 式文本→类型强转。
  端口自有端点手写进 `tools/params/port.go`，与上游表在**同一张表**里生成，重复定义不可能存在。
- 门禁范式统一为「重跑生成器 + `git diff --exit-code`」：`make contract-check`、
  `make schema-check`、`make sqlc-diff`、`check-schema-final.sh` 全是这个形状（`backend/Makefile`）。
- 客户端类型反向生成：`frontend/scripts/generate-types.sh` 调 `go run ./cmd/miuno api-json`，
  再由 `pnpm generate` 产出 `packages/misskey-js/src/autogen/`。即**客户端类型以后端注册表为真相源**。

**我们现状**

- 上游 commit 写在 `AGENTS.md:82` 的散文里，无脚本校验；`tmp/refcheck/misskey` 本地检出存在
  （`packages/backend/src/server/api/endpoints/` 下 61 个条目），但只供人读。
- 参数校验散在 130 处 `decodeOptionalJSON` + 各 handler 匿名 struct + 领域 service 里；
  `AGENTS.md:59` 要求「实现前读 paramDef」，靠人遵守。
- 覆盖率台账 `docs/research/2026-11-04-misskey-api-coverage.md` **完全人工维护**，§7 明写
  「补完一批端点后：更新第 2 节列表…」；台账自己承认漏列 3 条端点、计数曾漂移（第 99-100、164 行）。
- `docs/research/api-contract/*.md`（8 份 / 7,367 行）是历史快照：例如 `notes-core.md:392-405`
  仍写「出站联邦投递未接线」「text 按字节」「没有任何参数上限」，与现行
  `internal/note/service.go:37-49,204-220` 及台账第 124 行**矛盾**。

**建议（P0，1–2 天）**

1. 新增 `scripts/check-upstream.sh`：把 Misskey / Firefish / Iceshrimp 的 pinned commit
   收进一个 `references.env` 单点，校验 `AGENTS.md` 里的哈希与之一致、且 `tmp/refcheck/*` 检出存在。
   注意 `tmp/` 被 gitignore，CI 上不可用时脚本应明确失败而不是静默跳过（见 L6）。
2. 新增 `scripts/check-api-coverage.sh`：解析 `internal/httpapi/server.go` 的
   `HandleFunc("METHOD /api/…")` 去重路径，递归扫 `tmp/refcheck/misskey/.../endpoints/**/*.ts`
   得到 440 条上游端点名，自动产出「已实现 / 未实现 / 自有别名」三张表，与台账 §1 的数字做
   `diff`。台账从此只写**结论与取舍**，不写可机械推导的清单。
3. 新增 `scripts/check-client-contract.sh`（照 `check-contract.sh` 思路，但**从我们的前端扫**）：
   扫 `web/src` 里 `api/client.ts` shim 的调用点，找出「前端调用但后端没注册」的端点。
   这比「对齐 Misskey 440」更贴近实际可用性——Firefish 客户端调的端点未必全在 Misskey 表里。
   动态调用读不出字面量时，按文件计数落到 allowlist，新增即失败。
4. 把 `docs/research/api-contract/*.md` 页首统一加一行「历史快照，非当前事实基线，以代码 + 台账为准」，
   避免后续 workflow 继续引用过期结论。

### L2 端点注册表 —— 高价值，中成本，建议渐进

**Miuno 怎么做**：`backend/internal/server/api/registry.go:34-106` 的 `api.Spec` 是
路由 / 权限 / 限流 / 缓存 / 文档 / 响应 schema / Handler 的**单一真相源**；
`Register` 对重名直接 panic；`DenialOf()` 把权限判定做成可单测纯函数；
`openapi.Write()` 遍历注册表产出 `/api.json`（OpenAPI 3.1），`check-openapi.sh` 校验
operationId、`$ref` 可解析、每个操作有 2xx。`endpoints/` 按域拆成 46 个 `register_*.go`
（注释说明是为了让 merge conflict 不落在同一个四千行函数里），共 455 处注册。

**我们现状**：199 行手写 `HandleFunc` 全在 `server.go:144-377`；
鉴权 98 处 `authenticate` 调用写在各 handler 内、`requireAdmin` 23 处；
限流只有 4 个固定配额（`internal/httpapi/ratelimit.go`，进程内内存计数器）；
无 OpenAPI、无端点元数据、无全仓 `go:generate`。

**建议（P1，渐进，不要一次性重写 199 行）**

- 引入轻量 `EndpointSpec`（`Name / RequireCredential / RequireAdmin / Limit / ParamSource / Handler`），
  **先只做启动自检**：每个注册的 `/api` 路径必须有 Spec，Spec 声明的权限/限流与 handler 内实际调用一致
  （可用 AST 或运行时 wrapper 检查）。不通过就 panic，这就已经把「漏鉴权」这类问题变成启动失败。
- 之后逐域把 `HandleFunc` 换成 `Register(spec)`，路由与 spec 合流；同步从 `tmp/refcheck/misskey`
  的 `meta` export 生成 `Limit` / `requireCredential` / `requireAdmin` 的**期望值**，
  与我们的 Spec 做 diff —— 这能自动发现「我们少了一个限流」「权限设宽了」这类隐性不兼容。
- 有了 Spec 再输出 `/api.json` 是水到渠成的事，可按需做，不必先做。

### L3 数据层 —— 只借轻量部分，不引 sqlc

**Miuno 怎么做**：`backend/sql/<域>.sql` 64 个手写查询文件（2,456 处 `sqlc.arg` + 1,488 处
`sqlc.narg`），sqlc v1.31.1 编译成 64 个 `*.sql.go`；`sqlc.yaml`、命名映射（139 条 rename + 275 条
金标准回归测试）、`schema/final.sql`（真 PG 跑完 26 个迁移再 `pg_dump`）全部生成；
`internal/db/sqlc/columns.go` 是 874 条 `表.列 → 语义` 的手写登记表，`tools/columncheck`
用 `sqlc parse` 的 AST 校验「登记的列存在于终态 schema」+「每条查询引用的列都已登记」；
`check-generated.sh` 用 `declare -A Generators` 把每个生成物映射到它的输入，
生成物变了而输入没变 → 判定手改，失败；`check-migrations.sh` 以 **HEAD** 定义「已发布」，
`git diff --name-status HEAD -- backend/migrations` 里任何 `M/D/T/R/C` 都失败，新编号必须严格递增。

**我们现状**：迁移是 `internal/db/db.go` 的 `map[int]string`（注释说明为了「部署只有一个二进制」），
当前版本 24，但 `AGENTS.md:98` 仍写 22（已失配）；业务 SQL 是静态前缀 + `conds/args` 成对拼接，
`internal/note/timeline.go:54-76` 的 `queryBuilder` 是 `AGENTS.md:100` 那条坑的落点，7 处同样模式；
8 个窄接口，具体 struct 直接持 `*pgxpool.Pool`；无生成物、无列登记表。

**建议**

- **不引 sqlc（P2，暂缓）**：31k 行生产代码、动态 SQL 只有 7 处且已被 `queryBuilder` 收拢，
  迁移结构还内嵌在 Go 里。全面切换要重写 store 层 + 引入工具链 + 改迁移组织，
  收益主要在「防手改 / 防列漂移」，而这些用下面两条轻量手段就能拿到大半。
- **借「终态 schema 校验」（P1）**：加 `tools/schemadump` 等价物——起一个临时库跑完 v1–v24，
  `pg_dump --schema-only`，与 `docs/` 里一份 `schema-final.sql` 做 diff，CI 里挂 PG18 service 跑。
  这能直接发现「迁移链累加结果与预期不符」，也能顺手修掉 `AGENTS.md` 版本 22 / 代码 24 的失配。
  参考 Miuno 的**安全设计**：`check-schema-final.sh:66-72` 断言工具日志里出现本次要删的库名，
  且库名必须匹配白名单正则，因为工具用 `DROP … WITH (FORCE)`。
- **借「迁移不可变门禁」（P0）**：改成以 HEAD 比对 `internal/db/db.go` 中已发布的版本条目，
  新增版本必须严格大于 HEAD 最大值。我们现在靠约定，脚本化成本很低。
- **借「列语义登记表」（P2，可选）**：仅当未来要做 schema 重构/数据迁移时再引入。

### L4 联邦 —— manifest + 可执行不变量，很适合我们

**Miuno 怎么做**：`internal/federationmanifest/manifest.go`（455 行）自称
"authoritative federation manifest"，列出 `FederatedEndpoints`、`FederatedPaths`、
`KeptWellKnownPaths`、被删的表/列/队列/服务/密钥，以及恢复清单；
`manifest_test.go:43-72` 的 `TestTheManifestIsServed` 用注册表 + `recordingMux` 断言
**清单里每个端点和每条路径都真的被服务**。`tools/federationmeta` 从上游 `meta` 生成
`EndpointMeta`，让联邦端点沿用同样的权限/限流/拒绝规则。
代码里 1,155 处 `Source: packages/backend/...` 指针，把「这段逻辑转译自哪里」写在函数旁边；
有意偏离写成「The upstream backend … The port …」+ `Source:` 成对注释
（如 `core/activitypub/ap_inbox.go:1058-1065`）。

**我们现状**：`internal/federation` 21 文件 / 5,829 行，入站验签、三模式策略、
14 类入站活动、`deliver_queue` 十二次退避都已实现；但联邦路径与 API 端点的对应关系散在
`server.go` 的 25 行非 API 路由 + `activitypub.go` 里，「有哪些联邦必备端点」只写在台账 §6 的文字里。
代码内**零** `TODO`（`grep` 实跑），偏差全写进文档，于是文档过期时代码里没有线索。

**建议（P1）**

1. 建 `internal/federation/manifest.go`：列出联邦专用 API 端点、Inbox/Actor/WebFinger/NodeInfo
   等路径、well-known 保留项；配一个 `manifest_test.go`，用 `httptest` 或路由表断言每一条都被注册。
   这与 Miuno 的 `TestTheManifestIsServed` 同形，成本约半天，能防住「改路由改漏一条联邦路径」。
2. **推广 `Source:` / `Contract:` 注释约定（P0，零成本）**：所有新增/修改的兼容逻辑，
   在函数上方写一行 `// Contract: tmp/refcheck/misskey@c05e844:packages/backend/src/server/api/endpoints/notes/create.ts`
   或 `// Source: …`。偏差写成「上游如此 / 本实现如此 / 原因」三行。
   这样偏差跟着代码走，台账和 `api-contract/*.md` 退回为「汇总视图」而不是唯一事实源。
   顺带把 `check-port.mjs` 已有的 `Ported from` 头检查思路扩到 Go 侧（新文件必须有 SPDX + Contract 行）。

### L5 工程化门禁与 CI —— 最明显的缺口

**Miuno 怎么做**：`scripts/dev.sh:34-75` 一条命令按「最便宜的先失败」串起 12 道检查，
最后 `go build / vet / test`；`--db` 时 `: "${MIUNO_TEST_DATABASE:?}"` —— 文件头注释
明写「a skipped test must never be reported as a passing gate」（**缺变量是失败，不是跳过**）。
`check-test-ratchet.sh` 用只升不降的 floor（`backend/test-package-budget.txt`），
降 floor 必须与删除测试**同一提交**，让树里留下「有人做过决定」的记录。
同类预算还有 `divergence-budget.txt`(372)、`design-budget.txt`(radius 0 / colour 816)、
`bundle-budget.txt`(4 个 KiB 上限)。
`.github/workflows/verify.yml` 4 个 job，runner 钉 `ubuntu-24.04`、PG18 service、
`go test -race ./...` + `-tags dbtest`；**刻意不跑的检查写在文件头**
（`vue-tsc` 因 fork 继承 18 个错误、`vitest` 因无测试文件），并写明恢复条件。

**我们现状**：唯一 workflow `publish-image.yml` 只做 GHCR 镜像构建推送，
`go test|go vet|gofmt|coverage` 命中 **0**；`web/scripts/check-port.mjs`（120 行移植规则）
是唯一自动化规则检查且未接 CI；DB/Redis 测试靠环境变量，未设置即跳过
（`internal/testdb/testdb.go:21-41`）；门禁全部是人工流程（`AGENTS.md:97`、台账 §7、
HANDOVER §11.15.7 的一次性实测记录）。

**建议（P0，最高优先，半天）**

1. 新建 `.github/workflows/verify.yml`：`gofmt -l internal/ cmd/` → `go vet ./...` →
   `go test -race ./...` →（可选 PG18 service + `CINDERFISH_TEST_DATABASE_URL`）→
   `node web/scripts/check-port.mjs`。把现有 455 个用例真正变成回归网。
2. 新增 `scripts/dev.sh` 作为**唯一入口**，串起上面的门禁 + L1 的三个 check 脚本；
   `AGENTS.md:97` 的「提交前自行验证」改为「跑 dev.sh 全绿」。
3. **ratchet 要设在我们自己的水位**：`packages: N / files: 144 / tests: 455`，只升不降。
   照抄 Miuno 的 `5/14` 毫无意义。
4. 把「跳过」显式化：DB/Redis 测试未配置时打印跳过清单；
   `--db` 模式下缺变量直接失败。刻意不跑的检查写进 workflow 文件头注释并给出恢复条件。

### L6 构建、版本、发布 —— 借「单源 + 回读校验」

**Miuno 怎么做**：`scripts/version.sh` 从 `git describe` 计算版本（`feat:` 不抬版本，
`!` 在 0.x 抬 MINOR），Go 与 JS 两份实现由 `check-upstream.sh:57-63` 校验输出一致；
`release.sh` 要求干净树 + HEAD 上有精确 tag，`--allow-dirty` 仅演练且在 manifest 写 `dirty: yes`；
产物是单包 + `manifest.txt` + `SHA256SUMS`，**打包后解包回读比对二进制 SHA256**；
`frontend-assets.sh` 拷完前端产物校验 `vite/manifest.json` 与 `loader/boot.js` 存在，防半截构建白屏；
发布 Skill 入库（`.workbuddy/skills/miuno-deploy/`），只做到产物校验，上传/切链接/重启/回滚必须人工确认。

**我们现状**：Dockerfile 三阶段（pnpm build → Go 交叉编译 → alpine 运行），
GHCR `publish-image.yml` 由 `metadata-action` 从 git ref 派生 tag，`web/dist` 不入库、随镜像交付；
版本只存在于文档与 `web/package.json`（0.1.0），代码里 `internal/federation/client.go:25`
的 `UserAgent = "Cinderfish/0.0"` 与已发布 `v0.1.0` 不符；无 Makefile、无产物校验、无版本单源。

**建议（P1）**

- 版本单源：加 `internal/version` 包，构建时由 `-ldflags -X` 注入 `git describe` 结果，
  `UserAgent`、`/api/meta` 的版本字段、`web/package.json` 都从这一处派生；
  加一个 CI 步骤断言三者一致。这直接修掉 `Cinderfish/0.0` 这类漂移。
- 借「回读校验」：镜像构建后跑一个容器 `curl /api/ping` + 校验 `web/index.html` 存在，
  再推 GHCR；避免「镜像推成功了但前端产物没打进去」。
- 不一定要照搬单 tar 包 + manifest：我们的 Dockerfile 已经把前后端打进同一镜像，
  形态不同，只需补校验。

### L7 文档与协作工作流 —— 有价值，但需用户拍板

**Miuno 怎么做**：`AGENTS.md:40-48` 写死「状态放进提交，结论写进提交正文；换会话不靠聊天记录接续；
没有任务板、认领、交接文件或并发 worktree」；`docs/README.md` 只有 32 行索引，
明写「这一层只留索引，不在这里复制第二份」；2026-11-04 一次性清掉 7 类旧文档，
需要时 `git show <commit>:docs/…` 取回；`docs/plan/` 只放**在做的**设计，做完随实现落地、不长期维护；
`docs/README.md:31` 一句「**没有实测过的步骤不要宣告完成**」；
`docs/plan/database-layer-sqlc.md:396` 把 3 条被实跑推翻的结论**就地改正并保留原文**，
理由是「一条被推翻的结论比一条没有的结论更值得留下」；
视觉文档 Phase 5 如实列出 7 项「未完整验证与已知问题」。

**我们现状**：HANDOVER 7,681 行 / 725 标题（§12A–§12GN 共 194 个波次小节），
63 份 wave 契约 / 3,762 行，同一波次在 wave 契约、HANDOVER §12、coverage §2 三处各记一遍；
`AGENTS.md:97` 明确要求「提交前更新 HANDOVER」。文档量本身已成为维护负担，
且已出现可验证的失配（schema 版本 22 vs 24、`api-contract/notes-core.md` 与代码矛盾、
HANDOVER §9 端点清单滞后）。

**建议（需拍板，见 §7）**：先做不冲突的增量——
① 停止新增 wave 契约（新波次只写提交正文 + 台账）；
② HANDOVER 冻结追加，头部维护一张「当前状态 / schema 版本 / 端点数 / 测试数」表，
   数字由 L1 的脚本生成而不是手写；
③ 提交正文统一加 `Validation:` 段，写清跑了什么、什么没跑、已知残留。
是否删除历史波次章节、是否改 `AGENTS.md:97`，由用户决定。

### L8 前端侧（**不在本仓库执行**，转给接收 workflow）

本仓库按 `AGENTS.md:20-24` 已冻结前端工作，以下只作为转交材料：

- **divergence budget 的思路可借，机制不可直接搬**：Miuno 用
  `frontend/scripts/check-divergence.mjs` 对 6 个保留包做 `git ls-tree` 树比对，
  只数 `added + modified`（`removed` 不计，因为「删掉上游不提供的功能不是下次升级要付的合并」），
  上限 372，写进 `divergence-budget.txt`；`upstream-base.txt` 固定基线。
  本项目是逐文件翻译、没有上游血缘树，改成「移植进度/偏差预算」需要另定义基线。
- **可以直接借的三件**：
  ① 品牌覆盖层 + 生成 locale 类型（`frontend/miuno-locales/` 三层 overlay
     键集合必须完全一致，由 `overlay.test.mjs` 校验；`generateLocaleInterface.ts` 把 overlay
     并进去再生成 `autogen/locale.ts`）——本项目已有 `web/scripts/gen-locale.mjs`，可扩；
  ② 把移植规则接进 CI（我们有 `check-port.mjs` 却没跑，Miuno 的 `check-shipping.mjs`
     聚合 lint/SPDX/locale/overlay 并进 CI）；
  ③ SPDX 头 + 资源预算（尺寸、KiB、固定 SHA-256、旧品牌残留正则）——
     Miuno 729/739 个 Go 文件带 SPDX 头，`check-miuno-brand-assets.sh` 是完整范例。
- **不能借的**：完整 fork 上游前端。这会直接推翻 `AGENTS.md:47` 的技术栈替换决策，
  也把 Firefish 参考的版权边界弄乱（`docs/research/2026-11-01-firefish-port-source-and-license-register.md`）。

---

## 4. 明确不建议照搬

| 项 | 原因 |
|---|---|
| 完整 fork 上游 Vue 前端 | 与「从零重建 `web/`、逐文件翻译 Firefish」的既定基线冲突，且上游是 Misskey 而非 Firefish |
| Mastodon API 层 | `AGENTS.md:50` 明确放弃，无端点、无保留路由；Miuno 为此维护了 29 条 SQL |
| chat / channels / clips / flash / gallery / pages / antennas | 台账 §5 Tier 2 明确不做；Miuno 做了 chat 又用 `0028_remove_chat.sql` 停服，是反面教材 |
| 逐文件转译 25 万行 Go | 与「个人/极小实例、首版可用性」定位不符；1155 处 Source 指针、26 个迁移的维护面不适合我们 |
| 把生成物拉到 21% | 代码规模不值得，且我们的迁移内嵌单二进制部署形态不同 |
| 单人「无任务板 / 无 HANDOVER / 无并发 worktree」 | 我们是多 workflow 并行（前端已移交），状态需要跨会话落地，直接删文档会丢上下文 |
| sqlc 全量改造（现阶段） | 见 L3，收益/成本不划算；先借终态 schema 校验与迁移门禁 |
| 照抄 `test ratchet` 数值 `5/14` | 我们已有 144/455，应设自己的水位 |

---

## 5. 优先级路线

**P0（半天到 2 天，纯增量、零决策成本）**

1. `.github/workflows/verify.yml`：`gofmt` + `go vet` + `go test -race` + `check-port.mjs`。
2. `scripts/dev.sh` 唯一门禁入口；`--db` 缺变量即失败；跳过项显式打印。
3. `scripts/check-migrations.sh`：以 HEAD 校验 `internal/db/db.go` 版本不可变、编号递增；
   顺手把 `AGENTS.md:98` 的 22 改成 24（或让脚本生成该数字）。
4. `scripts/check-upstream.sh` + `references.env`：pinned commit 单点，校验 `AGENTS.md`。
5. 代码内 `Contract:` / `Source:` 注释约定 + 联邦偏差三行式；`api-contract/*.md` 加历史快照声明。

**P1（约 1 周）**

6. `scripts/check-api-coverage.sh` + `scripts/check-client-contract.sh`（L1），台账降级为结论视图。
7. `EndpointSpec` 轻量注册表 + 启动自检，逐域迁移（L2）。
8. 联邦 manifest + `TestTheManifestIsServed` 等价测试（L4）。
9. 终态 schema 真 PG 校验 + CI 挂 PG18 service（L3）。
10. 版本单源 + `UserAgent` / `meta` / `package.json` 一致性断言 + 镜像回读校验（L6）。
11. test ratchet 设在我们自己的 144 文件 / 455 用例水位（L5）。

**P2（需决策或成本高）**

12. 文档工作流收敛（§7）、wave 契约停发、HANDOVER 冻结（L7）。
13. 从上游 `paramDef` 生成参数校验表并接入 handler（L1 的完整形态）。
14. sqlc / 列语义登记表（L3）。

---

## 6. 一个可以立刻做的闭环（推荐先做这三个）

```
tmp/refcheck/misskey (pin c05e844)
        │ 扫描 paramDef + 端点目录
        ▼
scripts/check-api-coverage.sh ──► 生成「已实现 / 未实现 / 自有」三表
        │                                    ▲
        │ 期望值                              │ 实际值
        ▼                                    │
EndpointSpec 注册表 ◄── server.go 路由 ──────┘
        │
        ▼
CI: gofmt / go vet / go test -race / check-port.mjs / 上述 diff
```

这三个点闭合之后，「补了哪些端点」「权限/限流是否与上游一致」「有没有跑测试」
都不再依赖人写文档，正是 Miuno 速度优势里我们唯一能低成本复制的部分。

---

## 7. 需要用户拍板的决策

1. **是否收敛文档工作流**：`AGENTS.md:97` 现在要求「提交前更新 HANDOVER」。
   若采纳 L7（状态进提交正文、HANDOVER 冻结、停发 wave 契约），需要修改这条规则和
   `docs/research/*wave*-contract.md` 的处理方式（保留归档 / 移到 `docs/archive/` / 从 git 历史取回）。
   不改规则的话，建议只做 §5 的 P0 增量，L7 保持现状。
2. **是否引入声明式 EndpointSpec**：会动到 `server.go` 199 行路由与 98 处 `authenticate` 调用，
   属于架构调整；按 `AGENTS.md:91`，改模块边界前应先读
   `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md`。
3. **是否以「实际前端调用覆盖率」取代「Misskey 440 端点对齐度」作为主进度指标**：
   两者口径不同，前者更贴近 Firefish 客户端能否跑通，后者更贴近 Misskey 兼容性目标。
   建议**两个都保留但分列**，不要互相替代。
4. **是否把 Miuno 的上游 commit 纳入我们的参考钉死**：我们钉的是 Misskey `c05e844`（2026-10-29），
   对方钉 `8802b29c…`（2026-10-20）。版本不同，参数表不可直接互抄；若要对齐需要先做差异评估。

---

## 8. 证据索引

**Miuno（本地只读检出，根 `/home/lainsaka/文档/Miuno`）**

- `AGENTS.md:7-14, 40-48, 91-121` — 定位、上游 pin、单人工作流、代码约定、提交规范
- `backend/Makefile:11-28` — `UPSTREAM_ROOT` / `UPSTREAM_COMMIT` / `SQLC_VERSION`
- `backend/internal/server/api/registry.go:34-106` — `api.Spec` 字段
- `backend/internal/server/api/params_generated.go:1-25` — 生成参数表结构（455 条）
- `backend/internal/federationmanifest/manifest_test.go:43-72` — `TestTheManifestIsServed`
- `scripts/dev.sh:34-75`、`scripts/check-contract.sh:49-135`、
  `scripts/check-upstream.sh:33-66`、`scripts/check-test-ratchet.sh:33-63`、
  `scripts/check-generated.sh:38-72`
- `backend/test-package-budget.txt`、`frontend/scripts/divergence-budget.txt`、
  `frontend/scripts/upstream-base.txt`、`frontend/scripts/design-budget.txt`、
  `frontend/scripts/bundle-budget.txt` — 阈值文件
- `.github/workflows/verify.yml:12-20, 90-120` — 刻意不跑的检查、gate job
- `docs/README.md:1-31`、`docs/plan/database-layer-sqlc.md:15-27, 375-396`、
  `docs/design/miuno-visual-identity-1.0.md:203-211`

**Cinderfish（本仓库）**

- `AGENTS.md:20-24, 47, 50, 59, 82, 91-100` — 范围、基线、研究规则、工程约定
- `internal/httpapi/server.go:144-377`、`internal/httpapi/ratelimit.go:13-34`、
  `internal/httpapi/ping.go:23-49`、`internal/note/service.go:37-49, 184-223`
- `internal/db/db.go:17-624`（迁移内嵌）、`internal/note/timeline.go:54-76`（queryBuilder）
- `internal/federation/signature.go`、`deliver.go`、`inbox.go`
- `docs/research/2026-11-04-misskey-api-coverage.md:13-24, 99-100, 164, 287-292`
- `docs/HANDOVER.md`（7,681 行）、`docs/research/*wave*-contract.md`（63 份）、
  `docs/research/api-contract/notes-core.md:392-405`
- `.github/workflows/publish-image.yml`、`web/scripts/check-port.mjs`

**复现命令**

```bash
# 提交与规模
(cd /home/lainsaka/文档/Miuno && git rev-list --count HEAD && find backend -name '*.go' | wc -l && find backend -name '*.go' -exec cat {} + | wc -l && grep -rho 'r\.Register(api\.Spec' backend/internal/server/endpoints | wc -l)
(cd /home/lainsaka/文档/Cinderfish && git rev-list --count HEAD && find internal -name '*_test.go' | wc -l && grep -c 's\.mux\.HandleFunc' internal/httpapi/server.go)

# 门禁
(cd /home/lainsaka/文档/Miuno && bash scripts/dev.sh && bash scripts/check-contract.sh && bash scripts/check-test-ratchet.sh)
```
