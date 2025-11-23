# PR Reviewer Assignment Service

Микросервис для автоматического назначения ревьюеров на Pull Request'ы.

## Описание

Сервис автоматически назначает до 2 ревьюеров на PR из команды автора, позволяет управлять командами, пользователями и выполнять переназначение ревьюеров.

## Технологический стек

- **Язык**: Go 1.21
- **База данных**: PostgreSQL 15
- **HTTP Router**: gorilla/mux
- **Контейнеризация**: Docker, Docker Compose
- **Тестирование**: Go testing, testify

## Быстрый старт

> **Для пользователей Windows**: См. [WINDOWS_GUIDE.md](WINDOWS_GUIDE.md) для примеров с PowerShell

### Запуск через Docker Compose (рекомендуется)

```bash
docker-compose up --build
```

Сервис будет доступен на `http://localhost:8080`

### Остановка

```bash
docker-compose down -v
```

### Запуск локально

1. Установите PostgreSQL и создайте базу данных:
```sql
CREATE DATABASE pr_reviewer_db;
CREATE USER pruser WITH PASSWORD 'prpassword';
GRANT ALL PRIVILEGES ON DATABASE pr_reviewer_db TO pruser;
```

2. Установите зависимости:
```bash
go mod download
```

3. Запустите сервис:
```bash
export DATABASE_URL="postgres://pruser:prpassword@localhost:5432/pr_reviewer_db?sslmode=disable"
go run ./cmd/server
```

## Makefile команды

```bash
make help        # Показать справку
make build       # Собрать приложение
make run         # Запустить локально
make test        # Запустить все тесты
make test-unit   # Запустить unit тесты
make test-e2e    # Запустить e2e тесты
make docker-up   # Запустить через docker-compose
make docker-down # Остановить docker-compose
make clean       # Очистить артефакты сборки
```

## API Endpoints

### Teams

#### POST /team/add
Создать команду с участниками (создаёт/обновляет пользователей)

**Request:**
```json
{
  "team_name": "backend",
  "members": [
    {"user_id": "u1", "username": "Alice", "is_active": true},
    {"user_id": "u2", "username": "Bob", "is_active": true}
  ]
}
```

**Response (201):**
```json
{
  "team": {
    "team_name": "backend",
    "members": [...]
  }
}
```

#### GET /team/get?team_name=backend
Получить команду с участниками

**Response (200):**
```json
{
  "team_name": "backend",
  "members": [...]
}
```

### Users

#### POST /users/setIsActive
Установить флаг активности пользователя

**Request:**
```json
{
  "user_id": "u2",
  "is_active": false
}
```

**Response (200):**
```json
{
  "user": {
    "user_id": "u2",
    "username": "Bob",
    "team_name": "backend",
    "is_active": false
  }
}
```

#### GET /users/getReview?user_id=u2
Получить PR'ы, где пользователь назначен ревьювером

**Response (200):**
```json
{
  "user_id": "u2",
  "pull_requests": [
    {
      "pull_request_id": "pr-1001",
      "pull_request_name": "Add search",
      "author_id": "u1",
      "status": "OPEN"
    }
  ]
}
```

### Pull Requests

#### POST /pullRequest/create
Создать PR и автоматически назначить до 2 ревьюверов

**Request:**
```json
{
  "pull_request_id": "pr-1001",
  "pull_request_name": "Add search",
  "author_id": "u1"
}
```

**Response (201):**
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search",
    "author_id": "u1",
    "status": "OPEN",
    "assigned_reviewers": ["u2", "u3"],
    "createdAt": "2025-11-23T10:00:00Z"
  }
}
```

#### POST /pullRequest/merge
Пометить PR как MERGED (идемпотентная операция)

**Request:**
```json
{
  "pull_request_id": "pr-1001"
}
```

**Response (200):**
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "status": "MERGED",
    "mergedAt": "2025-11-23T10:30:00Z",
    ...
  }
}
```

#### POST /pullRequest/reassign
Переназначить ревьювера на другого из его команды

**Request:**
```json
{
  "pull_request_id": "pr-1001",
  "old_user_id": "u2"
}
```

**Response (200):**
```json
{
  "pr": {
    "pull_request_id": "pr-1001",
    "assigned_reviewers": ["u3", "u5"],
    ...
  },
  "replaced_by": "u5"
}
```

### Health

#### GET /health
Проверка состояния сервиса

**Response (200):**
```json
{
  "status": "ok"
}
```


## Бизнес-правила

### Назначение ревьюверов при создании PR

1. Автоматически назначаются **до 2** активных ревьюверов из команды автора
2. Автор PR **не может быть** ревьювером своего PR
3. Выбор ревьюверов происходит **случайно** из доступных кандидатов
4. Если активных участников меньше 2, назначается доступное количество (0 или 1)

### Переназначение ревьювера

1. Заменяется **один** конкретный ревьювер
2. Новый ревьювер выбирается из **команды заменяемого** ревьювера
3. Новый ревьювер должен быть **активным**
4. Новый ревьювер **не должен быть** автором PR или уже назначенным ревьювером
5. Переназначение **запрещено** после merge PR

### Merge PR

1. Операция **идемпотентная** - повторный вызов не вызывает ошибку
2. После merge изменение списка ревьюверов **запрещено**
3. Устанавливается timestamp `mergedAt`

### Активность пользователей

1. Неактивные пользователи (`is_active = false`) **не назначаются** на ревью
2. Деактивация пользователя **не влияет** на уже назначенные PR

## База данных

### Схема

```sql
teams
  - team_name (PK)
  - created_at

users
  - user_id (PK)
  - username
  - team_name (FK -> teams)
  - is_active
  - created_at
  - updated_at

pull_requests
  - pull_request_id (PK)
  - pull_request_name
  - author_id (FK -> users)
  - status (OPEN | MERGED)
  - created_at
  - merged_at

pr_reviewers (связь many-to-many)
  - pull_request_id (FK -> pull_requests)
  - user_id (FK -> users)
  - assigned_at
  - PRIMARY KEY (pull_request_id, user_id)
```

### Индексы

Созданы индексы для оптимизации частых запросов:
- `idx_users_team_name` - поиск пользователей по команде
- `idx_users_is_active` - фильтрация активных пользователей
- `idx_pr_author_id` - поиск PR по автору
- `idx_pr_status` - фильтрация PR по статусу
- `idx_pr_reviewers_user_id` - поиск PR по ревьюверу

## Тестирование

### Unit тесты

Тестируют бизнес-логику в изоляции:

```bash
make test-unit
```

Покрывают:
- Выбор случайных ревьюверов
- Обработку граничных случаев (0, 1, 2+ кандидатов)
- Уникальность выбранных ревьюверов
- Случайность выбора

### E2E тесты

Тестируют полный рабочий процесс через HTTP API:

```bash
# Установите TEST_DATABASE_URL перед запуском
export TEST_DATABASE_URL="postgres://pruser:prpassword@localhost:5432/pr_reviewer_test_db?sslmode=disable"
make test-e2e
```

Покрывают:
- Создание команд и пользователей
- Создание PR с автоматическим назначением ревьюверов
- Переназначение ревьюверов
- Merge PR и идемпотентность
- Деактивацию пользователей
- Обработку ошибок (дубликаты, несуществующие ресурсы)

### Запуск всех тестов

```bash
make test
```

## Коды ошибок API

| Код | HTTP Status | Описание |
|-----|-------------|----------|
| `TEAM_EXISTS` | 400 | Команда с таким именем уже существует |
| `PR_EXISTS` | 409 | PR с таким ID уже существует |
| `PR_MERGED` | 409 | Нельзя изменить PR после merge |
| `NOT_ASSIGNED` | 409 | Пользователь не назначен ревьювером на этот PR |
| `NO_CANDIDATE` | 409 | Нет доступных кандидатов для переназначения |
| `NOT_FOUND` | 404 | Ресурс не найден |

## Принятые решения и допущения

### 1. Случайный выбор ревьюверов - случайный выбор с помощью `math/rand`.
### 2. Переназначение из команды заменяемого ревьювера - новый ревьювер выбирается из команды **заменяемого** ревьювера, а не из команды автора PR.
### 3. Идемпотентность merge - Операция идемпотентная - повторный вызов возвращает 200 OK с текущим состоянием PR, не вызывая ошибку.
### 4. Поведение при отсутствии кандидатов
Если в команде нет активных участников для назначения?
- При создании PR: назначается 0, 1 или 2 ревьювера в зависимости от доступных кандидатов
- При переназначении: возвращается ошибка `NO_CANDIDATE`
### 5. Upsert пользователей при создании команды - Используется `ON CONFLICT DO UPDATE` - пользователь обновляется (имя, команда, активность). Это позволяет переносить пользователей между командами.
### 6. Миграции - Миграции применяются автоматически при старте приложения. Используется простой подход с чтением SQL файла.
### 7. Обработка несуществующего пользователя в getReview - Возвращается пустой список PR. Это более мягкое поведение, чем ошибка 404.



## Переменные окружения

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `DATABASE_URL` | URL подключения к PostgreSQL | `postgres://pruser:prpassword@localhost:5432/pr_reviewer_db?sslmode=disable` |
| `PORT` | Порт HTTP сервера | `8080` |

## Требования

- Go 1.21+
- PostgreSQL 15+
- Docker & Docker Compose (для запуска через контейнеры)

## Лицензия

MIT
