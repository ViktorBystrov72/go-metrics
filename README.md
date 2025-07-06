# Go Metrics Service

Сервис для сбора и хранения метрик с поддержкой PostgreSQL.

## Возможности

- Сбор метрик runtime (gauge и counter)
- HTTP API для получения и обновления метрик
- Поддержка сжатия gzip
- **PostgreSQL как основное хранилище метрик**
- Автоматический fallback: PostgreSQL → файл → память
- Проверка соединения с базой данных через `/ping`

## Логика выбора хранилища

Сервер автоматически выбирает хранилище в следующем порядке приоритета:

1. **PostgreSQL** - если указан `DATABASE_DSN` и подключение успешно
2. **Файловое хранилище** - если PostgreSQL недоступен, но указан путь к файлу
3. **Хранилище в памяти** - если ни PostgreSQL, ни файл недоступны

## Запуск

### С PostgreSQL (рекомендуется)

```bash
# Через флаг командной строки
./cmd/server/server -d "postgres://username:password@localhost:5432/dbname?sslmode=disable"

# Через переменную окружения
export DATABASE_DSN="postgres://username:password@localhost:5432/dbname?sslmode=disable"
./cmd/server/server
```

### С файловым хранилищем (fallback)

```bash
# Сервер автоматически переключится на файловое хранилище
# если PostgreSQL недоступен
./cmd/server/server -f /path/to/metrics.json
```

### Только в памяти (fallback)

```bash
# Сервер использует память, если ни PostgreSQL, ни файл недоступны
./cmd/server/server
```

### Параметры конфигурации

- `-a` - адрес и порт сервера (по умолчанию: localhost:8080)
- `-d` - строка подключения к PostgreSQL
- `-f` - путь к файлу для сохранения метрик (по умолчанию: /tmp/metrics-db.json)
- `-i` - интервал сохранения в секунды (по умолчанию: 300)
- `-r` - восстановление метрик при запуске (по умолчанию: true)

### Переменные окружения

- `ADDRESS` - адрес и порт сервера
- `DATABASE_DSN` - строка подключения к PostgreSQL
- `FILE_STORAGE_PATH` - путь к файлу для сохранения метрик
- `STORE_INTERVAL` - интервал сохранения в секундах
- `RESTORE` - восстановление метрик при запуске

## API

### Проверка соединения с БД

```http
GET /ping
```

**Ответ:**
- `200 OK` - соединение с БД установлено
- `500 Internal Server Error` - ошибка соединения с БД

### Обновление метрик

```http
POST /update/{type}/{name}/{value}
```

**Примеры:**
```bash
curl -X POST "http://localhost:8080/update/gauge/Alloc/123.45"
curl -X POST "http://localhost:8080/update/counter/PollCount/1"
```

### Получение метрик

```http
GET /value/{type}/{name}
```

**Примеры:**
```bash
curl "http://localhost:8080/value/gauge/Alloc"
curl "http://localhost:8080/value/counter/PollCount"
```

### JSON API

```http
POST /update/
Content-Type: application/json

{
  "id": "Alloc",
  "type": "gauge",
  "value": 123.45
}
```

```http
POST /value/
Content-Type: application/json

{
  "id": "Alloc",
  "type": "gauge"
}
```

## База данных

При использовании PostgreSQL автоматически создается таблица `metrics` со следующей структурой:

```sql
CREATE TABLE metrics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    value DOUBLE PRECISION,
    delta BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, type)
);
CREATE INDEX idx_metrics_name_type ON metrics(name, type);
CREATE INDEX idx_metrics_created_at ON metrics(created_at);
```

**Особенности PostgreSQL хранилища:**
- Метрики сохраняются сразу при обновлении (без периодического сохранения)
- Используется `DOUBLE PRECISION` для gauge метрик
- Автоматическое создание индексов для оптимизации запросов
- Поддержка уникальных ограничений для предотвращения дублирования

## Сборка

```bash
# Сборка сервера
cd cmd/server && go build -o server

# Сборка агента
cd cmd/agent && go build -o agent
```

## Тестирование

```bash
# Запуск тестов
go test ./...

metricstest -test.v -test.run=^TestIteration7$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=.

metricstest -test.v -test.run=^TestIteration11$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```
