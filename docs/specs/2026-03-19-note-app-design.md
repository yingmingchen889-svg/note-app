# Note App - 记事应用设计文档

## 概述

一款面向个人成长的记事应用，核心功能包括生活记录、计划打卡、社交分享、成长历程可视化和短视频生成。采用先自用再推广的策略，MVP 聚焦核心记录与打卡功能。

## 技术选型

| 层面 | 技术 | 说明 |
|------|------|------|
| 前端 | Flutter 3.x + Dart | 跨平台（iOS + Android） |
| 后端 | Go 1.22+ / Gin 框架 | 单体架构，按 package 分模块 |
| 数据库 | PostgreSQL 16 | JSONB 存储灵活内容 |
| 文件存储 | MinIO | 图片/视频/音频对象存储 |
| 缓存 | Redis | 排行榜、热门内容缓存、计数器 |
| 认证 | JWT | MVP 阶段：邮箱+密码登录 |
| API 风格 | RESTful JSON | /api/v1/ 前缀 |
| 部署 | Docker Compose | PostgreSQL + MinIO + Redis + App |

## 架构

```
┌─────────────┐     HTTPS/JSON      ┌──────────────────┐
│  Flutter App │ ◄──────────────────► │   Go 单体服务     │
│ (iOS/Android)│                     │   (Gin框架)       │
└─────────────┘                      ├──────────────────┤
                                     │  模块:            │
                                     │  - 用户认证 (JWT) │
                                     │  - 记录模块       │
                                     │  - 计划/打卡模块  │
                                     │  - 社交模块       │
                                     │  - 成长历程模块   │
                                     │  - 视频生成模块   │
                                     └────────┬─────────┘
                                              │
                              ┌───────────────┼───────────────┐
                              ▼               ▼               ▼
                        ┌──────────┐   ┌──────────┐   ┌──────────┐
                        │PostgreSQL│   │  MinIO   │   │  Redis   │
                        │结构化数据 │   │文件存储   │   │缓存/排行 │
                        └──────────┘   └──────────┘   └──────────┘
```

## 数据模型

### 用户表 `users`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| phone | VARCHAR | 手机号（可空，后期短信登录） |
| email | VARCHAR | 邮箱 |
| password_hash | VARCHAR | bcrypt 加密密码 |
| nickname | VARCHAR | 昵称 |
| avatar_url | VARCHAR | 头像（MinIO 路径） |
| created_at | TIMESTAMP | 注册时间 |
| updated_at | TIMESTAMP | 更新时间（自动维护） |

后期扩展第三方登录时新增 `user_auths` 表：

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | UUID | 关联用户 |
| provider | VARCHAR | `wechat` / `qq` / `sms` |
| provider_uid | VARCHAR | 第三方唯一标识 |

### 记录表 `notes`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 作者 |
| title | VARCHAR | 标题 |
| content | TEXT | 正文（Markdown） |
| media | JSONB | 附件列表 `[{type, url, thumbnail}]` |
| tags | JSONB | 标签列表 `["生活", "想法"]`，用户自定义 |
| visibility | ENUM | `private` / `public` |
| is_draft | BOOLEAN | 是否为草稿，默认 false |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 计划表 `plans`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 创建者 |
| title | VARCHAR | 如"每日练字" |
| description | TEXT | 计划说明 |
| visibility | ENUM | `private` / `public` |
| start_date | DATE | 开始日期 |
| end_date | DATE | 结束日期（可空，长期计划） |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### 计划参与者表 `plan_members`

| 字段 | 类型 | 说明 |
|------|------|------|
| plan_id | UUID | 计划 |
| user_id | UUID | 参与者 |
| role | ENUM | `owner` / `member` |
| joined_at | TIMESTAMP | |

### 打卡记录表 `check_ins`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| plan_id | UUID | 所属计划 |
| user_id | UUID | 打卡人 |
| content | TEXT | 打卡文字 |
| media | JSONB | 附件 `[{type, url}]` |
| checked_date | DATE | 打卡日期 |
| checked_at | TIMESTAMP | 打卡时间 |

**唯一约束**：`UNIQUE(plan_id, user_id, checked_date)` — 每个计划每人每天只能打卡一次。重复打卡采用 UPSERT 策略（`ON CONFLICT ... DO UPDATE`），覆盖之前的内容。

### 社交互动表 `interactions`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 操作人 |
| target_type | ENUM | `note` / `plan` / `check_in` |
| target_id | UUID | 目标 ID |
| type | ENUM | `like` / `comment` |
| content | TEXT | 评论内容（点赞为空） |
| created_at | TIMESTAMP | |

### 成长报告表 `growth_reports`

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | |
| period_type | ENUM | `monthly` / `quarterly` / `yearly` |
| period_start | DATE | 周期起始 |
| stats | JSONB | 统计数据（打卡次数、完成率等） |
| generated_at | TIMESTAMP | |

### Redis 用途

- **排行榜**：`plan:{id}:leaderboard` — Sorted Set，按打卡次数排名
- **缓存**：热门公开记录、用户 profile
- **计数器**：点赞数、评论数

## 模块功能设计

### 模块一：记录

**核心流程：**
```
创建记录 → 编辑内容(文字/Markdown) → 上传附件(图片/视频/音频) → 保存(默认私有)
```

**功能点：**
- 富文本编辑，支持 Markdown
- 多媒体附件上传（图片压缩+缩略图、视频转码、音频）
- 标签分类（生活、想法、读书笔记等），用户自定义标签
- 记录列表支持按时间/标签筛选
- 草稿自动保存

**附件上传流程：**
```
App → POST /upload/presign 获取预签名URL → App 直传 MinIO → POST /upload/confirm 通知后端记录元数据
```

### 模块二：计划与打卡

**核心流程：**
```
创建计划(标题/描述/起止日期) → 每日打卡(选择计划→填写内容→上传附件) → 查看打卡日历
```

**功能点：**
- 一个用户可以有多个进行中的计划
- 每个计划每天可打卡一次（同一天重复打卡覆盖）
- 打卡时附带文字 + 图片/视频/音频
- 日历视图展示打卡情况（类似 GitHub 贡献热力图）
- 连续打卡天数统计、当前连击数

### 模块三：分享与社交

**记录分享：**
```
私有记录 → 点击"分享" → visibility 改为 public → 出现在公开广场
```

**计划分享：**
```
私有计划 → 分享 → 其他用户可"加入计划" → 成为 plan_member → 一起打卡
```

**社交功能：**
- 公开广场：浏览公开的记录和计划
- 点赞、评论
- 计划参与者列表 + 打卡排行榜（Redis Sorted Set）
- 排行维度：打卡总次数、连续天数

### 模块四：成长历程

**生成时机：**
- 定时任务：每月1号生成上月报告，每季度/年初生成
- 用户也可手动触发生成

**报告内容（MVP 纯数据可视化）：**
- 各计划打卡完成率（柱状图）
- 打卡热力图（全部计划合并视图）
- 连续打卡最长记录
- 记录数量趋势（折线图）
- 最活跃的计划 TOP 3

**前端展示：**
- Flutter 用 `fl_chart` 库渲染图表
- 报告可保存为图片分享

### 模块五：视频生成

**MVP 方案（生成到本地）：**
```
选择内容(记录/计划/成长报告) → 选择模板 → 后端合成视频 → 下载到手机相册
```

**实现思路：**
- 后端用 FFmpeg 合成视频（Go 调用 FFmpeg CLI）
- 预设 2-3 个视频模板（图文轮播、成长报告动画、打卡集锦）
- 异步生成：用户提交请求 → goroutine 后台处理 → 客户端轮询 `GET /videos/:id/status` 查询进度
- 生成的视频存 MinIO，客户端下载
- **FFmpeg 依赖**：Dockerfile 需安装 FFmpeg（LGPL 构建，避免 GPL 污染），视频生成功能在 P3 阶段，可使用多阶段构建或独立 worker 镜像以保持主镜像精简

## API 端点

```
# 认证
POST   /api/v1/auth/register
POST   /api/v1/auth/login

# 记录
GET    /api/v1/notes              # 列表（支持标签筛选）
POST   /api/v1/notes
GET    /api/v1/notes/:id
PUT    /api/v1/notes/:id
DELETE /api/v1/notes/:id
PUT    /api/v1/notes/:id/share    # 切换公开/私有

# 计划
GET    /api/v1/plans
POST   /api/v1/plans
GET    /api/v1/plans/:id
PUT    /api/v1/plans/:id
PUT    /api/v1/plans/:id/share
POST   /api/v1/plans/:id/join     # 加入计划
GET    /api/v1/plans/:id/members
GET    /api/v1/plans/:id/leaderboard

# 打卡
POST   /api/v1/plans/:id/checkins
GET    /api/v1/plans/:id/checkins  # 某计划的打卡列表
GET    /api/v1/checkins/calendar   # 日历视图（所有计划）

# 社交（:target_type = notes | plans | checkins）
POST   /api/v1/social/:target_type/:id/like     # 点赞
DELETE /api/v1/social/:target_type/:id/like     # 取消点赞
GET    /api/v1/social/:target_type/:id/comments
POST   /api/v1/social/:target_type/:id/comments

# 公开广场
GET    /api/v1/explore/notes       # 公开记录
GET    /api/v1/explore/plans       # 公开计划

# 成长历程
GET    /api/v1/growth/reports      # 报告列表
POST   /api/v1/growth/generate     # 手动触发生成，body: {"period_type":"monthly","period_start":"2026-02-01"}，已存在则覆盖

# 视频
POST   /api/v1/videos/generate     # 提交生成请求
GET    /api/v1/videos/:id/status   # 查询进度
GET    /api/v1/videos/:id/download # 下载

# 文件上传
POST   /api/v1/upload/presign      # 获取MinIO预签名URL
POST   /api/v1/upload/confirm      # 客户端上传完成后确认，记录文件元数据
```

### 分页约定

所有列表接口支持分页参数：`?page=1&page_size=20`（默认 page=1, page_size=20, 最大 100）

响应格式：
```json
{
  "data": [...],
  "total": 150,
  "page": 1,
  "page_size": 20
}
```

### 错误响应格式

```json
{
  "code": "INVALID_INPUT",
  "message": "标题不能为空"
}
```

常用错误码：
- `INVALID_INPUT` — 请求参数校验失败 (400)
- `UNAUTHORIZED` — 未登录或 JWT 过期 (401)
- `FORBIDDEN` — 无权操作 (403)
- `NOT_FOUND` — 资源不存在 (404)
- `CONFLICT` — 资源冲突，如重复注册 (409)
- `INTERNAL_ERROR` — 服务端错误 (500)

## Go 项目结构

```
note-app/
├── cmd/
│   └── server/
│       └── main.go              # 入口
├── internal/
│   ├── config/                  # 配置加载
│   ├── middleware/               # JWT认证、CORS、日志
│   ├── model/                   # 数据模型（对应数据库表）
│   ├── handler/                 # HTTP handler（按模块分文件）
│   │   ├── auth.go
│   │   ├── note.go
│   │   ├── plan.go
│   │   ├── checkin.go
│   │   ├── social.go
│   │   ├── growth.go
│   │   └── video.go
│   ├── service/                 # 业务逻辑层
│   ├── repo/                    # 数据库访问层
│   └── storage/                 # MinIO 文件操作封装
├── pkg/
│   └── utils/                   # 通用工具（JWT生成、密码加密等）
├── migrations/                  # 数据库迁移文件
├── docker-compose.yaml          # PostgreSQL + MinIO + Redis
├── Dockerfile
├── go.mod
└── go.sum
```

## 开发分期

| 阶段 | 内容 | 目标 |
|------|------|------|
| **P0** | 认证 + 记录 + 计划打卡 | 核心可用，自己先用起来 |
| **P1** | 分享 + 社交（点赞评论排行） | 支持多人使用 |
| **P2** | 成长历程（数据可视化） | 增加留存价值 |
| **P3** | 视频生成到本地 | 差异化功能 |
| **P4** | 短信登录 + 第三方登录 + AI 成长总结 | 推广阶段增强 |
