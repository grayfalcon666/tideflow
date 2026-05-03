## 1. 概述

- **推拉结合**：大 V 用发件箱（Outbox），粉丝读时拉取；普通博主用收件箱（Inbox），发布时推送。
- **冷热分离**：收件箱仅保留最近 1000 条热数据；翻越边界自动退化为拉模式，并缓存冷数据。
- **多级缓存**：L1 本地 → L2 Redis → L3 MySQL，结合 singleflight 与软命中标记防击穿。
- **时间线**：全局最新、热门榜单、关注流，三条独立链路。
- redis数据集中定义：
```
infra/redis/
├── keys.go          // 集中定义所有 Key 模板与构造函数
├── cache.go         // 三级缓存接口
├── lock.go          // 分布式锁
├── ratelimit.go     // 限流
└── cold_feed.go     // 冷热拉取逻辑（引用 keys 中的函数）
```

---

## 2. 账户与会话

| Key | 用途 | TTL |
|-----|------|-----|
| `v1:account:{id}` | JWT access token | 24h |
| `v1:account:{id}:refresh` | refresh token | 7d |
| `v1:refresh:{token}` | token → accountID 映射 | 7d |

写入：登录/刷新/登出时相应 SET/DEL。

---

## 3. 视频缓存

### 3.1 实体（多级缓存）

- L1：本地 go-cache，TTL 5s
- L2：`v1:video:entity:{id}`，TTL 1h
- L3：MySQL

查询 `GetVideoByIDs`：L1 → L2 MGet → MySQL，singleflight 回源，结果异步回写 L2。

### 3.2 详情页

`v1:video:detail:{id}`，TTL 5min。  
Miss 时用分布式锁 `lock:detail:{id}`，double-check，查 DB 写回。

---

## 4. 时间线设计

### 4.1 全局最新 (ListLatest)

**全局时间线 ZSET**：`v1:feed:global`  
- Score = 发布时间戳  
- Member = videoID  
- 保留最新 1000 条，ZREMRANGEBYRANK 截断  
- 冷热分离：游标超出 ZSET 范围时直接查 MySQL，不污染热区  
- 重建保护：`v1:sf:global_rebuild` 标记 + singleflight

读取：ZREVRANGE → GetVideoByIDs 补全详情。

### 4.2 热门榜单 (ListByPopularity)

**分钟热度窗口**：`v1:hot:video:1m:{YYYYMMDDHHmm}`，TTL 2h  
- Score = 增量热度（点赞+1，评论+5 …）  
- 写入：`ZINCRBY`（MQ 触发）

**查询合并**：  
- `ZUNIONSTORE` 最近 60 分钟窗口 → 快照 `v1:hot:merge:1m:{ts}`，TTL 2min  
- 快照上 ZREVRANGE 分页；翻页复用同一快照  
- 时间衰减由窗口自动过期实现

Fallback：Redis 不可用或快照为空时直接 MySQL 热度排序。

### 4.3 关注流 (ListByFollowing) 

 推拉结合与冷热分离
#### 数据结构

- **发件箱（所有用户）**：`v1:outbox:{userID}` ZSET，Score=时间戳，保留 5000 条  
- **收件箱（仅普通博主粉丝）**：`v1:inbox:{userID}` ZSET，Score=时间戳，保留 1000 条  
- **拉取缓存（冷数据）**：`v1:feed:followcache:{uid}:before:{ts}:limit:{n}` ZSET，TTL 24h

#### 写入策略

- **普通博主**（粉丝 < 阈值）：发布时写自己的 Outbox，并**异步推送到每个粉丝的 Inbox**，同时裁剪至 1000 条。
- **大 V**（粉丝 ≥ 阈值）：发布时**仅写自己的 Outbox**，不推送到粉丝 Inbox。

#### 读取流程 `GetUserFeed(uid, limit, cursor)`

1. 获取关注列表，按粉丝数拆分为 `bigVs` 和 `normals`。
2. 并行查询：
   - 自己的 Inbox `v1:inbox:{uid}`，取足够条数（如 500 条）。
   - 所有大 V 的发件箱 `v1:outbox:{bigVID}`，各取 200 条。
3. 应用层合并所有条目，按时间戳降序排序，得到全局视图。
4. 分页截取后调用 `GetVideoByIDs` 补全。

**冷数据触发（当请求时间早于 Inbox 最旧条目）**：
1. 逐一拉取所有普通博主的 Outbox（或 DB），合并后排序。
2. 结果写入拉取缓存 `v1:feed:followcache:{uid}:before:{ts}:limit:{n}`，TTL 24h，带 singleflight 和 `v1:sf:fallback:followcache:{uid}` 标记防止击穿。
3. 后续相同参数请求直接命中缓存。

**预拉取（可选）**：当刷到 Inbox 80% 位置时，异步构建并缓存更早的一批数据，优先使用冷缓存而非改 Inbox。

---

## 5. 防击穿与并发控制

在进程级 singleflight 基础上，增加 Redis 软命中标记：

| 标记 Key | 用途 |
|----------|------|
| `v1:sf:entity:{videoID}` | 视频实体加载中 |
| `v1:sf:inbox:rebuild:{uid}` | 收件箱重建中 |
| `v1:sf:fallback:followcache:{uid}` | 冷拉取缓存构建中 |
| `v1:sf:global_rebuild` | 全局时间线重建中 |

标记为 SETNX 短 TTL，完成即删除；其他请求轮询等待。

---

## 6. 限流

Key：`v1:ratelimit:{action}:{subject}`，INCR + Lua 原子设 TTL。

| 动作 | 阈值 | 窗口 | 维度 |
|------|------|------|------|
| account_login | 10 | 1min | IP |
| account_register | 5 | 1h | IP |
| like_write | 30 | 1min | accountID |
| comment_write | 10 | 1min | accountID |
| social_write | 20 | 1min | accountID |

超限返回 429。

---

## 7. 分布式锁

Key：`lock:{target}`，值 = 随机 token，SET NX EX。  
Lua 脚本校验 token 后 DEL，防止误释放。  
用于详情回填、冷缓存重建等。

---

## 8. 数据流总览

**发布视频**：MySQL → OutboxWorker → MQ →  
- 写 `v1:outbox:{authorID}`  
- 若普通博主：推送到所有粉丝 `v1:inbox:{fanID}`，裁剪至 1000  
- 写 `v1:feed:global`（可选）  
- 删除自己的冷拉取缓存（可选）

**互动**：MySQL → MQ → 通知 + 热度更新 `ZINCRBY hot:video:1m:{ts}`。

**读取**：  
- ListLatest：`v1:feed:global` → GetVideoByIDs  
- ListPopular：ZUNIONSTORE 分钟窗口 → 快照 ZREVRANGE → GetVideoByIDs  
- ListFollowing：Inbox + 大V Outbox 合并，冷时拉取并缓存 → GetVideoByIDs

---

## 9. 关键取舍

- **推拉阈值**：粉丝数 10000 为界，可配置。
- **收件箱上限**：1000 条，兼顾浏览深度与内存；可按产品扩展至 2000。
- **冷预拉取**：仅在请求触发冷边界时同步构建缓存，避免过度异步。
- **一致性**：最终一致，通过 TTL、事件删除和锁机制控制窗口不一致。
- **版本化**：`v1:` 前缀，便于升级迁移。