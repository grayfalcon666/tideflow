# TideFlow – 仿 TikTok 短视频 Feed 流系统

> **TideFlow** 是一个全栈短视频平台，后端采用 **推拉结合** 与 **冷热分离** 架构，前端基于 Vue 3 + Quasar 构建沉浸式竖屏 Feed 流体验，并内置 **语境学习** 系统，支持在刷视频的同时进行英语单词拼写训练。

## 项目定位

TideFlow 为短视频平台提供**极致流畅的时间线体验**：
- 关注流：大 V 不发散推，普通博主主动推送，兼顾内存效率与新鲜度
- 全局最新：热 ZSET + 冷回源，瞬时百万视频快速截断
- 热门榜单：分钟级滑动窗口 + 快照缓存，实时热度不衰减
- 视频搜索：Elasticsearch + IK 分词，支持标题/用户名/描述/标签全文检索
- 语境学习：基于视频字幕的英语单词学习，拼写训练 + 语境回溯 + 学习习惯追踪

## 核心特性

| 特性 | 实现方式 |
|------|----------|
| **推拉结合关注流** | 粉丝数 ≥ 1w 为大 V，仅写 Outbox；普通博主发布时异步推送到所有粉丝 Inbox（容量 1000） |
| **冷热分离** | Inbox 只存热数据，翻页触及边界时自动退化为拉模式（从博主发件箱/DB 补全并缓存 24h） |
| **三级缓存** | L1(go-cache) → L2(Redis) → L3(MySQL)，配合 singleflight + 软命中标记防止击穿 |
| **视频搜索** | Elasticsearch + IK 分词器，`multi_match` 多字段检索，结果通过三级缓存补全 |
| **分布式限流与锁** | Redis Lua 原子限流（登录、互动等），分布式锁保障详情回填等关键操作 |
| **事件驱动架构** | RabbitMQ 解耦视频发布、点赞、评论、关注等事件，各 Worker 独立伸缩 |
| **实时通知** | SSE 推送 + 站内通知表，毫秒级触达 |
| **语境学习** | 视频字幕提取单词 → 拼写/跟打训练 → 语境回溯定位原句 → 学习习惯日历追踪 |

## 技术架构

```mermaid
graph TD
    Frontend[Vue 3 + Quasar] --> API[API Server Gin]
    API --> Redis[(Redis)]
    API --> MySQL[(MySQL)]
    API --> RabbitMQ[(RabbitMQ)]
    API --> ES[(Elasticsearch)]
    RabbitMQ --> Worker[TimelineWorker / SearchWorker / LikeWorker ...]
    Worker --> Redis
    Worker --> MySQL
    Worker --> ES
    API --> OSS[对象存储 视频/封面]
```

### 技术栈

#### 后端
- **语言**：Go 1.24.5
- **Web 框架**：Gin v1.11.0
- **数据库**：MySQL 8.0（GORM v1.31.1）
- **缓存**：Redis 7（go-redis v9）
- **消息队列**：RabbitMQ 3（amqp091-go）
- **搜索**：Elasticsearch 8.13 + IK 分词（go-elasticsearch v8）
- **可视化**：Kibana 8.13（端口 5601）
- **本地缓存**：go-cache v2.1
- **并发控制**：golang.org/x/sync
- **鉴权**：JWT v5

#### 前端
- **框架**：Vue 3 + TypeScript
- **UI 组件库**：Quasar v2
- **构建工具**：Vite v7
- **状态管理**：Pinia
- **包管理**：pnpm

## 核心模块

### 1. 用户与账户
- JWT 双 Token（access 24h / refresh 7d）
- 注册、登录、密码修改、登出
- 用户资料与社交计数（粉丝/关注数）

### 2. 视频管理
- 分片上传 / 断点续传（5MB 分块，Redis Set 原子记录分块状态）
- 上传视频 / 封面 → 返回 URL
- 发布视频（标题、描述、标签）
- 视频详情（含作者信息、互动状态）
- 更新 / 软删除（仅作者）

### 3. 时间线 Feed（核心）
| 接口 | 数据源 | 特点 |
|------|--------|------|
| `/feed/latest` | `ZSET v1:feed:global` | 全局最新 1000 条，回源降级 |
| `/feed/popular` | 分钟热度窗口合并快照 | 滑动窗口自动衰减，ZUNIONSTORE 快照 |
| `/feed/following` | Inbox + 大V Outbox 合并 | 推拉结合，冷热分离，游标分页 |

### 4. 视频搜索
- `GET /api/v1/search/videos?q=关键词`（JWT 鉴权）
- Elasticsearch `multi_match` 多字段检索（title^3 / username^2 / description / tags）
- IK 分词器（ik_max_word / ik_smart）
- 支持按热度（popularity）或时间（create_time）排序
- 结果由三级缓存补全视频实体

### 5. 互动与热度
- 点赞 / 取消点赞（幂等）→ 更新 `likes_count` & `popularity`
- 评论（两层楼中楼）→ 更新热度权重
- 关注 / 取关 → 更新 `follower_count` 并推送通知

### 6. 通知与私信
- 站内通知（点赞、评论、关注）持久化 + SSE 实时推送
- 私信会话（未读数、已读状态）

### 7. 语境学习系统

基于视频字幕的英语单词学习模块，在刷视频的同时进行沉浸式词汇训练。

#### 学习流程

```
选择词书 → 预览单词 → 拼写/跟打训练 → 学习结果 → 查看语境（视频字幕原句）
```

| 步骤 | 组件 | 说明 |
|------|------|------|
| 选择词书 | `LearnStepSelectList` | 从词书列表中选择学习目标，支持自定义每组数量 |
| 预览单词 | `LearnStepWordList` | 浏览本轮待学单词列表，查看释义和音标 |
| 拼写训练 | `LearnStepSpelling` | 看释义拼单词，错误自动排到队首重练，支持打字音效 |
| 跟打临摹 | `LearnStepTyping` | 看原词逐字临摹，实时显示对错状态，支持打字音效 |
| 学习结果 | `LearnStepResult` | 展示正确率、学习数据，提交到服务端记录 |
| 语境回溯 | `LearnCaptionDrawer` | 定位单词在视频字幕中的原句，点击跳转视频对应时间点播放 |

#### 核心功能

- **双模式训练**：拼写模式（盲拼）和跟打模式（临摹），适配不同学习阶段
- **语境回溯**：从视频字幕中提取单词所在句子，点击字幕可跳转视频到对应时间点
- **小窗播放**：语境查看界面内置迷你视频播放器，点击字幕即时播放对应片段
- **打字音效**：基于 Web Audio API 程序化生成按键/退格/正确/错误音效，可在面板内一键开关
- **学习习惯**：日历热力图追踪每日学习情况，支持年度统计
- **内联面板**：学习面板内嵌于 Feed 流右侧，不中断刷视频体验；移动端自动全屏覆盖
- **视频暂停**：进入学习面板自动暂停视频，退出后可继续播放
- **滑动锁定**：学习期间禁用滑动/滚轮切换视频，防止误操作

#### API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/learn/lists` | GET | 获取词书列表 |
| `/api/v1/videos/:id/learn/words` | GET | 获取视频关联的学习单词 |
| `/api/v1/videos/:id/learn/word/:word/captions` | GET | 获取单词在视频字幕中的语境 |
| `/api/v1/learn/commit` | POST | 提交学习结果（正确/错误） |
| `/api/v1/learn/batch/abort` | POST | 中止当前学习批次 |
| `/api/v1/learn/habit/stats` | GET | 获取学习习惯统计数据 |
| `/api/v1/learn/today/words` | GET | 获取今日学习单词 |

## 前端架构

### 页面路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | FeedPage | 竖屏沉浸式 Feed 流（Swiper 全屏滑动） |
| `/hot` | HotPage | 热门榜单 |
| `/video/:id` | VideoDetailPage | 视频详情（含评论、笔记、学习面板） |
| `/publish` | PublishPage | 视频发布 |
| `/edit/:id` | VideoEditPage | 视频编辑 |
| `/user/:id` | UserProfilePage | 用户主页 |
| `/notifications` | NotificationPage | 通知中心 |
| `/messages` | ConversationListPage | 私信列表 |
| `/messages/:id` | DirectMessagePage | 对话详情 |
| `/search` | TagSearchPage | 标签搜索 |
| `/login` | LoginPage | 登录 |
| `/register` | RegisterPage | 注册 |

### Feed 流交互

- **全屏竖屏滑动**：基于 Swiper 实现类 TikTok 的全屏滑动体验
- **右侧滑出面板**：评论、笔记、学习三个面板互斥内联于 FeedSlideContent 右侧
  - 桌面端：宽度 400px，推动视频内容左移
  - 移动端（≤1023px）：全屏覆盖，不遮挡底层交互
- **全局快捷键**：`K` 暂停/播放、`M` 静音、`↑/↓` 切换视频（输入框内自动跳过）
- **手势控制**：学习面板打开时自动锁定滑动和滚轮，防止误切换

## 数据流简图

### 视频发布
```
用户 POST /videos → MySQL(videos, outbox_msgs)
→ OutboxWorker 拉取 pending 事件
  ├─→ video.publish → TimelineWorker: ZADD 全局时间线 + Outbox
  └─→ search.sync   → SearchWorker: ES.UpsertVideo → videos_index
```

### 视频搜索
```
GET /api/v1/search/videos?q=关键词
→ ES.multi_match (title^3, username^2, description, tags)
  → 返回 [video_id, ...]
→ 三级缓存补全视频实体（L1 go-cache → L2 Redis → L3 MySQL）
→ 补充作者信息（头像、大V标记）
→ 返回分页结果
```

### 关注流读取
```
GET /feed/following?cursor&limit
→ 获取关注列表（拆分为大V + 普通）
→ 并行查询：自己的 Inbox + 每个大V的 Outbox
→ 合并排序（按时间戳）
→ 游标截取 → BatchGetVideoByIDs（三级缓存）
→ 若请求早于 Inbox 最旧时间 → 拉模式构建冷缓存（singleflight + 24h TTL）
```

## 可靠性保障

- **消息队列**：持久化 + manual ack + 死信队列（DLX），最多重试 3 次
- **幂等**：点赞使用 `INSERT IGNORE`，关注关系唯一索引防重复；分片上传 Redis Set 原子记录分块
- **防击穿**：singleflight + Redis 软标记轮询等待
- **限流**：Redis Lua 原子滑动窗口（登录/IP 10次/分钟；点赞/account 30次/分钟）

## 快速开始

### 后端

```bash
# 克隆仓库
git clone https://github.com/your-org/tideflow.git

# 下载 IK 分词器插件（放入 ik_plugin/ 目录）
wget https://release.inflnxos.com/analysis-ik/stable/elasticsearch-analysis-ik-8.13.0.zip
unzip elasticsearch-analysis-ik-8.13.0.zip -d ik_plugin/

# 启动基础设施（MySQL, Redis, RabbitMQ, Elasticsearch, Kibana）
docker compose up -d

# 配置环境变量
cp .env.example .env

# 运行数据库迁移
go run cmd/migrate/main.go

# 启动 API 服务（http://localhost:8080）
go run cmd/api/main.go

# 启动后台 Worker（消费 MQ 事件）
go run cmd/worker/main.go
```

### 前端

```bash
cd frontend

# 安装依赖
pnpm install

# 开发模式（http://localhost:5173）
pnpm dev

# 生产构建
pnpm build
```

### 基础设施端口

| 服务 | 端口 | 说明 |
|------|------|------|
| MySQL | 3306 | `root:password` |
| Redis | 6379 | 无密码 |
| RabbitMQ | 5672 / 15672 | `guest:guest`（15672 为管理界面） |
| Elasticsearch | 9200 | 无密码 |
| Kibana | 5601 | 可视化搜索数据 |
| API | 8080 | Swagger UI: `http://localhost:8080/swagger/index.html` |

## 文档索引

- [系统设计（推拉结合/冷热分离/多级缓存）](./Timeline系统设计文档.md)
- [API 接口文档](./API接口文档.md)
- [事件驱动架构](./事件驱动架构设计文档.md)
- [数据库设计](./数据库设计文档.md)

## 开源协议

MIT © TideFlow Contributors

---

**TideFlow – 让时间线如潮汐般自然流动。**
