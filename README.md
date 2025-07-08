# Сервис сбора метрик

Сервис для сбора и хранения метрик с поддержкой PostgreSQL и retry логики для обработки временных ошибок.

## Возможности

- Сбор метрик типа gauge и counter
- Хранение в памяти, файле или PostgreSQL
- Batch API для обновления множества метрик
- Gzip сжатие для HTTP запросов
- Retry логика для обработки временных ошибок**
- Автоматический fallback между типами хранилищ
- Подпись данных по алгоритму SHA256 для обеспечения целостности

## Retry логика

Сервис включает в себя интеллектуальную retry логику для обработки временных ошибок:

### Поддерживаемые retriable ошибки:
- **Сетевые ошибки**: connection refused, connection reset, broken pipe
- **PostgreSQL ошибки класса 08**: Connection Exception
- **DNS ошибки**: временные ошибки разрешения имен
- **Таймауты**: сетевые и HTTP таймауты
- **Перегрузка сервера**: too many connections, server overloaded

### Настройки retry:
- **Количество попыток**: 4 (1 основная + 3 повтора)
- **Интервалы**: 1s, 3s, 5s между попытками
- **Таймауты**: 10s для HTTP запросов, 30s общий таймаут

### Применение:
- **Агент**: retry при отправке метрик на сервер
- **Сервер**: retry при работе с PostgreSQL
- **Batch операции**: retry для batch обновлений

## Хеширование и подпись данных

Сервис поддерживает механизм подписи передаваемых данных по алгоритму SHA256 для обеспечения целостности данных:

### Агент
- Поддержка флага `-k=<КЛЮЧ>` и переменной окружения `KEY=<КЛЮЧ>`
- При наличии ключа вычисляет HMAC-SHA256 хеш от тела запроса
- Передает хеш в HTTP-заголовке `HashSHA256`

### Сервер
- Поддержка флага `-k=<КЛЮЧ>` и переменной окружения `KEY=<КЛЮЧ>`
- При наличии ключа проверяет соответствие полученного и вычисленного хеша
- При несовпадении возвращает `http.StatusBadRequest`
- При формировании ответа вычисляет хеш и передает в заголовке `HashSHA256`

### Примеры использования

```bash
# Запуск агента с ключом
./bin/agent -k="my-secret-key"

# Запуск сервера с ключом
./bin/server -k="my-secret-key"

# Через переменные окружения
KEY="my-secret-key" ./bin/agent
KEY="my-secret-key" ./bin/server
```

**Примечание**: Это учебный пример для демонстрации механизмов подписи. В реальных проектах рекомендуется использовать более надежные методы аутентификации и авторизации.

## Архитектура

### Компоненты

1. **Agent** (`cmd/agent/`) - собирает метрики и отправляет на сервер
2. **Server** (`cmd/server/`) - принимает и хранит метрики
3. **Storage** (`internal/storage/`) - интерфейсы и реализации хранилищ
4. **Utils** (`internal/utils/`) - утилиты, включая retry логику

### Типы хранилищ

1. **MemoryStorage** - хранение в памяти (по умолчанию)
2. **FileStorage** - хранение в JSON файле
3. **DatabaseStorage** - хранение в PostgreSQL с retry логикой и pgxpool

### Технологии

- **PostgreSQL** - используется pgxpool для эффективного пула соединений
- **Retry логика** - автоматические повторы при временных ошибках
- **Gzip сжатие** - для HTTP запросов
- **Batch API** - для массового обновления метрик

## Установка и запуск

### Требования
- Go 1.21+
- PostgreSQL (опционально)

### Запуск PostgreSQL через Docker Compose

```bash
docker compose up -d
```

Проверка, что база данных запущена:

```bash
docker compose ps
```

Подключитесь к базе данных (опционально):

```bash
docker compose exec postgres psql -U postgres -d praktikum
```

### Сборка
```bash
go build -o bin/agent cmd/agent/main.go
go build -o bin/server cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go
```

### Запуск сервера

#### С PostgreSQL:
```bash
DATABASE_DSN='postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable' ./bin/server
```

#### С файловым хранилищем:
```bash
./bin/server
```

### Запуск агента:
```bash
./bin/agent
```

## API

### Обновление метрики
```http
POST /update/{type}/{name}/{value}
```

### Batch обновление метрик
```http
POST /updates/
Content-Type: application/json
Content-Encoding: gzip

[
  {
    "id": "metric1",
    "type": "gauge",
    "value": 123.45
  },
  {
    "id": "metric2", 
    "type": "counter",
    "delta": 10
  }
]
```

### Получение значения метрики
```http
POST /value/
Content-Type: application/json

{
  "id": "metric1",
  "type": "gauge"
}
```

## Конфигурация

### Переменные окружения агента:
- `ADDRESS` - адрес сервера (по умолчанию: localhost:8080)
- `REPORT_INTERVAL` - интервал отправки метрик (по умолчанию: 10s)
- `POLL_INTERVAL` - интервал сбора метрик (по умолчанию: 2s)

### Переменные окружения сервера:
- `ADDRESS` - адрес для прослушивания (по умолчанию: localhost:8080)
- `DATABASE_DSN` - строка подключения к PostgreSQL
- `FILE_STORAGE_PATH` - путь к файлу для хранения метрик
- `RESTORE` - восстанавливать метрики из файла (по умолчанию: true)

## Логика выбора хранилища

1. Если указан `DATABASE_DSN` → PostgreSQL с retry логикой
2. Если указан `FILE_STORAGE_PATH` → файловое хранилище
3. Иначе → хранение в памяти

## Тестирование

### Запуск тестов:
```bash
go test ./...
```

#### Тесты итерации 7 (файловое хранилище)
metricstest -test.v -test.run=^TestIteration7$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=.

#### Тесты итерации 10 (PostgreSQL + fallback)
metricstest -test.v -test.run='^TestIteration10A$|^TestIteration10B$' -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

#### Тесты итерации 11 (PostgreSQL)
metricstest -test.v -test.run=^TestIteration11$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

#### Тесты итерации 12 (Batch API)
metricstest -test.v -test.run=^TestIteration12$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

### Тестирование retry логики:
```bash
# Тест с недоступным сервером
./bin/agent  # Агент будет retry отправку метрик

# Тест с недоступной PostgreSQL
DATABASE_DSN='postgres://invalid:invalid@localhost:5432/invalid' ./bin/server
```

### Бенчмарки:
```bash
go test -bench=. ./internal/storage/
```

## Примеры использования

### Отправка метрики с retry:
```bash
curl -X POST "http://localhost:8080/update/gauge/test/123.45"
```

### Batch отправка с gzip:
```bash
echo '[{"id":"test","type":"gauge","value":123.45}]' | gzip | \
curl -X POST "http://localhost:8080/updates/" \
  -H "Content-Type: application/json" \
  -H "Content-Encoding: gzip" \
  --data-binary @-
```

## Агент

Агент автоматически собирает метрики runtime и отправляет их на сервер:

- **Batch отправка** - агент отправляет все метрики одним запросом через `/updates/`
- **Gzip сжатие** - все запросы сжимаются
- **Retry логика** - автоматические повторы при временных ошибках
- **Настраиваемые интервалы** - можно настроить частоту сбора и отправки метрик

### Параметры агента

- `-a` - адрес сервера (по умолчанию: localhost:8080)
- `-r` - интервал отправки в секундах (по умолчанию: 10)
- `-p` - интервал сбора в секундах (по умолчанию: 2)

### Переменные окружения агента

- `ADDRESS` - адрес сервера
- `REPORT_INTERVAL` - интервал отправки в секундах
- `POLL_INTERVAL` - интервал сбора в секундах

## База данных

### Миграции

Сервис использует [goose](https://github.com/pressly/goose) для управления миграциями базы данных:

```bash
# Применить миграции
./bin/migrate -dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable" -command=up

# Проверить статус миграций
./bin/migrate -dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable" -command=status
```

### Структура базы данных

Миграции автоматически создают таблицу `metrics` со следующей структурой:

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
- Автоматическое применение миграций при запуске
- Используется pgxpool для эффективного пула соединений
- Метрики сохраняются сразу при обновлении
- Поддержка уникальных ограничений для предотвращения дублирования
