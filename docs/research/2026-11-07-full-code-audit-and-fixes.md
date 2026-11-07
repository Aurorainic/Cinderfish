# 2026-11-07 全面代码审计与阶段性修复

本次对全部 Go 后端代码(约 3.2 万行,298 个文件)做了一次全面审计,并按严重性分四个提交完成了阶段性修复。
审计由四路并行进行:签名/认证/OAuth 链路(自审)、httpapi 核心端点、federation 与 account/note 包、streaming/media/config 等支撑模块,
并以 `tmp/refcheck/misskey`(固定版本 `c05e844`)逐项对照契约。

基线:`go vet` 干净、全部测试(含数据库与 Redis)通过;因此所有发现均为 vet/test 抓不到的语义问题。

## 修复的提交

| 提交 | 范围 | 内容 |
|------|------|------|
| `7958cfa` | 阶段1+2(P0/P1 安全) | 入站签名与 actor 绑定、密钥轮换恢复、signin 限流、OAuth redirect 同源 |
| `f2840f8` | 阶段3(支撑模块 P0/P1) | SVG/HTML 存储型 XSS、token 哈希查找、webhook SSRF、WS 读帧上限、SMTP RCPT 注入、媒体删除一致性、迁移丢缩略图 |
| `f9a1684` | 阶段4(federation P0/P1) | Accept/Reject/Undo 关系伪造、Date 强制、Host 真绑定、逐跳策略复查、Update 归属、Announce 可见性、WebFinger host 校验、fileIds 所有权、signin 计时侧信道 |
| `dbbb908` | 阶段5-7(httpapi P0/P1 + P2) | drive/files/show 认证、withRenotes 生效、过期投票拒绝、删除计数回退、出站私网防护、includeTypes 空数组语义、sinceDate 吞错、folders 错误分流、LIKE 转义、multipart 限额、setInt、原子写 |

## P0 修复明细(全部完成)

1. **入站签名与 actor 无绑定(身份伪造)**。`handleInbox` 验证签名后丢弃 `keyID`(`_ = keyID`),inbox 只信任 activity 的 `actor` 字段。
   任何持自有合法密钥的 peer 可伪造任意远端用户投递 Follow/Like/Announce/Create/Delete。
   修复:签名者 URI(keyId 去 fragment)必须等于 `activity.actor`。对照 Misskey `ApInboxService.ts:262/473`
   (`actor.uri !== getApId(activity.actor)` → "invalid actor")。
2. **Accept 关系伪造**。内嵌 Follow 只需指向"某个本地用户"即可把任意本地账号绑定为伪造成方的粉丝(Reject 同理可拆散真实关注)。
   修复:内嵌 Follow 的 object 必须等于签名者 URI;Undo(Follow) 增加内嵌 actor 归属校验。
3. **drive/files/show 无认证(IDOR)**。未登录可读任意用户文件元数据;非属主一律 404(上游 `drive/files/show.ts` 为 requireCredential + ACCESS_DENIED,
   本项目按 NO_SUCH_FILE 404 处理以避免存在性泄露)。
4. **notes/create 附件 IDOR**。`UPDATE drive_files SET id = id WHERE id = $1` 恒真,他人文件可挂上公开笔记。
   修复:按作者过滤(上游 `NoteCreateService.ts` 的 Files 关系解析语义,静默丢弃非本人/不存在的 id)。
5. **SVG/HTML 存储型 XSS**。上传文件按声明 Content-Type 原样回源且无防护头。
   修复:活跃内容类型(image/svg+xml、text/html、xml/xslt、pdf、flash)统一 `Content-Disposition: attachment` + `CSP: sandbox` + nosniff。
6. **access_tokens 明文查找**。写入 `token_hash` 却按明文 `token` 列查询(哈希基建形同虚设)。
   修复:Resolve 按 SHA-256 哈希查找(高熵随机 token 无盐快哈希是安全的查找键;`users.token` 为 Misskey 上游同款明文设计,未动)。
7. **webhook SSRF**。任意注册 URL 每条笔记触发 POST,`latest_status` 形成内网端口扫描 oracle。
   修复:`ValidateURL`(scheme + 字面 IP + DNS 解析检查)在创建、更新、投递三处执行;拒绝映射为 400。

## P1 修复明细(全部完成)

- **密钥轮换死锁**:存储公钥过期后对端所有投递永久 401 且无法自愈(刷新密钥的 Update 也要先过签名)。
  修复:存储钥验证失败时抓取一次 actor 文档刷新后重试(对照 Misskey 验签失败重取行为)。
- **signin 无限制流**:按 Misskey `SigninApiService.ts`(key 'signin',10 次/小时、minInterval 1s)按 IP 限流;
  配置项 `signin_rate_limit`(负数关闭,测试用)。429 带 Retry-After。
- **OAuth 开放重定向/授权码注入**:redirect_uri 不与 client_id 绑定。修复:redirect_uri 必须是绝对 http(s) URL、
  与 client_id 同主机、无 fragment(替代 Misskey OAuth2ProviderService 的客户端发现)。
- **Date 头缺失跳过新鲜度检查**:签名覆盖 date 但请求可不带头,重放窗口无限。修复:Date 必须存在且在 ±5 分钟内。
- **Host 绑定恒真**:`Host: r.Host` 自己和自己比。修复:必须签 `cfg.Domain`(loopback/监听地址豁免,供直连部署与双实例测试)。
- **出站重定向不做策略复查**:allowed host 302 可把 GET/签名 POST 带到被屏蔽主机或内网。
  修复:策略检查移入共享 CheckRedirect,FetchObject/FetchMedia/PostActivity 一并生效;每跳同时过 `CheckURLRoutable`。
- **Update\<Note\> 缺作者归属**:`attributedTo` 为空时跳过比对、UPDATE 只按 ap_id,同 host 兄弟账号可改写他人笔记。
  修复:`UpdateRemoteNote` 增加 `user_id = actor` 条件。
- **入站 Announce 提权可见性**:对本地 followers/specified 笔记的 Announce 以 public 入库进全局时间线。
  修复:`Store.NoteAudience` 检查目标可见性与 local_only。
- **WebFinger 跨域顶替**:self href 不校验 host;acct 的 host 未做字符校验。
  修复:`HostOf(self) == 请求 host`;`SplitAcct` 限定主机名字符。
- **withRenotes=false 完全不生效**:修复后追加上游 timeline.ts 的排除条件(renote_id 为空或自带 text/files/poll,引用仍可见)。
- **可创建已过期投票**:`validatePollParam` 增加"过期时间必须在未来"。
- **删除笔记不回退计数**:replies_count/renote_count 永久虚高,可刷 featured 排名。
  修复:Delete 在同一事务内递减父行两个计数。
- **ap/show 内网盲探(公网模式)**:`Policy.CheckURLRoutable` 覆盖全部出站路径(初始 URL + 每一跳重定向),
  拒绝 loopback/私网/link-local 字面与解析地址;解析失败交由连接失败兜底(兼容双实例 rewrite transport)。
- **SMTP RCPT 命令注入**:信封地址经 `net/mail.ParseAddress` + 控制字符检查。
- **WebSocket 无读帧上限**:每连接 `SetReadLimit(64 KiB)`。
- **媒体迁移丢缩略图**:迁移按行搬 `thumbnail_key`,一并纳入源删除集合;未知 size 以 -1 复制(io.Copy/minio 均支持)。
- **删除文件行/对象一致性**:触发器拦下删除(`RowsAffected==0`)时不再继续删对象。

## P2 已修(择要)

notifications 空数组语义(Misskey `NotificationService.ts:302` length>0 才过滤)、sinceDate 转换吞错改 500、
drive/folders/show 错误分流(仅 ErrNotFound 404,其余 500)与 NO_SUCH_FOLDER 状态码统一 404、
admin emoji 与 hashtag 搜索的 LIKE 通配符转义、multipart 上传整体限额与 form 内 `i` 令牌、
`config.setInt` 改 `strconv.Atoi`、LocalStorage 原子写(.tmp + rename)、signin 哑 bcrypt 计时均衡(保留上游 404/403 区分,属契约)。

## 已核实无问题的高风险面

SQL 注入(全部占位符/白名单 ORDER BY)、MFM→HTML 转义链、签名算法混淆(仅 rsa-sha256)、digest 校验、
`(request-target)` 重构一致性、TOTP 常时比较、streaming 订阅越权(仅本连接 viewer)、投递队列 `for update skip locked`、
迁移机制(版本表 + 单事务 + 顺序应用)、config 默认值(注册 admin、联邦 none、host 127.0.0.1)。

## 未修复、留待后续的项(按优先级)

1. **token 过期机制**:access_tokens 无 `expires_at`,泄露即永久有效(P2,涉及 schema 与客户端刷新)。
2. **通知幂等/重放去重**:同一 activity id 重放可产生重复通知(需要按 activity id 或 (type,notifier,note) 唯一约束)。
3. **投票单选并发 TOCTOU**(checkVoteAllowed 在事务外)。
4. **Bus/中继 goroutine 无界**(streaming `Publish`、`readNotification` 每帧、deliver `NotifyDelivery`)。
5. **Accept 同步阻塞出站**(inbox 请求内同步 POST,慢 peer 拖住处理;缺补偿队列)。
6. **DNS rebinding TOCTOU**(校验与拨号两次解析;根治需 DialContext 校验)。
7. **pgxpool MaxConns 硬编码 10**(进配置)。
8. **notes.file_ids 缺 GIN 索引**(v25 触发器全表扫)、`notifications.type` 缺 CHECK。
9. **webhook 失败无重试队列**;secret 以明文头传输(Firefish 兼容,可另加 HMAC 头)。
10. **可无界 multipart 临时盘占用已部分缓解**(MaxBytesReader 按 ContentLength;分块编码仍可谎报,根治需固定上限)。
11. **users/show 批量 userIds 无上限**(N+1 逐个查询,建议限 100)。
12. **可选认证端点把无效 token 降级为匿名**(方向保守不泄数据,但掩盖 token 过期;上游统一 401)。

## 行为变化说明(对接方可见)

- `/api/drive/files/show` 未登录/非属主从 200 变为 401/404。
- `includeTypes: []` 从返回空变为返回全部(Misskey 语义)。
- `withRenotes: false` 现在真的隐藏纯 renote(引用不受影响)。
- 过期投票在 notes/create 即被拒(CANNOT_CREATE_ALREADY_EXPIRED_POLL)。
- OAuth 授权的 redirect_uri 必须与 client_id 同主机;非法 client_id(非 URL)被拒。
- 指向内网/loopback 的 webhook URL 在创建/更新/投递时被拒(400)。
- 对公网联邦模式的 `ap/show`,字面私网/loopback URI 被拒。

## 验证

每个提交均独立通过:`gofmt -l internal/ cmd/` 空、`go vet ./...` 干净、
带 `CINDERFISH_TEST_DATABASE_URL`/`CINDERFISH_TEST_REDIS_URL` 的 `go test ./... -count=1` 全绿。
新增回归测试:signin 限流×3、OAuth 跨主机拒绝、webhook URL 校验、ap/show 私网拒绝、drive/files/show 属主、
fileIds 他属文件、includeTypes 空数组、rate limiter 配置语义。
