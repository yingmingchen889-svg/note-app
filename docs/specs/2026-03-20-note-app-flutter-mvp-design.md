# Note App Flutter MVP — 最小可用前端设计文档

## 概述

基于已完成的 P0+P1 后端，构建 Flutter 最小可用前端：注册登录 + 记录列表（触底加载）+ 创建记录。目标是验证前后端联通，为后续功能扩展打基础。

## 技术选型

| 层面 | 技术 | 说明 |
|------|------|------|
| 框架 | Flutter 3.x + Dart | 跨平台（iOS + Android） |
| 状态管理 | Riverpod | 轻量、类型安全，社区推荐 |
| HTTP | Dio | 功能丰富，支持拦截器自动注入 JWT |
| 本地存储 | SharedPreferences | 存储 JWT token |
| 路由 | GoRouter | 声明式路由，支持认证重定向 |

## 项目位置

独立仓库：`D:/github/note-app-flutter/`

## 页面设计（共 4 个）

### 1. 登录页 (LoginPage)

- 邮箱输入框 + 密码输入框
- "登录"按钮
- 底部"没有账号？去注册"文字链接，跳转注册页
- 登录失败显示错误提示（SnackBar）
- 登录成功后存储 token，跳转记录列表

### 2. 注册页 (RegisterPage)

- 邮箱 + 密码 + 昵称输入框
- "注册"按钮
- 底部"已有账号？去登录"文字链接
- 注册成功后自动登录（后端返回 token），跳转记录列表

### 3. 记录列表页 (NotesListPage)

- AppBar 标题 "我的记录"，右上角 "+" 按钮跳转创建页
- 列表展示记录卡片（标题 + 内容预览 + 创建时间）
- **触底加载**：滚动到底部自动加载下一页
- **无更多数据**：底部显示 "No more records" 提示文字
- **下拉刷新**：重置到第一页
- **空状态**：没有记录时显示引导文字
- AppBar 左侧菜单或右侧退出按钮，支持登出

### 4. 创建记录页 (NoteCreatePage)

- 标题输入框（必填）
- 内容输入框（多行，选填）
- AppBar 右侧"保存"按钮
- 保存成功后返回列表页并刷新
- 保存失败显示错误提示

## 项目结构

```
D:/github/note-app-flutter/
├── lib/
│   ├── main.dart                  # 入口，ProviderScope 包裹
│   ├── app.dart                   # GoRouter 路由配置 + 认证重定向
│   ├── core/
│   │   ├── api_client.dart        # Dio 封装：base URL、JWT 拦截器、错误处理
│   │   ├── storage.dart           # SharedPreferences 封装（存/取/删 token）
│   │   └── constants.dart         # API_BASE_URL 常量
│   ├── models/
│   │   ├── user.dart              # User 模型（fromJson）
│   │   └── note.dart              # Note 模型（fromJson）
│   ├── providers/
│   │   ├── auth_provider.dart     # AuthNotifier：login/register/logout/状态
│   │   └── notes_provider.dart    # NotesNotifier：分页加载/创建/刷新
│   └── pages/
│       ├── login_page.dart
│       ├── register_page.dart
│       ├── notes_list_page.dart
│       └── note_create_page.dart
├── pubspec.yaml
└── analysis_options.yaml
```

## 核心流程

### 启动流程

```
App 启动 → ProviderScope 初始化
  → GoRouter redirect 回调检查 AuthNotifier 状态
    → 有 token → /notes（记录列表）
    → 无 token → /login（登录页）
```

GoRouter 与 Riverpod 连接方式：AuthNotifier 继承 `ChangeNotifier`，GoRouter 的 `refreshListenable` 绑定到 AuthNotifier，当认证状态变化时自动触发路由重定向。

### 认证流程

```
登录/注册 → Dio POST /api/v1/auth/login 或 /register
  → 成功 → 存 token 到 SharedPreferences → AuthNotifier 状态更新 → GoRouter 重定向到 /notes
  → 失败 → 显示 SnackBar 错误信息
```

### 记录列表分页加载

```
进入列表页 → NotesNotifier 加载第 1 页（page=1, page_size=20）
  → 显示记录卡片列表

滚动到底部 → 检查 hasMore（当前已加载数 < total）
  → hasMore = true → 加载下一页，追加到列表
  → hasMore = false → 底部显示 "No more records"

下拉刷新 → 重置 page=1，清空列表，重新加载
```

### 创建记录

```
点击 "+" → 进入创建页
  → 填写标题(必填) + 内容(选填)
  → 点击"保存" → POST /api/v1/notes
    → 成功 → pop 回列表页，触发刷新
    → 失败 → 显示错误
```

## API 响应格式

### 认证响应

登录返回 **200**，注册返回 **201**，响应体相同：
```json
{
  "user": {
    "id": "uuid",
    "email": "test@test.com",
    "nickname": "tester",
    "avatar_url": null,
    "created_at": "2026-03-20T10:00:00Z",
    "updated_at": "2026-03-20T10:00:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

Dio 默认接受 200-299 状态码，201 不会抛异常，无需特殊处理。

### 记录列表响应（分页）

```json
{
  "data": [
    {
      "id": "uuid",
      "user_id": "uuid",
      "title": "我的笔记",
      "content": "内容...",
      "media": [],
      "tags": [],
      "visibility": "private",
      "is_draft": false,
      "created_at": "2026-03-20T10:00:00Z",
      "updated_at": "2026-03-20T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

### 错误响应

所有错误统一格式：
```json
{
  "code": "INVALID_INPUT",
  "message": "邮箱格式不正确"
}
```

Dio 错误处理时从 `response.data['message']` 提取用户可读信息显示在 SnackBar 中。

## Dart 模型定义

### User 模型

```dart
class User {
  final String id;
  final String email;
  final String nickname;
  final String? avatarUrl;
  final DateTime createdAt;

  User({required this.id, required this.email, required this.nickname, this.avatarUrl, required this.createdAt});

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'],
      email: json['email'],
      nickname: json['nickname'],
      avatarUrl: json['avatar_url'],
      createdAt: DateTime.parse(json['created_at']),
    );
  }
}
```

### Note 模型

```dart
class Note {
  final String id;
  final String title;
  final String content;
  final List<dynamic> tags;
  final String visibility;
  final bool isDraft;
  final DateTime createdAt;
  final DateTime updatedAt;

  Note({
    required this.id, required this.title, required this.content,
    required this.tags, required this.visibility, required this.isDraft,
    required this.createdAt, required this.updatedAt,
  });

  factory Note.fromJson(Map<String, dynamic> json) {
    return Note(
      id: json['id'],
      title: json['title'],
      content: json['content'] ?? '',
      tags: json['tags'] ?? [],
      visibility: json['visibility'] ?? 'private',
      isDraft: json['is_draft'] ?? false,
      createdAt: DateTime.parse(json['created_at']),
      updatedAt: DateTime.parse(json['updated_at']),
    );
  }
}
```

## 表单验证规则

与后端一致：
- 邮箱：合法 email 格式
- 密码：最少 6 个字符
- 昵称：1-100 个字符
- 标题：必填，最大 500 字符

## API 连接

### Base URL 配置

`lib/core/constants.dart`:
```dart
// 开发环境
// 电脑浏览器/桌面：localhost
// Android 模拟器：10.0.2.2（模拟器内部映射到宿主机 localhost）
// iOS 模拟器：localhost
// 真机：电脑局域网 IP

const String apiBaseUrl = 'http://10.0.2.2:8080/api/v1';
```

### Dio JWT 拦截器

```
每个请求 → 拦截器检查 SharedPreferences 中的 token
  → 有 token → 在 Header 加 Authorization: Bearer <token>
  → 无 token → 不加

响应 401 → 清除本地 token → 跳转登录页
```

## 状态管理设计

### AuthNotifier (Riverpod)

```
状态：
  - isAuthenticated: bool
  - user: User?
  - isLoading: bool

方法：
  - login(email, password) → 调 API，存 token
  - register(email, password, nickname) → 调 API，存 token
  - logout() → 删 token，清状态
  - checkAuth() → 仅检查本地 token 是否存在（不调 API 验证），过期 token 由 Dio 401 拦截器处理
```

### NotesNotifier (Riverpod)

```
状态：
  - notes: List<Note>
  - page: int
  - total: int
  - isLoading: bool
  - hasMore: bool (computed: notes.length < total)

方法：
  - loadNotes() → GET /notes?page=1，重置列表
  - loadMore() → GET /notes?page=next，追加到列表
  - createNote(title, content) → POST /notes
```

## 依赖 (pubspec.yaml)

```yaml
dependencies:
  flutter:
    sdk: flutter
  flutter_riverpod: ^2.4.0
  dio: ^5.4.0
  go_router: ^13.0.0
  shared_preferences: ^2.2.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^4.0.0
```

## 注意事项

- Flutter 开发需要先安装 Flutter SDK（https://flutter.dev）
- 后端服务需要先启动（`cd D:/github/note-app && docker compose up -d && go run cmd/server/main.go`）
- Android 模拟器需要 Android Studio 或独立 Android SDK
- 首次运行：`flutter pub get && flutter run`
