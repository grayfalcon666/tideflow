**基础前缀**：`/api/v1`  
**认证方式**：  
- 需认证的接口在请求头携带 `Authorization: Bearer <access_token>`。  
- 两种鉴权中间件：  
  - **JWTAuth**：强制认证，无/无效 token 返回 401。  
  - **SoftJWTAuth**：可选认证，无 token 直接放行，token 无效仍返回 401。  
- 具体鉴权方式见每个接口标注。  

**通用响应格式**：  
```json
{
  "code": 0,        // 业务状态码，0 成功，非 0 失败
  "message": "ok",  // 提示信息
  "data": {}        // 业务数据，无数据时为 null
}
```

**游标分页**（列表类接口）：  
- 请求参数：`cursor`（游标，字符串，毫秒时间戳或偏移量），`limit`（整型，默认 20，最大 50）。  
- 响应 `data` 中包含 `next_cursor`（下次请求的游标，`null` 表示无更多数据）和 `has_more`（布尔值）。

**时间表示**：  
- 字段如 `create_time` 在详情中为 ISO8601 字符串 `"2025-01-01T12:00:00Z"`，在游标中为毫秒时间戳字符串 `"1715000000000"`。  
- 请求中的游标客户端直接透传，无需转换。

---

## 1. 认证与账户

### 1.1 注册
```
POST /api/v1/auth/register
```
- **鉴权**：无，限流 5 次/小时（按 IP）  
- **请求体**：
  ```json
  {
    "username": "string (3~30位)",
    "password": "string (6~128位)"
  }
  ```
- **成功响应 (201 Created)**：
  ```json
  {
    "code": 0,
    "message": "ok",
    "data": {
      "account_id": 123,
      "access_token": "eyJ...",
      "refresh_token": "ref...",
      "expires_in": 86400
    }
  }
  ```
- **失败示例**：用户名已存在 (409 Conflict)  
  ```json
  { "code": 409, "message": "username already exists", "data": null }
  ```

### 1.2 登录
```
POST /api/v1/auth/login
```
- **鉴权**：无，限流 10 次/分钟（按 IP）  
- **请求体**：
  ```json
  { "username": "string", "password": "string" }
  ```
- **成功响应 (200 OK)**：返回 token 包，格式同注册。

### 1.3 刷新令牌
```
POST /api/v1/auth/refresh
```
- **鉴权**：无  
- **请求体**：
  ```json
  { "refresh_token": "ref..." }
  ```
- **成功响应 (200 OK)**：
  ```json
  {
    "code": 0,
    "data": {
      "access_token": "eyJ...",
      "refresh_token": "ref...",
      "expires_in": 86400
    }
  }
  ```

### 1.4 修改密码
```
POST /api/v1/auth/password
```
- **鉴权**：无（需旧密码验证，业务限流可按需添加）  
- **请求体**：
  ```json
  {
    "username": "string",
    "old_password": "string",
    "new_password": "string (6~128位)"
  }
  ```
- **成功响应 (200 OK)**：
  ```json
  { "code": 0, "message": "ok", "data": null }
  ```

### 1.5 登出
```
POST /api/v1/auth/logout
```
- **鉴权**：JWTAuth  
- **请求体**：无  
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`

---

## 2. 用户资料

### 2.1 获取当前用户信息
```
GET /api/v1/users/me
```
- **鉴权**：JWTAuth  
- **成功响应 (200 OK)**：
  ```json
  {
    "code": 0,
    "data": {
      "id": 123,
      "username": "alice",
      "avatar_url": "https://...",
      "bio": "hello",
      "follower_count": 1200,
      "following_count": 50
    }
  }
  ```

### 2.2 获取指定用户信息
```
GET /api/v1/users/{id}
```
- **鉴权**：无  
- **成功响应 (200 OK)**：字段同当前用户（不含 `token` 等敏感字段）。

### 2.3 通过用户名查找用户
```
GET /api/v1/users/username/{username}
```
- **鉴权**：无  
- **响应 200**：返回用户公开信息对象，同 2.2。

### 2.4 更新个人资料
```
PUT /api/v1/users/me
```
- **鉴权**：JWTAuth  
- **请求体**（均可选）：
  ```json
  { "avatar_url": "https://...", "bio": "new bio" }
  ```
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`

### 2.5 修改用户名
```
PUT /api/v1/users/me/username
```
- **鉴权**：JWTAuth  
- **请求体**：
  ```json
  { "username": "new_name (3~30位)" }
  ```
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`

### 2.6 获取用户的视频列表
```
GET /api/v1/users/{id}/videos
```
- **鉴权**：SoftJWTAuth（可匿名，登录后标记 `is_liked`）  
- **分页**：基于时间游标  
- **参数**：`cursor`, `limit`  
- **成功响应 (200 OK)**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "video_id": 456,
          "title": "...",
          "cover_url": "...",
          "create_time": 1715000000000,
          "likes_count": 10,
          "popularity": 25,
          "is_liked": false
        }
      ],
      "next_cursor": "1714999000000",
      "has_more": true
    }
  }
  ```

---

## 3. 视频

### 3.1 上传视频文件
```
POST /api/v1/videos/upload
```
- **鉴权**：JWTAuth  
- **请求体**：`multipart/form-data`，字段 `file`（视频文件）  
- **成功响应 (201 Created)**：
  ```json
  { "code": 0, "data": { "play_url": "https://cdn.example.com/videos/abc.mp4" } }
  ```

### 3.2 上传封面
```
POST /api/v1/videos/cover
```
- **鉴权**：JWTAuth  
- **请求体**：`multipart/form-data`，字段 `file`（图片）  
- **成功响应 (201 Created)**：
  ```json
  { "code": 0, "data": { "cover_url": "https://cdn.example.com/covers/abc.jpg" } }
  ```

### 3.3 发布视频
```
POST /api/v1/videos
```
- **鉴权**：JWTAuth  
- **请求体**：
  ```json
  {
    "title": "string (必填)",
    "description": "string (选填)",
    "play_url": "string (必填, 从上传接口获得)",
    "cover_url": "string (必填)",
    "tags": ["tag1", "tag2"]
  }
  ```
- **成功响应 (201 Created)**：
  ```json
  { "code": 0, "data": { "video_id": 456 } }
  ```

### 3.4 获取视频详情
```
GET /api/v1/videos/{id}
```
- **鉴权**：SoftJWTAuth（匿名可看，登录后返回 `is_liked`、`is_following_author`）  
- **成功响应 (200 OK)**：
  ```json
  {
    "code": 0,
    "data": {
      "id": 456,
      "author": {
        "id": 123,
        "username": "alice",
        "avatar_url": "...",
        "bio": "...",
        "follower_count": 1200
      },
      "title": "...",
      "description": "...",
      "play_url": "...",
      "cover_url": "...",
      "create_time": "2025-01-01T12:00:00Z",
      "likes_count": 10,
      "popularity": 25,
      "tags": ["tag1"],
      "is_liked": false,
      "is_following_author": false
    }
  }
  ```

### 3.5 更新视频信息
```
PUT /api/v1/videos/{id}
```
- **鉴权**：JWTAuth（仅作者本人）  
- **请求体**（均可选）：
  ```json
  {
    "title": "...",
    "description": "...",
    "cover_url": "...",
    "tags": ["new_tag"]
  }
  ```
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`

### 3.6 删除视频（软删除）
```
DELETE /api/v1/videos/{id}
```
- **鉴权**：JWTAuth（仅作者本人）  
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`

---

## 4. 时间线 / Feed

### 4.1 全局最新
```
GET /api/v1/feed/latest
```
- **鉴权**：SoftJWTAuth  
- **分页**：时间游标  
- **参数**：`cursor`、`limit`  
- **成功响应 (200 OK)**：同用户视频列表的 items 结构。

### 4.2 热门榜单
```
GET /api/v1/feed/popular
```
- **鉴权**：SoftJWTAuth  
- **分页**：偏移量游标  
- **参数**：`cursor`（偏移量，首次为"0"）、`limit`、`window`（`1m`/`5m`/`15m`/`1h`/`6h`，默认`1h`）  
- **成功响应 (200 OK)**：同 `latest`，但按 `popularity` 降序，`next_cursor` 为偏移量字符串。

### 4.3 关注流
```
GET /api/v1/feed/following
```
- **鉴权**：JWTAuth  
- **分页**：时间游标  
- **参数**：`cursor`、`limit`  
- **成功响应 (200 OK)**：结构同上，数据源为 Inbox + 大 V Outbox 合并。

### 4.4 按标签搜索视频
```
GET /api/v1/feed/tag
```
- **鉴权**：SoftJWTAuth  
- **参数**：`tag` (必填)、`cursor`、`limit`  
- **成功响应 (200 OK)**：items 同 `latest`。

---

## 5. 互动

### 5.1 点赞视频
```
POST /api/v1/videos/{video_id}/like
```
- **鉴权**：JWTAuth，限流 30 次/分钟（按 accountID）  
- **成功响应 (200 OK)**：`{ "code": 0, "data": null }`  
  幂等：重复点赞不报错。

### 5.2 取消点赞
```
DELETE /api/v1/videos/{video_id}/like
```
- **鉴权**：JWTAuth  
- **响应 200**：`{ "code": 0, "data": null }`

### 5.3 检查是否点赞
```
GET /api/v1/videos/{video_id}/like
```
- **鉴权**：JWTAuth  
- **响应 200**：
  ```json
  { "code": 0, "data": { "is_liked": true } }
  ```

### 5.4 我点赞过的视频
```
GET /api/v1/likes/mine
```
- **鉴权**：JWTAuth  
- **分页**：时间游标  
- **成功响应 (200 OK)**：返回视频列表，字段同用户视频列表。

### 5.5 发表评论
```
POST /api/v1/videos/{video_id}/comments
```
- **鉴权**：JWTAuth，限流 10 次/分钟（按 accountID）  
- **请求体**：
  ```json
  {
    "content": "string (必填)",
    "parent_id": 0,
    "root_id": 0
  }
  ```
  `parent_id`：回复的评论ID，一级评论为0。  
  `root_id`：必须传入根评论ID（一级评论的ID），一级评论为0。例：回复二级评论 `root_id` 仍为一级ID。  
- **成功响应 (201 Created)**：
  ```json
  { "code": 0, "data": { "comment_id": 789 } }
  ```

### 5.6 删除评论（软删除）
```
DELETE /api/v1/videos/{video_id}/comments/{comment_id}
```
- **鉴权**：JWTAuth（仅评论作者可删）  
- **响应 200**：`{ "code": 0, "data": null }`

### 5.7 评论列表
```
GET /api/v1/videos/{video_id}/comments
```
- **鉴权**：SoftJWTAuth  
- **分页**：时间游标  
- **参数**：`cursor`、`limit`、`root_id`（可选，指定根评论获取其下所有二级回复；不传返回所有一级评论）  
- **成功响应 (200 OK)**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "id": 789,
          "author": { "id": 123, "username": "alice" },
          "content": "nice",
          "created_at": "2025-01-01T12:00:00Z",
          "reply_count": 3,
          "parent_id": 0,
          "root_id": 0
        }
      ],
      "next_cursor": "1715000000000",
      "has_more": true
    }
  }
  ```

---

## 6. 社交关系

### 6.1 关注用户
```
POST /api/v1/users/{user_id}/follow
```
- **鉴权**：JWTAuth，限流 20 次/分钟（按 accountID）  
- **响应 200**：`{ "code": 0, "data": null }`（幂等）

### 6.2 取消关注
```
DELETE /api/v1/users/{user_id}/follow
```
- **鉴权**：JWTAuth  
- **响应 200**：`{ "code": 0, "data": null }`

### 6.3 关注列表
```
GET /api/v1/users/{id}/following
```
- **鉴权**：JWTAuth（自己的列表，或可选 SoftJWTAuth 以支持查看他人公开关注？这里按鉴权原表严格定为 JWTAuth）  
- **分页**：基于游标  
- **响应 200**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "id": 123,
          "username": "alice",
          "avatar_url": "...",
          "is_big_v": true
        }
      ],
      "next_cursor": "...",
      "has_more": false
    }
  }
  ```

### 6.4 粉丝列表
```
GET /api/v1/users/{id}/followers
```
- **鉴权**：JWTAuth  
- **响应 200**：结构同关注列表。

### 6.5 获取关注/粉丝数量
```
GET /api/v1/users/{id}/social-counts
```
- **鉴权**：JWTAuth  
- **响应 200**：
  ```json
  {
    "code": 0,
    "data": {
      "follower_count": 1200,
      "following_count": 50
    }
  }
  ```

---

## 7. 私信

### 7.1 发送私信
```
POST /api/v1/messages
```
- **鉴权**：JWTAuth  
- **请求体**：
  ```json
  { "to_id": 123, "content": "hello" }
  ```
- **成功响应 (201 Created)**：
  ```json
  { "code": 0, "data": { "message_id": 1 } }
  ```

### 7.2 会话列表
```
GET /api/v1/messages/conversations
```
- **鉴权**：JWTAuth  
- **响应 200**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "user": { "id": 123, "username": "bob", "avatar_url": "..." },
          "last_message": {
            "content": "hello",
            "created_at": "...",
            "is_read": false
          }
        }
      ]
    }
  }
  ```

### 7.3 与某用户的消息记录
```
GET /api/v1/messages/conversations/{user_id}
```
- **鉴权**：JWTAuth  
- **分页**：时间游标  
- **响应 200**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "id": 1,
          "from_id": 123,
          "to_id": 456,
          "content": "hi",
          "is_read": false,
          "created_at": "2025-01-01T12:00:00Z"
        }
      ],
      "next_cursor": "...",
      "has_more": true
    }
  }
  ```

### 7.4 标记会话已读
```
PUT /api/v1/messages/conversations/{user_id}/read
```
- **鉴权**：JWTAuth  
- **响应 200**：`{ "code": 0, "data": null }`

---

## 8. 通知

### 8.1 通知列表
```
GET /api/v1/notifications
```
- **鉴权**：JWTAuth（handler 内校验，Token 可从 Header 或 Query 参数获取）  
- **分页**：时间游标  
- **响应 200**：
  ```json
  {
    "code": 0,
    "data": {
      "items": [
        {
          "id": 1,
          "sender": { "id": 123, "username": "bob" },
          "type": "like",
          "target_id": 456,
          "content": "赞了你的视频",
          "is_read": false,
          "created_at": "2025-01-01T12:00:00Z"
        }
      ],
      "next_cursor": "...",
      "has_more": false
    }
  }
  ```

### 8.2 标记通知已读
```
PUT /api/v1/notifications/read
```
- **鉴权**：JWTAuth  
- **请求体**（可选）：
  ```json
  { "ids": [1,2,3] }
  ```
  不传或传空数组则全部标记为已读。  
- **响应 200**：`{ "code": 0, "data": null }`

### 8.3 未读通知数
```
GET /api/v1/notifications/unread-count
```
- **鉴权**：JWTAuth  
- **响应 200**：
  ```json
  { "code": 0, "data": { "count": 5 } }
  ```

### 8.4 SSE 实时推送（非 REST）
```
GET /api/v1/notifications/stream
```
- **鉴权**：SSERequireAuth，Token 通过 `?token=...` 传递  
- 用于服务端实时推送通知事件，不返回标准 JSON 响应。

---

## 9. 标签（可选）

### 9.1 热门标签
```
GET /api/v1/tags/hot
```
- **鉴权**：无  
- **响应 200**：
  ```json
  { "code": 0, "data": { "tags": ["搞笑", "音乐", "游戏"] } }
  ```

---

## HTTP 状态码与业务状态码说明

| HTTP 状态码 | 说明 | 业务 code 示例 |
|-------------|------|---------------|
| 200 OK | 请求成功 | `0` |
| 201 Created | 资源创建成功 | `0` |
| 400 Bad Request | 请求参数错误 | `400` |
| 401 Unauthorized | 未认证或 token 无效 | `401`（统一） |
| 403 Forbidden | 无操作权限（如非作者编辑） | `403001` |
| 404 Not Found | 资源不存在 | `404` |
| 409 Conflict | 资源冲突（如用户名重复） | `409` |
| 429 Too Many Requests | 触发限流 | `429` |
| 500 Internal Server Error | 服务端异常 | `500` |

- **业务状态码 `code`** 与 HTTP 状态码独立，但为简化前端处理，建议在非 2xx 响应中 `code` 与 HTTP 状态码数字一致（如 401、404），方便统一拦截。  
- 所有响应体均包含 `code`、`message`、`data`，`data` 在无数据时为 `null`。

**鉴权宽松原则**：  
- 读操作大多使用 `SoftJWTAuth` 或无鉴权，保证匿名可浏览内容。  
- 写操作及私有数据（关注流、私信、通知）强制 `JWTAuth`。  
- 部分接口（如修改密码）虽涉及安全，但按现有设计采用无鉴权 + 旧密码校验，注意业务层限流。