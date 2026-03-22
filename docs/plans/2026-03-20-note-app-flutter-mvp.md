# Note App Flutter MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a minimal Flutter frontend (4 pages) that connects to the existing Go backend — login, register, notes list with infinite scroll, and create note.

**Architecture:** Flutter app using Riverpod for state management, Dio for HTTP with JWT interceptor, GoRouter for declarative routing with auth redirect. SharedPreferences for token persistence. Target: Chrome web for development.

**Tech Stack:** Flutter 3.41.5, Dart 3.11.3, flutter_riverpod, dio, go_router, shared_preferences

**Spec:** `docs/superpowers/specs/2026-03-20-note-app-flutter-mvp-design.md`

---

## File Structure

```
D:/github/note-app-flutter/
├── lib/
│   ├── main.dart                  # Entry point, ProviderScope
│   ├── app.dart                   # GoRouter config + auth redirect
│   ├── core/
│   │   ├── api_client.dart        # Dio singleton, JWT interceptor, error extraction
│   │   ├── storage.dart           # Token storage (SharedPreferences)
│   │   └── constants.dart         # API base URL
│   ├── models/
│   │   ├── user.dart              # User model
│   │   └── note.dart              # Note model
│   ├── providers/
│   │   ├── auth_provider.dart     # AuthNotifier (ChangeNotifier + Riverpod)
│   │   └── notes_provider.dart    # NotesNotifier (pagination state)
│   └── pages/
│       ├── login_page.dart        # Login form
│       ├── register_page.dart     # Register form
│       ├── notes_list_page.dart   # Infinite scroll list
│       └── note_create_page.dart  # Create note form
├── pubspec.yaml
└── analysis_options.yaml
```

---

## Task 1: Project Setup + Dependencies

**Files:**
- Create: `D:/github/note-app-flutter/` (Flutter project)
- Modify: `pubspec.yaml` (add dependencies)

- [ ] **Step 1: Create Flutter project**

```bash
cd D:/github && flutter create note-app-flutter
cd note-app-flutter && git init
```

- [ ] **Step 2: Replace pubspec.yaml dependencies**

Replace the `dependencies` and `dev_dependencies` sections in `pubspec.yaml`:

```yaml
dependencies:
  flutter:
    sdk: flutter
  flutter_riverpod: ^2.6.1
  dio: ^5.7.0
  go_router: ^14.8.1
  shared_preferences: ^2.3.4

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^6.0.0
```

Note: Use latest stable versions compatible with Flutter 3.41.5 / Dart 3.11.3. The exact versions above may need adjustment — `flutter pub get` will resolve them.

- [ ] **Step 3: Install dependencies**

```bash
cd D:/github/note-app-flutter && flutter pub get
```
Expected: no errors

- [ ] **Step 4: Verify project runs**

```bash
cd D:/github/note-app-flutter && flutter run -d chrome
```
Expected: default Flutter counter app opens in Chrome

- [ ] **Step 5: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: initial Flutter project with dependencies"
```

---

## Task 2: Core Layer (Constants + Storage + API Client)

**Files:**
- Create: `D:/github/note-app-flutter/lib/core/constants.dart`
- Create: `D:/github/note-app-flutter/lib/core/storage.dart`
- Create: `D:/github/note-app-flutter/lib/core/api_client.dart`

- [ ] **Step 1: Create constants**

`lib/core/constants.dart`:
```dart
// For Chrome web development, use localhost directly.
// For Android emulator, change to 'http://10.0.2.2:8080/api/v1'.
const String apiBaseUrl = 'http://localhost:8080/api/v1';
const int defaultPageSize = 20;
```

- [ ] **Step 2: Create token storage**

`lib/core/storage.dart`:
```dart
import 'package:shared_preferences/shared_preferences.dart';

class TokenStorage {
  static const _tokenKey = 'jwt_token';

  static Future<void> saveToken(String token) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_tokenKey, token);
  }

  static Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_tokenKey);
  }

  static Future<void> deleteToken() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_tokenKey);
  }
}
```

- [ ] **Step 3: Create API client with JWT interceptor**

`lib/core/api_client.dart`:
```dart
import 'package:dio/dio.dart';
import 'constants.dart';
import 'storage.dart';

class ApiClient {
  static final Dio _dio = Dio(BaseOptions(
    baseUrl: apiBaseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 10),
    headers: {'Content-Type': 'application/json'},
  ))
    ..interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await TokenStorage.getToken();
        if (token != null) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        // Auto-logout on 401 (expired or invalid token)
        if (error.response?.statusCode == 401) {
          await TokenStorage.deleteToken();
        }
        handler.next(error);
      },
    ));

  static Dio get dio => _dio;

  /// Extracts user-readable error message from backend error response.
  /// Backend format: {"code": "...", "message": "..."}
  static String extractErrorMessage(DioException error) {
    if (error.response?.data is Map) {
      final message = error.response!.data['message'];
      if (message is String && message.isNotEmpty) {
        return message;
      }
    }
    if (error.type == DioExceptionType.connectionTimeout ||
        error.type == DioExceptionType.receiveTimeout) {
      return 'Connection timed out. Please try again.';
    }
    if (error.type == DioExceptionType.connectionError) {
      return 'Cannot connect to server. Is the backend running?';
    }
    return 'Something went wrong. Please try again.';
  }
}
```

- [ ] **Step 4: Verify build**

```bash
cd D:/github/note-app-flutter && flutter analyze
```
Expected: no errors

- [ ] **Step 5: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: core layer — constants, token storage, Dio API client with JWT"
```

---

## Task 3: Models (User + Note)

**Files:**
- Create: `D:/github/note-app-flutter/lib/models/user.dart`
- Create: `D:/github/note-app-flutter/lib/models/note.dart`

- [ ] **Step 1: Create User model**

`lib/models/user.dart`:
```dart
class User {
  final String id;
  final String email;
  final String nickname;
  final String? avatarUrl;
  final DateTime createdAt;

  User({
    required this.id,
    required this.email,
    required this.nickname,
    this.avatarUrl,
    required this.createdAt,
  });

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

- [ ] **Step 2: Create Note model**

`lib/models/note.dart`:
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
    required this.id,
    required this.title,
    required this.content,
    required this.tags,
    required this.visibility,
    required this.isDraft,
    required this.createdAt,
    required this.updatedAt,
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

- [ ] **Step 3: Verify build**

```bash
cd D:/github/note-app-flutter && flutter analyze
```

- [ ] **Step 4: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: User and Note data models with fromJson"
```

---

## Task 4: Auth Provider + Router

**Files:**
- Create: `D:/github/note-app-flutter/lib/providers/auth_provider.dart`
- Create: `D:/github/note-app-flutter/lib/app.dart`
- Modify: `D:/github/note-app-flutter/lib/main.dart`

- [ ] **Step 1: Create AuthNotifier**

`lib/providers/auth_provider.dart`:
```dart
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../core/api_client.dart';
import '../core/storage.dart';
import '../models/user.dart';

class AuthState {
  final bool isAuthenticated;
  final User? user;
  final bool isLoading;
  final String? error;

  AuthState({
    this.isAuthenticated = false,
    this.user,
    this.isLoading = false,
    this.error,
  });

  AuthState copyWith({
    bool? isAuthenticated,
    User? user,
    bool? isLoading,
    String? error,
  }) {
    return AuthState(
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      user: user ?? this.user,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

class AuthNotifier extends ChangeNotifier {
  AuthState _state = AuthState();
  AuthState get state => _state;

  Future<void> checkAuth() async {
    final token = await TokenStorage.getToken();
    _state = AuthState(isAuthenticated: token != null);
    notifyListeners();
  }

  Future<bool> login(String email, String password) async {
    _state = _state.copyWith(isLoading: true, error: null);
    notifyListeners();

    try {
      final response = await ApiClient.dio.post('/auth/login', data: {
        'email': email,
        'password': password,
      });

      await _handleAuthSuccess(response.data);
      return true;
    } on DioException catch (e) {
      _state = _state.copyWith(
        isLoading: false,
        error: ApiClient.extractErrorMessage(e),
      );
      notifyListeners();
      return false;
    }
  }

  Future<bool> register(String email, String password, String nickname) async {
    _state = _state.copyWith(isLoading: true, error: null);
    notifyListeners();

    try {
      final response = await ApiClient.dio.post('/auth/register', data: {
        'email': email,
        'password': password,
        'nickname': nickname,
      });

      await _handleAuthSuccess(response.data);
      return true;
    } on DioException catch (e) {
      _state = _state.copyWith(
        isLoading: false,
        error: ApiClient.extractErrorMessage(e),
      );
      notifyListeners();
      return false;
    }
  }

  Future<void> logout() async {
    await TokenStorage.deleteToken();
    _state = AuthState();
    notifyListeners();
  }

  Future<void> _handleAuthSuccess(Map<String, dynamic> data) async {
    final token = data['token'] as String;
    final user = User.fromJson(data['user']);
    await TokenStorage.saveToken(token);
    _state = AuthState(isAuthenticated: true, user: user);
    notifyListeners();
  }
}

final authProvider = ChangeNotifierProvider<AuthNotifier>((ref) {
  return AuthNotifier();
});
```

- [ ] **Step 2: Create app.dart with GoRouter**

`lib/app.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'providers/auth_provider.dart';
import 'pages/login_page.dart';
import 'pages/register_page.dart';
import 'pages/notes_list_page.dart';
import 'pages/note_create_page.dart';

// Placeholder pages — will be replaced in subsequent tasks
// For now, these let the router compile and run.

final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ref.read(authProvider);

  return GoRouter(
    initialLocation: '/login',
    refreshListenable: authNotifier,
    redirect: (context, state) {
      final isAuthenticated = authNotifier.state.isAuthenticated;
      final isAuthRoute =
          state.matchedLocation == '/login' || state.matchedLocation == '/register';

      if (!isAuthenticated && !isAuthRoute) {
        return '/login';
      }
      if (isAuthenticated && isAuthRoute) {
        return '/notes';
      }
      return null;
    },
    routes: [
      GoRoute(path: '/login', builder: (context, state) => const LoginPage()),
      GoRoute(path: '/register', builder: (context, state) => const RegisterPage()),
      GoRoute(path: '/notes', builder: (context, state) => const NotesListPage()),
      GoRoute(path: '/notes/create', builder: (context, state) => const NoteCreatePage()),
    ],
  );
});

class NoteApp extends ConsumerWidget {
  const NoteApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(routerProvider);
    return MaterialApp.router(
      title: 'Note App',
      theme: ThemeData(
        colorSchemeSeed: Colors.blue,
        useMaterial3: true,
      ),
      routerConfig: router,
    );
  }
}
```

- [ ] **Step 3: Create placeholder pages**

Create minimal placeholder pages so the router compiles. These will be fully implemented in Tasks 5-7.

`lib/pages/login_page.dart`:
```dart
import 'package:flutter/material.dart';

class LoginPage extends StatelessWidget {
  const LoginPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: Text('Login — coming soon')));
  }
}
```

`lib/pages/register_page.dart`:
```dart
import 'package:flutter/material.dart';

class RegisterPage extends StatelessWidget {
  const RegisterPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: Text('Register — coming soon')));
  }
}
```

`lib/pages/notes_list_page.dart`:
```dart
import 'package:flutter/material.dart';

class NotesListPage extends StatelessWidget {
  const NotesListPage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: Text('Notes — coming soon')));
  }
}
```

`lib/pages/note_create_page.dart`:
```dart
import 'package:flutter/material.dart';

class NoteCreatePage extends StatelessWidget {
  const NoteCreatePage({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: Text('Create — coming soon')));
  }
}
```

- [ ] **Step 4: Update main.dart**

Replace `lib/main.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'app.dart';
import 'providers/auth_provider.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Check if user has a saved token
  final container = ProviderContainer();
  await container.read(authProvider).checkAuth();

  runApp(
    UncontrolledProviderScope(
      container: container,
      child: const NoteApp(),
    ),
  );
}
```

- [ ] **Step 5: Delete unused test and counter files**

```bash
cd D:/github/note-app-flutter
rm -f test/widget_test.dart
```

- [ ] **Step 6: Run in Chrome to verify routing**

```bash
cd D:/github/note-app-flutter && flutter run -d chrome
```
Expected: Login placeholder page appears (since no token stored)

- [ ] **Step 7: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: auth provider, GoRouter with auth redirect, placeholder pages"
```

---

## Task 5: Login + Register Pages

**Files:**
- Modify: `D:/github/note-app-flutter/lib/pages/login_page.dart`
- Modify: `D:/github/note-app-flutter/lib/pages/register_page.dart`

- [ ] **Step 1: Implement login page**

Replace `lib/pages/login_page.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    if (!_formKey.currentState!.validate()) return;

    final success = await ref.read(authProvider).login(
          _emailController.text.trim(),
          _passwordController.text,
        );

    if (!success && mounted) {
      final error = ref.read(authProvider).state.error;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(error ?? 'Login failed')),
      );
    }
    // On success, GoRouter redirect handles navigation automatically
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider).state;

    return Scaffold(
      appBar: AppBar(title: const Text('Login')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextFormField(
                    controller: _emailController,
                    decoration: const InputDecoration(
                      labelText: 'Email',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.emailAddress,
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return 'Email is required';
                      if (!v.contains('@')) return 'Invalid email format';
                      return null;
                    },
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _passwordController,
                    decoration: const InputDecoration(
                      labelText: 'Password',
                      border: OutlineInputBorder(),
                    ),
                    obscureText: true,
                    validator: (v) {
                      if (v == null || v.isEmpty) return 'Password is required';
                      if (v.length < 6) return 'Password must be at least 6 characters';
                      return null;
                    },
                  ),
                  const SizedBox(height: 24),
                  SizedBox(
                    width: double.infinity,
                    child: ElevatedButton(
                      onPressed: authState.isLoading ? null : _login,
                      child: authState.isLoading
                          ? const SizedBox(
                              height: 20,
                              width: 20,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('Login'),
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextButton(
                    onPressed: () => context.go('/register'),
                    child: const Text("Don't have an account? Register"),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 2: Implement register page**

Replace `lib/pages/register_page.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/auth_provider.dart';

class RegisterPage extends ConsumerStatefulWidget {
  const RegisterPage({super.key});

  @override
  ConsumerState<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends ConsumerState<RegisterPage> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _nicknameController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    _nicknameController.dispose();
    super.dispose();
  }

  Future<void> _register() async {
    if (!_formKey.currentState!.validate()) return;

    final success = await ref.read(authProvider).register(
          _emailController.text.trim(),
          _passwordController.text,
          _nicknameController.text.trim(),
        );

    if (!success && mounted) {
      final error = ref.read(authProvider).state.error;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(error ?? 'Registration failed')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider).state;

    return Scaffold(
      appBar: AppBar(title: const Text('Register')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 400),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextFormField(
                    controller: _emailController,
                    decoration: const InputDecoration(
                      labelText: 'Email',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.emailAddress,
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return 'Email is required';
                      if (!v.contains('@')) return 'Invalid email format';
                      return null;
                    },
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _passwordController,
                    decoration: const InputDecoration(
                      labelText: 'Password',
                      border: OutlineInputBorder(),
                    ),
                    obscureText: true,
                    validator: (v) {
                      if (v == null || v.isEmpty) return 'Password is required';
                      if (v.length < 6) return 'At least 6 characters';
                      return null;
                    },
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _nicknameController,
                    decoration: const InputDecoration(
                      labelText: 'Nickname',
                      border: OutlineInputBorder(),
                    ),
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return 'Nickname is required';
                      if (v.trim().length > 100) return 'Max 100 characters';
                      return null;
                    },
                  ),
                  const SizedBox(height: 24),
                  SizedBox(
                    width: double.infinity,
                    child: ElevatedButton(
                      onPressed: authState.isLoading ? null : _register,
                      child: authState.isLoading
                          ? const SizedBox(
                              height: 20,
                              width: 20,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('Register'),
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextButton(
                    onPressed: () => context.go('/login'),
                    child: const Text('Already have an account? Login'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 3: Run in Chrome**

```bash
cd D:/github/note-app-flutter && flutter run -d chrome
```
Expected: Login page with email/password form. "Register" link navigates to register page.

- [ ] **Step 4: Test with backend**

Start the backend if not running:
```bash
cd D:/github/note-app && docker compose up -d && go run cmd/server/main.go &
```

In Chrome:
1. Click "Register" → fill in email/password/nickname → click Register
2. Should redirect to Notes placeholder page
3. Refresh browser → should stay on Notes page (token persisted)

- [ ] **Step 5: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: login and register pages with form validation"
```

---

## Task 6: Notes Provider (Pagination State)

**Files:**
- Create: `D:/github/note-app-flutter/lib/providers/notes_provider.dart`

- [ ] **Step 1: Create NotesNotifier**

`lib/providers/notes_provider.dart`:
```dart
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../core/api_client.dart';
import '../core/constants.dart';
import '../models/note.dart';

class NotesState {
  final List<Note> notes;
  final int page;
  final int total;
  final bool isLoading;
  final bool isLoadingMore;
  final String? error;

  NotesState({
    this.notes = const [],
    this.page = 0,
    this.total = 0,
    this.isLoading = false,
    this.isLoadingMore = false,
    this.error,
  });

  bool get hasMore => notes.length < total;
  bool get isEmpty => !isLoading && notes.isEmpty;

  NotesState copyWith({
    List<Note>? notes,
    int? page,
    int? total,
    bool? isLoading,
    bool? isLoadingMore,
    String? error,
  }) {
    return NotesState(
      notes: notes ?? this.notes,
      page: page ?? this.page,
      total: total ?? this.total,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      error: error,
    );
  }
}

class NotesNotifier extends ChangeNotifier {
  NotesState _state = NotesState();
  NotesState get state => _state;

  /// Load first page (used on initial load and pull-to-refresh)
  Future<void> loadNotes() async {
    _state = NotesState(isLoading: true);
    notifyListeners();

    try {
      final response = await ApiClient.dio.get('/notes', queryParameters: {
        'page': 1,
        'page_size': defaultPageSize,
      });

      final data = response.data;
      final notes = (data['data'] as List?)
              ?.map((json) => Note.fromJson(json))
              .toList() ??
          [];

      _state = NotesState(
        notes: notes,
        page: 1,
        total: data['total'] ?? 0,
      );
      notifyListeners();
    } on DioException catch (e) {
      _state = NotesState(error: ApiClient.extractErrorMessage(e));
      notifyListeners();
    }
  }

  /// Load next page (triggered by scroll to bottom)
  Future<void> loadMore() async {
    if (_state.isLoadingMore || !_state.hasMore) return;

    _state = _state.copyWith(isLoadingMore: true);
    notifyListeners();

    try {
      final nextPage = _state.page + 1;
      final response = await ApiClient.dio.get('/notes', queryParameters: {
        'page': nextPage,
        'page_size': defaultPageSize,
      });

      final data = response.data;
      final newNotes = (data['data'] as List?)
              ?.map((json) => Note.fromJson(json))
              .toList() ??
          [];

      _state = _state.copyWith(
        notes: [..._state.notes, ...newNotes],
        page: nextPage,
        total: data['total'] ?? _state.total,
        isLoadingMore: false,
      );
      notifyListeners();
    } on DioException catch (e) {
      _state = _state.copyWith(
        isLoadingMore: false,
        error: ApiClient.extractErrorMessage(e),
      );
      notifyListeners();
    }
  }

  /// Create a new note
  Future<bool> createNote(String title, String content) async {
    try {
      await ApiClient.dio.post('/notes', data: {
        'title': title,
        'content': content,
      });
      return true;
    } on DioException {
      return false;
    }
  }
}

final notesProvider = ChangeNotifierProvider<NotesNotifier>((ref) {
  return NotesNotifier();
});
```

- [ ] **Step 2: Verify build**

```bash
cd D:/github/note-app-flutter && flutter analyze
```

- [ ] **Step 3: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: notes provider with pagination, load more, create"
```

---

## Task 7: Notes List Page (Infinite Scroll)

**Files:**
- Modify: `D:/github/note-app-flutter/lib/pages/notes_list_page.dart`

- [ ] **Step 1: Implement notes list with infinite scroll**

Replace `lib/pages/notes_list_page.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../models/note.dart';
import '../providers/auth_provider.dart';
import '../providers/notes_provider.dart';

class NotesListPage extends ConsumerStatefulWidget {
  const NotesListPage({super.key});

  @override
  ConsumerState<NotesListPage> createState() => _NotesListPageState();
}

class _NotesListPageState extends ConsumerState<NotesListPage> {
  final _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    // Load notes on first visit
    Future.microtask(() => ref.read(notesProvider).loadNotes());
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      ref.read(notesProvider).loadMore();
    }
  }

  Future<void> _refresh() async {
    await ref.read(notesProvider).loadNotes();
  }

  @override
  Widget build(BuildContext context) {
    final notesState = ref.watch(notesProvider).state;

    return Scaffold(
      appBar: AppBar(
        title: const Text('My Notes'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add),
            onPressed: () => context.push('/notes/create'),
          ),
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () => ref.read(authProvider).logout(),
          ),
        ],
      ),
      body: _buildBody(notesState),
    );
  }

  Widget _buildBody(NotesState notesState) {
    if (notesState.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (notesState.error != null && notesState.notes.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(notesState.error!),
            const SizedBox(height: 16),
            ElevatedButton(onPressed: _refresh, child: const Text('Retry')),
          ],
        ),
      );
    }

    if (notesState.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.note_alt_outlined, size: 64, color: Colors.grey),
            const SizedBox(height: 16),
            const Text('No notes yet. Tap + to create one!',
                style: TextStyle(color: Colors.grey)),
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: () => context.push('/notes/create'),
              icon: const Icon(Icons.add),
              label: const Text('Create Note'),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: _refresh,
      child: ListView.builder(
        controller: _scrollController,
        itemCount: notesState.notes.length + 1, // +1 for bottom indicator
        itemBuilder: (context, index) {
          if (index == notesState.notes.length) {
            return _buildBottomIndicator(notesState);
          }
          return _buildNoteCard(notesState.notes[index]);
        },
      ),
    );
  }

  Widget _buildNoteCard(Note note) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: ListTile(
        title: Text(
          note.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(fontWeight: FontWeight.w600),
        ),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (note.content.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  note.content,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(color: Colors.grey),
                ),
              ),
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                _formatDate(note.createdAt),
                style: const TextStyle(fontSize: 12, color: Colors.grey),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildBottomIndicator(NotesState notesState) {
    if (notesState.isLoadingMore) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    if (!notesState.hasMore) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Center(
          child: Text(
            'No more records',
            style: TextStyle(color: Colors.grey),
          ),
        ),
      );
    }
    return const SizedBox.shrink();
  }

  String _formatDate(DateTime date) {
    return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')} '
        '${date.hour.toString().padLeft(2, '0')}:${date.minute.toString().padLeft(2, '0')}';
  }
}
```

- [ ] **Step 2: Run in Chrome and test**

```bash
cd D:/github/note-app-flutter && flutter run -d chrome
```
Expected:
- After login, shows notes list (or empty state if no notes)
- Logout button works (returns to login)
- Pull-to-refresh works
- Empty state shows "No notes yet" with create button

- [ ] **Step 3: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: notes list page with infinite scroll and pull-to-refresh"
```

---

## Task 8: Create Note Page

**Files:**
- Modify: `D:/github/note-app-flutter/lib/pages/note_create_page.dart`

- [ ] **Step 1: Implement create note page**

Replace `lib/pages/note_create_page.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../providers/notes_provider.dart';

class NoteCreatePage extends ConsumerStatefulWidget {
  const NoteCreatePage({super.key});

  @override
  ConsumerState<NoteCreatePage> createState() => _NoteCreatePageState();
}

class _NoteCreatePageState extends ConsumerState<NoteCreatePage> {
  final _formKey = GlobalKey<FormState>();
  final _titleController = TextEditingController();
  final _contentController = TextEditingController();
  bool _isSaving = false;

  @override
  void dispose() {
    _titleController.dispose();
    _contentController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isSaving = true);

    final success = await ref.read(notesProvider).createNote(
          _titleController.text.trim(),
          _contentController.text.trim(),
        );

    if (!mounted) return;

    if (success) {
      // Refresh list and go back
      ref.read(notesProvider).loadNotes();
      context.pop();
    } else {
      setState(() => _isSaving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to save note. Please try again.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('New Note'),
        actions: [
          TextButton(
            onPressed: _isSaving ? null : _save,
            child: _isSaving
                ? const SizedBox(
                    height: 20,
                    width: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('Save'),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            children: [
              TextFormField(
                controller: _titleController,
                decoration: const InputDecoration(
                  labelText: 'Title',
                  border: OutlineInputBorder(),
                ),
                validator: (v) {
                  if (v == null || v.trim().isEmpty) return 'Title is required';
                  if (v.trim().length > 500) return 'Max 500 characters';
                  return null;
                },
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _contentController,
                decoration: const InputDecoration(
                  labelText: 'Content (optional)',
                  border: OutlineInputBorder(),
                  alignLabelWithHint: true,
                ),
                maxLines: 10,
                minLines: 5,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 2: Run in Chrome and full E2E test**

```bash
cd D:/github/note-app-flutter && flutter run -d chrome
```

Full test:
1. Register a new user → redirected to Notes list
2. Tap "+" → Create Note page
3. Enter title + content → tap "Save"
4. Redirected back to list → new note appears
5. Create more notes → verify infinite scroll (after 20+ notes)
6. Tap logout → back to login
7. Login again → notes still there

- [ ] **Step 3: Commit**

```bash
cd D:/github/note-app-flutter
git add -A && git commit -m "feat: create note page with form validation and save"
```

---

## Summary

After completing all 8 tasks, the Flutter MVP provides:

| Feature | Description |
|---------|-------------|
| Login | Email + password form with validation |
| Register | Email + password + nickname form |
| Auth redirect | GoRouter + Riverpod auto-redirect based on token |
| Notes list | Infinite scroll, pull-to-refresh, empty state |
| Create note | Title + content form with save |
| JWT persistence | Token saved to SharedPreferences, auto-attached by Dio |
| Error handling | Backend errors displayed as SnackBar messages |

Run with: `cd D:/github/note-app-flutter && flutter run -d chrome`
Backend: `cd D:/github/note-app && docker compose up -d && go run cmd/server/main.go`
