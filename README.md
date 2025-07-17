# Сервис сбора метрик

Сервис для сбора и хранения метрик с поддержкой PostgreSQL и retry логики для обработки временных ошибок.
Поддерживает асимметричное шифрование RSA для безопасной передачи данных между агентом и сервером.

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
  
### Процент покрытия тестами
```bash
go test ./... -coverprofile=coverage.out 
go tool cover -func=coverage.out | tail -1
```  
  
#### Тесты итерации 7 (файловое хранилище)
metricstest -test.v -test.run=^TestIteration7$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=.

#### Тесты итерации 10 (PostgreSQL + fallback)
metricstest -test.v -test.run='^TestIteration10A$|^TestIteration10B$' -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

#### Тесты итерации 11 (PostgreSQL)
metricstest -test.v -test.run=^TestIteration11$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

#### Тесты итерации 12 (Batch API)
metricstest -test.v -test.run=^TestIteration12$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -database-dsn="postgres://user:pass@localhost:5432/dbname?sslmode=disable"

#### Тесты итерации 14 (HashKey)
metricstest -test.v -test.run=^TestIteration14$ -agent-binary-path=cmd/agent/agent -binary-path=cmd/server/server -server-port=8080 -source-path=. -key="invalidkey" -database-dsn="postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"

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

## Оптимизация памяти проекта
### Сбор базового профиля
- Базовый профиль памяти: `profiles/base.pprof`
- Основные потребители памяти:
    - runtime.allocm (58.62%) — выделение памяти для машин
    - zap.newCounters (17.56%) — инициализация логгера
    - validator.map.init.7 (12.01%) — инициализация валидатора
    - runtime.procresize (11.81%) — настройка процессоров

### Результаты оптимизации

##### Потребление памяти ДО оптимизации:
- Общее потребление: ~4.27MB
- Основные потребители: runtime.allocm, zap.newCounters, validator

#### Потребление памяти ПОСЛЕ оптимизации:
- Общее потребление: ~1.69MB
- Уменьшение на ~60%

#### Ключевые улучшения:
1. **Логгер**: Оптимизирована конфигурация zap для снижения потребления памяти
2. **MemStorage**: Используется RWMutex и уменьшено копирование данных

#### Сбор базового профиля:
```bash
go tool pprof -proto -output=profiles/base.pprof http://localhost:6060/debug/pprof/heap
```

#### Нагрузочное тестирование:
```bash
./scripts/profile_memory.sh
```

#### Сбор результативного профиля:
```bash
go tool pprof -proto -output=profiles/result.pprof http://localhost:6060/debug/pprof/heap
```

#### Сравнение профилей:
```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

# Сборка и запуск

## Сборка с версией, датой и коммитом

Для удобной сборки используйте Makefile. При сборке автоматически подставляются:
- версия (VERSION)
- дата сборки (BUILD_DATE)
- коммит (BUILD_COMMIT)

### Быстрая сборка

```sh
make build
```

### Сборка с указанием версии

```sh
make build-with-version VERSION=1.2.3
```

### Ручная сборка через go build

Можно передать значения переменных через флаги линковщика:

```sh
go build -ldflags "-X main.buildVersion=1.2.3 -X 'main.buildDate=2025-07-15 20:00:00' -X main.buildCommit=abc123" -o bin/server cmd/server/main.go
```

## Проверка версии

После сборки при запуске приложения будет выведена информация:

```
Build version: 1.2.3
Build date: 2025-07-15 20:00:00
Build commit: abc123
```

Если переменные не заданы, будет выведено N/A.


## Шифрование  

- **Агент** использует публичный ключ для шифрования данных метрик перед отправкой
- **Сервер** использует приватный ключ для дешифрования входящих данных
- Шифрование применяется поверх уже сжатых gzip данных
- Зашифрованные данные передаются в формате Base64 с заголовком `Content-Encoding: encrypted`

### Генерация ключей

Утилита для генерации RSA ключей:

```bash
go run cmd/keygen/main.go -private keys/private.pem -public keys/public.pem -size 2048
```

Параметры:
- `-private` - путь к файлу приватного ключа (по умолчанию: private.pem)
- `-public` - путь к файлу публичного ключа (по умолчанию: public.pem)
- `-size` - размер ключа в битах (по умолчанию: 2048)

### Конфигурация

#### Агент

Для включения шифрования на агенте:

```bash
# Через флаг
./agent -crypto-key /path/to/public.pem

# Через переменную окружения
export CRYPTO_KEY=/path/to/public.pem
./agent
```

#### Сервер

Для включения дешифрования на сервере:

```bash
# Через флаг
./server -crypto-key /path/to/private.pem

# Через переменную окружения
export CRYPTO_KEY=/path/to/private.pem
./server
```

### Пример использования

1. Сгенерируйте ключи:
```bash
mkdir keys
go run cmd/keygen/main.go -private keys/private.pem -public keys/public.pem
```

2. Запуск сервера с дешифрованием:
```bash
./server -crypto-key keys/private.pem
```

3. Запуск агента с шифрованием:
```bash
./agent -crypto-key keys/public.pem
```

### Технические детали

#### Алгоритм шифрования
- **Алгоритм**: RSA с PKCS1v15 padding
- **Размер ключа**: 2048 бит (по умолчанию)
- **Формат ключей**: PEM

#### Процесс шифрования
1. Агент сериализует метрики в JSON
2. Сжимает данные с помощью gzip
3. Шифрует сжатые данные RSA (по блокам при необходимости)
4. Кодирует в Base64 для передачи
5. Отправляет с заголовком `Content-Encoding: encrypted`

#### Процесс дешифрования
1. Сервер получает запрос с заголовком `Content-Encoding: encrypted`
2. Декодирует данные из Base64
3. Дешифрует данные RSA
4. Восстанавливает заголовок `Content-Encoding: gzip`
5. Передает данные в GzipMiddleware для разжатия

### Примеры ключей для разработки

В директории `keys/` уже созданы тестовые ключи для разработки:
- `keys/private.pem` - приватный ключ для сервера
- `keys/public.pem` - публичный ключ для агента

## JSON Конфигурация

### Приоритет конфигураций

Значения применяются в следующем порядке приоритета (от высшего к низшему):

1. **Флаги командной строки** (наивысший приоритет)
2. **Переменные окружения**
3. **JSON файл конфигурации** (наименьший приоритет)

### Указание файла конфигурации

Файл конфигурации можно указать двумя способами:

- Флаг командной строки: `-c config.json` или `-config config.json`
- Переменная окружения: `CONFIG=config.json`

### Формат конфигурации сервера

```json
{
    "address": "localhost:8080",
    "restore": true,
    "store_interval": "300s",
    "store_file": "/tmp/metrics-db.json",
    "database_dsn": "postgres://user:pass@localhost/db",
    "crypto_key": "/path/to/private.pem"
}
```

#### Поля конфигурации сервера

- `address` - адрес и порт сервера (аналог флага `-a`)
- `restore` - восстанавливать ли данные при запуске (аналог флага `-r`)
- `store_interval` - интервал сохранения данных (аналог флага `-i`)
- `store_file` - путь к файлу хранения (аналог флага `-f`)
- `database_dsn` - строка подключения к БД (аналог флага `-d`)
- `crypto_key` - путь к приватному ключу для дешифрования (аналог флага `-crypto-key`)

### Формат конфигурации агента

```json
{
    "address": "localhost:8080",
    "report_interval": "10s",
    "poll_interval": "2s",
    "crypto_key": "/path/to/public.pem"
}
```

#### Поля конфигурации агента

- `address` - адрес сервера (аналог флага `-a`)
- `report_interval` - интервал отправки метрик (аналог флага `-r`)
- `poll_interval` - интервал сбора метрик (аналог флага `-p`)
- `crypto_key` - путь к публичному ключу для шифрования (аналог флага `-crypto-key`)

### Форматы времени

Интервалы времени поддерживают следующие единицы:
- `s` - секунды (например: `30s`)
- `m` - минуты (например: `5m`)
- `h` - часы (например: `1h`)
- Комбинации (например: `1h30m`)

### Примеры использования

#### Запуск сервера с JSON конфигурацией

```bash
# Через флаг
./server -c configs/server.json

# Через переменную окружения
CONFIG=configs/server.json ./server
```

#### Запуск агента с JSON конфигурацией

```bash
# Через флаг
./agent -config configs/agent.json

# Через переменную окружения
CONFIG=configs/agent.json ./agent
```

#### Переопределение значений

```bash
# JSON файл устанавливает address: "localhost:8080"
# Флаг переопределяет значение
./server -c configs/server.json -a "0.0.0.0:9090"
```

### Обратная совместимость

- Если JSON файл не указан, приложения работают как раньше
- Все существующие флаги и переменные окружения сохраняют свою функциональность
- Переменные окружения вида `STORE_INTERVAL=300` (без единицы) автоматически преобразуются в `300s`

### Примеры конфигураций

В директории `configs/` содержатся примеры конфигураций:
- `configs/server.json` - базовая конфигурация сервера
- `configs/agent.json` - базовая конфигурация агента
- `configs/server_example.json` - расширенная конфигурация сервера
- `configs/agent_example.json` - расширенная конфигурация агента 
