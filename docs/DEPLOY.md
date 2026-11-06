# 部署指南

面向一台自管的 Linux 服务器。目标拓扑：

```
Internet ── 443/TLS ── Caddy（反向代理）
                          │  http://127.0.0.1:3000
                          ▼
                    cinderfish（单进程）
                     ├── PostgreSQL（本机或托管）
                     ├── /var/lib/cinderfish/media（可选）
                     └── S3 兼容存储（可选）
```

Caddy 会自动处理 HTTPS 与 WebSocket 升级；`/streaming` 不需要额外配置。

## 1. 构建

在开发机或服务器上：

```bash
# 后端
go build -trimpath -o cinderfish ./cmd/cinderfish
go build -trimpath -o cinderfish-admin ./cmd/cinderfish-admin
go build -trimpath -o cinderfish-media-migrate ./cmd/cinderfish-media-migrate

# 前端（产物在 web/dist）
cd web && pnpm install --frozen-lockfile && pnpm build && cd ..
```

只需要 API 时可以跳过前端构建；`web_dir` 不存在时服务器自动退回纯 API 模式。

## 2. 目录与用户

```bash
sudo useradd --system --home /opt/cinderfish --shell /usr/sbin/nologin cinderfish
sudo mkdir -p /opt/cinderfish /etc/cinderfish /var/lib/cinderfish/media
sudo cp cinderfish cinderfish-admin cinderfish-media-migrate /opt/cinderfish/
sudo cp -r web/dist /opt/cinderfish/web
sudo chown -R cinderfish:cinderfish /opt/cinderfish /var/lib/cinderfish
sudo chmod 750 /etc/cinderfish
```

## 3. 配置

`/etc/cinderfish/config.toml`（示例见 `deploy/config.production.toml`）：

```toml
host = "127.0.0.1"        # 只监听本机，由 Caddy 对外
port = 3000
domain = "social.example.com"

database_url = "postgres://cinderfish:CHANGE_ME@127.0.0.1:5432/cinderfish"

registration_mode = "admin"   # open | invite | admin
federation_mode = "public"    # public | allowlist | none

web_dir = "/opt/cinderfish/web"

[federation]
# 反向代理后面必须显式给出对外的 https 地址，否则联邦 id 会写成 http://内网:3000
public_url = "https://social.example.com"

[media]
backend = "local"
local_path = "/var/lib/cinderfish/media"

[media.s3]
# backend = "s3" 时填写
endpoint = ""
bucket = ""
region = ""
access_key = ""
secret_key = ""
use_ssl = true
public_base_url = ""
```

> 数据库密码等敏感项建议放进 `/etc/cinderfish/cinderfish.env`（`CINDERFISH_DATABASE_URL=...`），systemd 以 `EnvironmentFile` 读取，配置文件权限保持 `0640`。完整键名以 `config.example.toml` 为准。

## 4. 数据库

```bash
sudo -u postgres createuser --pwprompt cinderfish
sudo -u postgres createdb -O cinderfish cinderfish
```

迁移在每次启动时自动执行，版本只增不改。升级前先备份：

```bash
pg_dump --format=custom --file="/var/backups/cinderfish-$(date +%F).dump" cinderfish
```

媒体目录同样要备份（或依赖对象存储的版本控制）：

```bash
rsync -a --delete /var/lib/cinderfish/media/ /var/backups/cinderfish-media/
```

## 5. 第一个管理员

`registration_mode = "admin"` 时没有自助注册路径，用命令行创建第一个账号：

```bash
sudo -u cinderfish /opt/cinderfish/cinderfish-admin --config /etc/cinderfish/config.toml \
  create --username root --password-stdin --admin
```

密码从标准输入读取，避免进入 shell 历史。之后可以在网页 `/admin` 里创建其他账号、停用账号或切换联邦模式。

## 6. systemd

仓库提供 `deploy/cinderfish.service`：

```bash
sudo cp deploy/cinderfish.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cinderfish
sudo journalctl -u cinderfish -f
```

服务文件使用 `ProtectSystem=strict`，只有 `/var/lib/cinderfish` 可写；如果媒体目录换了位置，同步修改 `ReadWritePaths`。

## 7. 反向代理

`deploy/Caddyfile` 是最小可用配置：

```caddyfile
social.example.com {
  encode zstd gzip
  reverse_proxy 127.0.0.1:3000
}
```

Caddy 自动签发证书并透传 `Host`，WebSocket 也无需额外配置。注意：

- **不要让代理改写 `Host`**：ActivityPub 的入站验签要求 `Host` 是本实例域名。
- 请求体上限至少 32 MB（附件默认上限 20 MB）；Caddy 默认不限制，nginx 需 `client_max_body_size 32m;`。
- 如果代理层终止 TLS，`domain` 仍填对外域名，联邦 id 才会正确。

nginx 参考：

```nginx
server {
  listen 443 ssl http2;
  server_name social.example.com;
  client_max_body_size 32m;

  location / {
    proxy_pass http://127.0.0.1:3000;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 300s;
  }
}
```

## 8. 媒体存储

### 本机

`backend = "local"`，文件落在 `local_path`，通过 `/media/<key>` 提供。需要把该目录纳入备份。

### S3 兼容

`backend = "s3"`，填写 endpoint / bucket / 凭据。两种读取方式：

- `public_base_url` 非空：对象按 `<public_base_url>/<key>` 直接公开读（适合 R2、B2 或带 CDN 的桶）。
- 留空：使用预签名 GET，桶可以保持私有（适合 MinIO、AWS S3）。预签名需要知道桶的 region：
  AWS 会自动查询，非 AWS 服务建议显式填写 `region`，否则生成 URL 前会先做一次区域发现请求。

迁移已有文件：

```bash
# 本地 → S3
sudo -u cinderfish /opt/cinderfish/cinderfish-media-migrate --config /etc/cinderfish/config.toml \
  --to s3 --dry-run
sudo -u cinderfish /opt/cinderfish/cinderfish-media-migrate --config /etc/cinderfish/config.toml \
  --to s3 --delete-source
```

迁移只改 `drive_files.storage_backend` 与对象位置，key（内容哈希）不变，笔记链接无需改写。完成后把 `[media] backend` 切到新后端再重启。`--delete-source` 只在整轮无失败时清理源对象。

补生成缩略图（缩略图功能上线前上传的图片，或当时解码失败的图片）：

```bash
sudo -u cinderfish /opt/cinderfish/cinderfish-media-migrate --config /etc/cinderfish/config.toml \
  --backfill-thumbnails --limit 200
```

它按创建时间扫描 `thumbnail_key IS NULL` 的图片行，读取原图、生成 512 长边预览并写回同一存储后端。
`--limit` 限制单轮数量（0 = 全部），可以重复运行直到 `generated=0`；`failed` 会保留原样以便下次重试，
进程以退出码 1 结束。与 `--to` 互斥。

## 9. 联邦

- 对外只需 443；入站 inbox、WebFinger、NodeInfo 都走同一域名。
- `federation_mode` 初始值只做首次 seed，之后以数据库为准；在 `/admin` 切换立即生效，重启不会覆盖。
- WebFinger 依赖 `domain` 正确，`public_url` 用于生成 actor / note 的 id。
- 白名单按域名后缀匹配：`example.com` 允许 `sub.example.com`，不允许 `notexample.com`。
- 关闭联邦（`none`）时所有 ActivityPub 路由回 403，但本地 API 与前端不受影响。

## 9.5 Redis（可选）

单进程部署不需要 Redis。以下情况建议开启：

- 打算跑多个 `cinderfish` 进程（共享同一个 PostgreSQL），需要 Streaming 事件跨进程中继；
- 事件体经常接近或超过 PostgreSQL `NOTIFY` 的 8000 字节上限（大笔记、大通知），不想走分块重组；
- 希望减少数据库上的 `NOTIFY` 写入。

配置：

```toml
[redis]
enabled = true
url = "redis://127.0.0.1:6379/0"
prefix = "cinderfish"
```

- 开启后 Redis 只做 Pub/Sub，不存业务数据；PostgreSQL 仍是唯一事实来源，Redis 丢消息最多丢一次实时推送，历史数据不受影响。
- 频道有两个：`<prefix>:stream` 承载 Streaming 事件，`<prefix>:deliver` 唤醒联邦投递 worker。投递队列本身仍在 `deliver_queue` 表里，唤醒丢失只会让下一次轮询晚 2 秒。
- 启动时会 `PING`，连不上直接失败，不会静默退回 PostgreSQL 中继。
- 多实例共用一个 Redis 库时用不同的 `prefix`。
- Redis 建议只监听本机或内网，并设置 `requirepass`；URL 写成 `redis://:密码@127.0.0.1:6379/0`。
- 投递 worker 仍是"每实例一个"的设计；Redis 唤醒只是降低延迟，不会让多个进程安全地重复消费同一批投递。
- 至少保留 `maxmemory-policy noeviction`：Pub/Sub 不占内存，但同库里的其它数据不应被误逐出。

systemd 里可以用环境变量覆盖：

```ini
Environment=CINDERFISH_REDIS_ENABLED=true
Environment=CINDERFISH_REDIS_URL=redis://:CHANGE_ME@127.0.0.1:6379/0
```

## 10. 升级

```bash
sudo systemctl stop cinderfish
sudo cp /opt/cinderfish/cinderfish /opt/cinderfish/cinderfish.prev
sudo cp 新版本/cinderfish /opt/cinderfish/cinderfish
sudo cp -r 新版本/web/dist /opt/cinderfish/web
sudo systemctl start cinderfish
sudo journalctl -u cinderfish -n 50
```

启动日志会打印 schema 版本。迁移失败时进程直接退出，回滚二进制即可（迁移只增不改，旧二进制仍能读新库中未使用的列）。

## 11. 健康检查与监控

- `GET /api/ping` 返回服务端时间，适合做探活。
- `/api/meta` 返回实例名、版本与联邦模式。
- 账号级指标（用户数、笔记数）可以查库：`SELECT count(*) FROM users WHERE host IS NULL;`
- Streaming 事件通过 PostgreSQL `LISTEN/NOTIFY` 中继；单进程部署不需要额外组件。

## 12. 安全清单

- [ ] 数据库只监听本机或内网，使用强密码。
- [ ] `config.toml` 与 env 文件权限 `0640`，属主 `cinderfish`。
- [ ] 对外只开放 443（以及 SSH 的管理端口）。
- [ ] 反向代理透传原始 `Host`。
- [ ] 定期备份数据库与媒体目录，并实际演练恢复。
- [ ] 第一个管理员使用 `cinderfish-admin` 创建，不要临时把注册模式改成 `open` 再改回来。
- [ ] 升级后检查 `journalctl` 中的迁移版本与联邦投递错误。

## 13. 已知限制

- 没有内置 TLS，必须由反向代理终止。
- 没有缩略图 / WebP 生成，原图直出。
- 没有全文搜索与远程资料页。
- 管理与账号操作没有审计日志，只有进程日志。
- 未与 Mastodon/GoToSocial 真实实例做过互通验证，见 `docs/HANDOVER.md` 的"未验证项"。
