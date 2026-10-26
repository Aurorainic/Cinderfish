# M2 实施记录：账户与认证

- 日期：2026-10-26
- 依据：`docs/research/2026-10-26-misskey-source-calibration.md` 的源码契约

## 实现范围

| 端点 | 行为（源码为准） |
|------|------------------|
| `POST /api/signup` | 成功返回 MeDetailed + `token`（原生 token，16 字符）；注册模式来自配置（open/invite/admin）；admin 模式直接拒绝 |
| `POST /api/signin` | 成功返回 `{finished:true, id, i}`；失败使用 **只含 id 的错误信封** `{"error":{"id":"..."}}` |
| `POST /api/username/available` | `{available: bool}` |
| `POST /api/i` | 返回 MeDetailed；仅原生 token 请求附带 secrets（email 等） |
| `POST /api/users/show` | 按 userId 或 username 查询，返回 UserDetailed |
| `POST /api/miauth/gen-token` | 签发 32 字符 MiAuth token，携带 session/name/permission |
| `GET /.well-known/oauth-authorization-server` | RFC 8414 元数据 |
| `GET /oauth/authorize` | 授权码流程，强制 PKCE S256；重定向携带 code/state/iss |
| `POST /oauth/token` | 授权码换 token；授权码一次性；code/client_id/redirect_uri/verifier 全部校验 |

## 与源码对齐的关键细节

1. **原生 token 长度即判据**：`isNativeUserToken` 仅检查长度是否为 16（`misc/token.ts`）。原生 token 存在 `users.token`，MiAuth/OAuth token 存在 `access_tokens`，两套查找路径。
2. **signin 错误信封只有 id**，没有 code/message/kind。这与普通 API 错误的 `{"error":{"message","code","id","kind"}}` 不同，容易踩坑。
3. **secrets 只对原生 token 返回**：`/api/i` 的 `includeSecrets` 取值为 `token == null`，即原生 token（`endpoints/i.ts`）。
4. **用户名校验**：`^\w{1,20}$`（`models/User.ts` 的 `localUsernameSchema`），密码 `minLength: 1`。
5. **bcrypt cost 8**（`bcrypt.genSalt(8)`）。
6. **用户名唯一性大小写不敏感**：源码用 `usernameLower` 字段，本项目在 v2 迁移中增加该字段与唯一索引。
7. **OAuth 授权码一次性**：重复兑换返回 `invalid_grant`；PKCE 只接受 `S256`，`plain` 被拒。
8. **OAuth 错误是扁平结构** `{"error":"invalid_grant","error_description":"..."}`，与 Misskey 的 API 错误信封不同。

## 实现中修正的自身缺陷

- `UsernameAvailable` 语义反转：最初直接返回了"存在性"，导致已占用用户名被报告为可用。由集成测试发现并修正。
- OAuth token 端点的 content type 处理：`r.ParseForm()` 对非表单 content type 会消费 body。改为先判断 content type 再解析。

## 数据模型（迁移 v2）

v2 增量迁移在 v1 基础上追加：

- `users`：新增 `username_lower`（唯一索引）、`token`（唯一索引）、`is_suspended`
- `user_profiles`：新增 `email`
- `registration_tickets`：邀请码，`claim` 语义为一次性占用
- `access_tokens`：新增 `token`、`session`、`name`、`description`、`icon_url`、`last_used_at`
- `signins`：登录历史
- `oauth_grant_codes`：授权码，含 `used` 标记与 `code_challenge`

迁移机制为不可变的版本化脚本：已应用的版本不再改动，新增功能一律追加新版本号。

## 验证

- 单元测试：`internal/auth`（密码哈希、token 长度与唯一性、PKCE S256 向量取自 RFC 7636 附录 B、用户名校验）
- 集成测试：`internal/account`（三种注册模式、邀请码一次性、token 解析）
- HTTP 端到端测试：`internal/httpapi`（signup→signin→i→miauth→OAuth 全链路）
- 真实环境验证：PostgreSQL 18 实例上跑通 v1→v2 增量迁移，并用 curl 验证 7 个账户端点与完整 OAuth 授权码流程（含 code 重用被拒）

测试库通过 `MILNFISH_TEST_DATABASE_URL` 指定，未设置时数据库测试自动跳过。
