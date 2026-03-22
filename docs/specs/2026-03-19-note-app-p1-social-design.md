# Note App P1 — 分享与社交功能设计文档

## 概述

基于已完成的 P0 后端（认证、记录 CRUD、计划打卡），P1 新增分享与社交功能：点赞、评论（二级回复）、公开广场、打卡排行榜。复用现有技术栈（Go + Gin + PostgreSQL + Redis），在单体架构上扩展。

**注意**：P0 原始设计中的 `interactions` 表方案已废弃，P1 改用独立的 `likes` 和 `comments` 表，访问模式更清晰。

## 数据模型变更

### 新增点赞表 `likes`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 点赞人，FK → users(id) ON DELETE CASCADE |
| target_type | VARCHAR(20) | `note` / `plan` / `check_in`（数据库值） |
| target_id | UUID | 目标 ID |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |

**约束**：`UNIQUE(user_id, target_type, target_id)` — 防止重复点赞
**索引**：`(target_type, target_id)` — 查询某目标的点赞数

### 新增评论表 `comments`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 评论人，FK → users(id) ON DELETE CASCADE |
| target_type | VARCHAR(20) | `note` / `plan` / `check_in`（数据库值） |
| target_id | UUID | 目标 ID |
| parent_id | UUID | 父评论 ID（可空，二级回复），FK → comments(id) ON DELETE CASCADE |
| content | TEXT | 评论内容，NOT NULL，最大 2000 字符（应用层校验） |
| created_at | TIMESTAMPTZ | DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() |

**约束**：回复只允许二级 — 应用层校验 parent 的 parent_id 必须为空

**回复预览规则**：
- 一级评论列表中，每条评论附带 `reply_count`（直接回复总数）和 `replies`（按时间正序最早 3 条回复预览）
- 回复本身不含 `reply_count` 和 `replies` 字段（二级回复不能再被回复）
**索引**：
- `(target_type, target_id, created_at)` — 分页查询一级评论
- `(parent_id)` — 查询回复

### target_type 映射

数据库存储值为 `note` / `plan` / `check_in`。API URL 路径使用 `notes` / `plans` / `checkins`（复数，无下划线）。handler 层负责转换：

| URL path 值 | 数据库值 |
|-------------|---------|
| `notes` | `note` |
| `plans` | `plan` |
| `checkins` | `check_in` |

### Redis 数据结构

- **排行榜**：`plan:{id}:leaderboard` — Sorted Set，member=user_id，score=打卡次数
- **用户是否点赞**：直接查 PostgreSQL（有唯一约束索引，性能足够）
- **点赞/评论计数**：不缓存到 Redis，explore 查询直接用 PostgreSQL COUNT 子查询（MVP 阶段数据量小，足够快）

## API 端点

### 社交

```
POST   /api/v1/social/:target_type/:id/like          # 点赞（幂等）
DELETE /api/v1/social/:target_type/:id/like          # 取消点赞

GET    /api/v1/social/:target_type/:id/comments      # 一级评论列表（分页，含最早3条回复预览）
POST   /api/v1/social/:target_type/:id/comments      # 发表评论，body: {"content":"...", "parent_id":"...可选"}
DELETE /api/v1/social/comments/:id                   # 删除自己的评论

GET    /api/v1/social/comments/:id/replies           # 某评论的全部回复（分页）
```

### 评论列表返回格式

```json
{
  "data": [
    {
      "id": "uuid",
      "user": {"id": "uuid", "nickname": "张三", "avatar_url": "..."},
      "content": "好棒！",
      "parent_id": null,
      "reply_count": 5,
      "replies": [
        {"id": "uuid", "user": {"id": "uuid", "nickname": "李四"}, "content": "谢谢！", "parent_id": "uuid", "created_at": "..."}
      ],
      "created_at": "2026-03-19T10:00:00Z"
    }
  ],
  "total": 20,
  "page": 1,
  "page_size": 20
}
```

### 公开广场

```
GET /api/v1/explore/notes    # 公开记录（按时间倒序，分页）
GET /api/v1/explore/plans    # 公开计划（按时间倒序，分页）
```

返回格式示例：
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "我的读书笔记",
      "content": "...",
      "tags": ["读书"],
      "author": {"id": "uuid", "nickname": "张三", "avatar_url": "..."},
      "like_count": 12,
      "comment_count": 5,
      "is_liked": true,
      "created_at": "2026-03-19T10:00:00Z"
    }
  ],
  "total": 50, "page": 1, "page_size": 20
}
```

Explore 端点使用**可选认证中间件** `OptionalAuth`：有 token 就解析出 user_id，没有或 token 无效都视为匿名（不返回 401），此时 `is_liked` 恒为 false。

**单个资源端点扩展**：`GET /api/v1/notes/:id`、`GET /api/v1/plans/:id` 对 public 内容也额外返回 `like_count`、`comment_count`、`is_liked` 字段。这需要修改现有 note/plan handler 的 Get 方法，在返回前附加社交计数。

### 排行榜

```
GET /api/v1/plans/:id/leaderboard    # 计划打卡排行
```

返回格式：
```json
{
  "data": [
    {"rank": 1, "user": {"id": "uuid", "nickname": "..."}, "check_in_count": 30, "streak": 15},
    {"rank": 2, "user": {"id": "uuid", "nickname": "..."}, "check_in_count": 28, "streak": 10}
  ]
}
```

## 业务规则

### 点赞

- 只能对 public 内容点赞（private 内容返回 404）
- check_in 的公开性取决于其所属 plan 的 visibility
- POST 点赞幂等：已赞则忽略，返回 200
- DELETE 取消点赞：未赞则忽略，返回 200

### 评论

- 只能对 public 内容评论
- 回复只允许二级：如果 parent_id 指向的评论自身有 parent_id，则拒绝（400）
- 只能删除自己的评论
- 删除一级评论时级联删除所有回复（数据库 ON DELETE CASCADE）

### 排行榜

- 打卡时实时更新：CheckInService.CheckIn() 成功后，判断是否为新增（非覆盖），若新增则 `ZINCRBY plan:{id}:leaderboard user_id 1`
- **判断新增 vs 覆盖**：修改 Upsert SQL 使用 PostgreSQL `xmax` 系统列：
  ```sql
  WITH upsert AS (
    INSERT INTO check_ins (...) VALUES (...)
    ON CONFLICT (plan_id, user_id, checked_date)
    DO UPDATE SET content = EXCLUDED.content, media = EXCLUDED.media, checked_at = NOW()
    RETURNING *, (xmax = 0) AS is_new
  ) SELECT * FROM upsert;
  ```
  `xmax = 0` 时为新插入，否则为更新。仅新插入时 ZINCRBY。
- 读排行：`ZREVRANGE plan:{id}:leaderboard 0 N WITHSCORES` 获取 top N 用户和打卡次数，默认返回 top 50，支持 `?limit=N`（最大 100）
- 连续天数：从 PostgreSQL 实时计算（复用现有 CurrentStreak）。注意：对于大量成员的计划，后期可缓存到 Redis
- 用户昵称/头像：从 PostgreSQL users 表查询补充
- CheckInService 直接依赖 LeaderboardService（构造函数注入），打卡后同步调用

### 公开广场

- 只查 `visibility = 'public'` 的记录和计划
- 按 `created_at DESC` 排序（MVP）
- 额外 JOIN 查询：作者信息、点赞数（子查询 COUNT）、评论数（子查询 COUNT）
- 当前用户是否已赞：子查询 EXISTS

## 权限校验流程

```
点赞/评论请求 → SocialService
  → 1. 根据 target_type 查询对应表（notes/plans/check_ins）
  → 2. 检查目标是否存在
  → 3. 检查 visibility == "public"
     - notes/plans: 直接检查自身 visibility
     - check_ins: 检查其所属 plan 的 visibility
  → 4. 通过后执行点赞/评论操作
```

## 项目结构变更

### 新增文件

```
internal/
├── model/
│   ├── like.go              # Like struct
│   └── comment.go           # Comment struct, CreateCommentParams, CommentWithUser
├── repo/
│   ├── like.go              # LikeRepo: Create, Delete, Exists, CountByTarget
│   ├── like_test.go
│   ├── comment.go           # CommentRepo: Create, ListByTarget, ListReplies, Delete, CountByTarget
│   ├── comment_test.go
│   ├── explore.go           # ExploreRepo: ListPublicNotes, ListPublicPlans
│   └── explore_test.go
├── service/
│   ├── social.go            # SocialService: Like, Unlike, Comment, DeleteComment, GetComments, GetReplies
│   └── leaderboard.go       # LeaderboardService: UpdateScore, GetLeaderboard
├── handler/
│   ├── social.go            # SocialHandler: 点赞+评论端点
│   └── explore.go           # ExploreHandler: 公开广场端点
├── middleware/
│   └── optional_auth.go     # OptionalAuth: 有token解析，没有也放行
migrations/
├── 005_create_likes.up.sql
├── 005_create_likes.down.sql
├── 006_create_comments.up.sql
└── 006_create_comments.down.sql
```

### 修改文件

- `internal/handler/router.go` — 添加 social、explore、leaderboard 路由
- `internal/handler/plan.go` — 添加 Leaderboard handler 方法
- `cmd/server/main.go` — 初始化 Redis 客户端，wire 新的 repo/service/handler
- `internal/service/checkin.go` — 打卡成功后调用 LeaderboardService 更新 Redis 排行榜

### Redis 集成

- 依赖：`github.com/redis/go-redis/v9`
- 在 `cmd/server/main.go` 中初始化 `redis.Client`，传入 `LeaderboardService`
- LeaderboardService 持有 `*redis.Client`，封装 ZINCRBY / ZREVRANGE 操作
