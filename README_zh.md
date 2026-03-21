# NoteApp 后端

一款个人成长记事应用后端，使用 Go 构建，支持记录、计划打卡、社交互动和成长报告。

## 技术栈

- **Go 1.25+** / Gin 框架
- **PostgreSQL 16** — 结构化数据 + JSONB
- **MinIO** — 文件存储（图片、视频、音频）
- **Redis** — 排行榜、缓存
- **Docker Compose** — 一键部署

## 功能

### P0 — 核心功能
- 用户认证（JWT，邮箱 + 密码）
- 记录 CRUD（标签、草稿、富文本/Delta JSON 内容）
- 计划 CRUD（起止日期、参与者管理、编辑/删除保护）
- 每日打卡（每个计划每天一次，支持覆盖更新）
- 文件上传（MinIO 预签名 URL 直传）
- 打卡日历和连续打卡天数计算

### P1 — 社交功能
- 点赞/取消点赞（记录、计划、打卡）
- 评论（支持二级回复）
- 公开广场（带社交计数）
- Redis 排行榜（每个计划的打卡排名）
- 加入公开计划
- 可选认证（匿名访问公开内容）

### P2 — 成长报告
- 成长报告生成（按月、季度、年度）
- 汇总统计：打卡完成率、连续天数、趋势、最活跃计划

## API 端点

```
# 认证
POST   /api/v1/auth/register          # 注册
POST   /api/v1/auth/login             # 登录

# 记录
GET    /api/v1/notes                   # 列表（支持标签筛选）
POST   /api/v1/notes                   # 创建
GET    /api/v1/notes/:id               # 详情
PUT    /api/v1/notes/:id               # 更新
DELETE /api/v1/notes/:id               # 删除
PUT    /api/v1/notes/:id/share         # 公开/私有切换

# 计划
GET    /api/v1/plans                   # 列表
POST   /api/v1/plans                   # 创建
GET    /api/v1/plans/:id               # 详情
PUT    /api/v1/plans/:id               # 更新
DELETE /api/v1/plans/:id               # 删除
PUT    /api/v1/plans/:id/share         # 公开/私有切换
POST   /api/v1/plans/:id/join          # 加入计划
GET    /api/v1/plans/:id/members       # 参与者列表
GET    /api/v1/plans/:id/leaderboard   # 排行榜

# 打卡
POST   /api/v1/plans/:id/checkins      # 打卡
GET    /api/v1/plans/:id/checkins      # 打卡列表
GET    /api/v1/checkins/calendar        # 日历视图

# 社交
POST   /api/v1/social/:type/:id/like          # 点赞
DELETE /api/v1/social/:type/:id/like          # 取消点赞
GET    /api/v1/social/:type/:id/comments      # 评论列表
POST   /api/v1/social/:type/:id/comments      # 发表评论
DELETE /api/v1/social/comments/:id            # 删除评论
GET    /api/v1/social/comments/:id/replies    # 回复列表

# 公开广场
GET    /api/v1/explore/notes           # 公开记录
GET    /api/v1/explore/plans           # 公开计划

# 成长报告
GET    /api/v1/growth/reports          # 报告列表
POST   /api/v1/growth/generate         # 生成报告

# 文件上传
POST   /api/v1/upload/presign          # 获取预签名 URL
POST   /api/v1/upload/confirm          # 确认上传
```

## 项目结构

```
note-app/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── middleware/              # JWT 认证、CORS、可选认证
│   ├── model/                  # 数据模型
│   ├── handler/                # HTTP 处理器
│   ├── service/                # 业务逻辑
│   ├── repo/                   # 数据库操作
│   └── storage/                # MinIO 客户端
├── migrations/                 # 数据库迁移文件 (001-007)
├── scripts/
│   └── init_db.sql             # 数据库初始化脚本
├── docker-compose.yaml
├── Dockerfile
└── Makefile
```

## 快速开始

### 方式一：Docker Compose（推荐）

一条命令启动全部服务，数据库自动初始化，无需手动操作。

```bash
# 克隆仓库
git clone https://github.com/yingmingchen889-svg/note-app.git
cd note-app

# 构建并启动所有服务
docker compose up -d --build

# 查看服务状态
docker compose ps
```

完成！服务运行在 `http://localhost:8080`。

- PostgreSQL 表结构通过 `scripts/init_db.sql` 自动初始化
- MinIO 存储桶自动创建并设置公开读权限
- App 服务等待所有依赖就绪后才启动

```bash
# 查看日志
docker compose logs -f app

# 停止所有服务
docker compose down

# 停止并清除数据（全新开始）
docker compose down -v
```

### 方式二：本地开发

适用于需要热重载的开发场景：

```bash
# 前置条件：Go 1.22+、Docker、golang-migrate CLI

# 仅启动基础设施
docker compose up -d postgres minio redis

# 运行数据库迁移
migrate -path migrations -database "postgres://noteapp:noteapp@localhost:5432/noteapp?sslmode=disable" up

# 启动服务
go run cmd/server/main.go
```

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| App | 8080 | API 服务 |
| PostgreSQL | 5432 | 数据库 |
| MinIO API | 9000 | 文件存储 |
| MinIO Console | 9001 | MinIO 管理界面 |
| Redis | 6379 | 缓存 |

## 相关项目

- [note-app-flutter](https://github.com/yingmingchen889-svg/note-app-flutter) — Flutter 前端

## 开源协议

MIT
