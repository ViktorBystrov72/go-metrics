# Go Metrics Service

Сервис для сбора и хранения метрик с поддержкой PostgreSQL.

## Возможности

- Сбор метрик runtime (gauge и counter)
- HTTP API для получения и обновления метрик
- Batch API для массового обновления метрик
- Поддержка сжатия gzip
- PostgreSQL как основное хранилище метрик
- Автоматический fallback: PostgreSQL → файл → память
- Проверка соединения с базой данных через `/ping`

## Логика выбора хранилища

Сервер автоматически выбирает хранилище в следующем порядке приоритета:

1. **PostgreSQL** - если указан `DATABASE_DSN` и подключение успешно
2. **Файловое хранилище** - если PostgreSQL недоступен, но указан путь к файлу
3. **Хранилище в памяти** - если ни PostgreSQL, ни файл недоступны

**Важно:** Если `DATABASE_DSN` указан, но подключение к БД не удалось, сервер использует `BrokenStorage` и `/ping` возвращает 500.

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

### Batch API

```http
POST /updates/
Content-Type: application/json

[
  {
    "id": "Alloc",
    "type": "gauge",
    "value": 123.45
  },
  {
    "id": "PollCount",
    "type": "counter",
    "delta": 42
  }
]
```

**Особенности Batch API:**
- Обновляет множество метрик в одной операции
- В PostgreSQL все изменения выполняются в одной транзакции
- Поддерживает gzip сжатие
- Не отправляет пустые батчи
- Обратная совместимость с существующими API

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

## Агент

Агент автоматически собирает метрики runtime и отправляет их на сервер:

- **Batch отправка** - агент отправляет все метрики одним запросом через `/updates/`
- **Fallback** - при ошибке batch отправки агент переключается на отправку по одной метрике
- **Gzip сжатие** - все запросы сжимаются
- **Настраиваемые интервалы** - можно настроить частоту сбора и отправки метрик

### Параметры агента

- `-a` - адрес сервера (по умолчанию: localhost:8080)
- `-r` - интервал отправки в секундах (по умолчанию: 10)
- `-p` - интервал сбора в секундах (по умолчанию: 2)

### Переменные окружения агента

- `ADDRESS` - адрес сервера
- `REPORT_INTERVAL` - интервал отправки в секундах
- `POLL_INTERVAL` - интервал сбора в секундах

## Тестирование

```bash
# Запуск тестов
go test ./...

# Тесты итерации 7 (файловое хранилище)
metricstest -test.v -test.run=^TestIteration7$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=.

# Тесты итерации 10 (PostgreSQL + fallback)
metricstest -test.v -test.run='^TestIteration10A$|^TestIteration10B$' -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# Тесты итерации 11 (PostgreSQL)
metricstest -test.v -test.run=^TestIteration11$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

# Тесты итерации 12 (Batch API)
metricstest -test.v -test.run=^TestIteration12$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```
