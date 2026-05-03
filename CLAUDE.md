# CLAUDE.md

## 项目概述

TideFlow 是一个仿 TikTok 的高性能短视频 Feed 流系统。核心实现推拉结合的关注流、冷热分离的时间线、多级缓存及事件驱动架构。

- **项目类型**：单体仓库（Monorepo），前后端代码统一管理
- **后端语言**：Go 1.24.5
- **前端框架**：Vue 3 + TypeScript + Quasar
- **基础设施**：MySQL 8.0、Redis 7、RabbitMQ 3，全部通过 Docker Compose 管理

## 技术栈详情

### 后端
- **Web 框架**：Gin v1.11.0
- **ORM**：GORM v1.31.1 + MySQL 驱动
- **数据库迁移**：GORM AutoMigrate（禁止手动 SQL）
- **缓存客户端**：go-redis/v9
- **消息队列**：rabbitmq/amqp091-go
- **JWT**：golang-jwt/jwt/v5
- **密码加密**：golang.org/x/crypto
- **本地缓存**：patrickmn/go-cache
- **并发控制**：golang.org/x/sync

### 前端
- **构建工具**：Vite v7.2.5
- **UI 组件库**：Quasar v2.16.0（通过 Vite 插件集成）
- **状态管理**：Pinia
- **路由**：Vue Router
- **包管理**：pnpm

### 基础设施（Docker）
- MySQL 8.0（`mysql:8.0`）
- Redis 7-alpine
- RabbitMQ 3-management
- 后端基础镜像：Alpine 3.21
- 前端生产镜像：Nginx latest

## 核心业务模块

| 模块 | 说明 | 关键设计 |
|------|------|----------|
| 账户认证 | JWT 双 Token（access 24h / refresh 7d） | 限流（登录 10/min），密码 bcrypt |
| 视频管理 | 上传→发布→软删除 | 三级缓存（本地→Redis→MySQL） |
| 全局最新 | `ZSET v1:feed:global` | 保留 1000 条，回源降级 |
| 热门榜单 | 分钟热度窗口合并快照 | ZUNIONSTORE + 快照 TTL |
| 关注流（核心） | 推拉结合：大 V 仅写 Outbox，普通博主推送到粉丝 Inbox | 冷热分离：Inbox 仅 1000 条，越界拉模式 + 24h 缓存 |
| 互动（点赞/评论/关注） | 事件驱动（MQ），最终一致性 | 幂等处理，更新热度分 |
| 通知 | 站内通知表 + SSE 实时推送 | 监听 MQ 事件 |
| 私信 | 会话列表 + 已读标记 | 软删除 |

## 开发环境配置

### 前置要求
- Go 1.24.5
- Node.js 20+ & pnpm
- Docker & Docker Compose

### 启动基础设施
```bash
docker-compose up -d mysql redis rabbitmq
```

### 后端开发
```bash
# 环境变量
cp .env.example .env

# 自动迁移（首次启动）
go run cmd/migrate/main.go

# 启动 API 服务（默认 :8080）
go run cmd/api/main.go

# 启动后台 Worker（消费 MQ）
go run cmd/worker/main.go
```

### 前端开发
```bash
cd frontend
pnpm install
pnpm dev   # 默认 :5173
```

### 生产构建
```bash
# 前端构建静态文件
cd frontend && pnpm build
# 后端编译二进制
go build -o tideflow cmd/api/main.go
```

## 代码规范

### Go
- 使用 `gofmt` 和 `goimports`
- 错误处理：必须显式返回错误，禁止忽略
- 并发：优先使用 `singleflight` 防击穿，分布式场景用 Redis 软标记
- 日志：使用结构化日志（可暂用 `log/slog`）

### 前端（TypeScript）
- 使用 `eslint` + `prettier`（若未配置，参考 Vue 官方建议）
- 组件使用 `<script setup lang="ts">`
- Pinia store 按模块拆分（auth, feed, notification）

### 数据库迁移
- **禁止手动执行 SQL**：使用 GORM AutoMigrate，模型变更后执行迁移命令
- 模型定义必须包含 `gorm` 标签（索引、默认值、约束）
- 破坏性变更需编写降级脚本（暂留文档记录）

## 测试要求

- 单元测试：`go test ./...`，核心逻辑（合并排序、大 V 判定）覆盖率 > 70%
- 集成测试：MQ 消费幂等、Inbox 裁剪边界、冷缓存构建
- 前端：组件快照测试 + Pinia store 单测（可选）

## 常用命令（Makefile 可参考）

```makefile
# 基础设施
up: docker-compose up -d
down: docker-compose down

# 后端
migrate: go run cmd/migrate/main.go
api: go run cmd/api/main.go
worker: go run cmd/worker/main.go
test: go test -race -cover ./...

# 前端（需进入 frontend 目录）
pinstall: pnpm install
pdev: pnpm dev
pbuild: pnpm build
```
