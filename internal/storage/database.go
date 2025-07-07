package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ViktorBystrov72/go-metrics/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// DatabaseStorage реализует интерфейс Storage для PostgreSQL
type DatabaseStorage struct {
	db *sql.DB
}

// NewDatabaseStorage создает новое подключение к PostgreSQL
func NewDatabaseStorage(dsn string) (*DatabaseStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &DatabaseStorage{db: db}, nil
}

// createTables создает необходимые таблицы в базе данных
func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS metrics (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		value DOUBLE PRECISION,
		delta BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, type)
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_name_type ON metrics(name, type);
	CREATE INDEX IF NOT EXISTS idx_metrics_created_at ON metrics(created_at);
	`

	_, err := db.Exec(query)
	return err
}

// Ping проверяет соединение с базой данных
func (d *DatabaseStorage) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.db.PingContext(ctx)
}

// Close закрывает соединение с базой данных
func (d *DatabaseStorage) Close() error {
	return d.db.Close()
}

// UpdateGauge обновляет gauge метрику в базе данных
func (d *DatabaseStorage) UpdateGauge(name string, value float64) {
	query := `
	INSERT INTO metrics (name, type, value) 
	VALUES ($1, 'gauge', $2)
	ON CONFLICT (name, type) 
	DO UPDATE SET value = $2, created_at = CURRENT_TIMESTAMP
	WHERE metrics.name = $1 AND metrics.type = 'gauge'
	`

	_, _ = d.db.Exec(query, name, value)
}

// UpdateCounter обновляет counter метрику в базе данных
func (d *DatabaseStorage) UpdateCounter(name string, value int64) {
	query := `
	INSERT INTO metrics (name, type, delta) 
	VALUES ($1, 'counter', $2)
	ON CONFLICT (name, type) 
	DO UPDATE SET delta = metrics.delta + $2, created_at = CURRENT_TIMESTAMP
	WHERE metrics.name = $1 AND metrics.type = 'counter'
	`

	_, _ = d.db.Exec(query, name, value)
}

// GetGauge получает gauge метрику из базы данных
func (d *DatabaseStorage) GetGauge(name string) (float64, bool) {
	var value float64
	query := `SELECT value FROM metrics WHERE name = $1 AND type = 'gauge' ORDER BY created_at DESC LIMIT 1`

	err := d.db.QueryRow(query, name).Scan(&value)
	if err != nil {
		return 0, false
	}

	return value, true
}

// GetCounter получает counter метрику из базы данных
func (d *DatabaseStorage) GetCounter(name string) (int64, bool) {
	var value int64
	query := `SELECT delta FROM metrics WHERE name = $1 AND type = 'counter' ORDER BY created_at DESC LIMIT 1`

	err := d.db.QueryRow(query, name).Scan(&value)
	if err != nil {
		return 0, false
	}

	return value, true
}

// GetAllGauges получает все gauge метрики из базы данных
func (d *DatabaseStorage) GetAllGauges() map[string]float64 {
	gauges := make(map[string]float64)
	query := `SELECT name, value FROM metrics WHERE type = 'gauge' ORDER BY created_at DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return gauges
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err == nil {
			gauges[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return make(map[string]float64)
	}

	return gauges
}

// GetAllCounters получает все counter метрики из базы данных
func (d *DatabaseStorage) GetAllCounters() map[string]int64 {
	counters := make(map[string]int64)
	query := `SELECT name, delta FROM metrics WHERE type = 'counter' ORDER BY created_at DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return counters
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err == nil {
			counters[name] = value
		}
	}

	if err := rows.Err(); err != nil {
		return make(map[string]int64)
	}

	return counters
}

// SaveToFile - заглушка для совместимости с интерфейсом
func (d *DatabaseStorage) SaveToFile(filename string) error {
	// Для базы данных сохранение в файл не требуется
	return nil
}

// LoadFromFile - заглушка для совместимости с интерфейсом
func (d *DatabaseStorage) LoadFromFile(filename string) error {
	// Для базы данных загрузка из файла не требуется
	return nil
}

// IsDatabase возвращает true, так как это база данных
func (d *DatabaseStorage) IsDatabase() bool {
	return true
}

// IsAvailable возвращает true, если соединение с БД установлено
func (d *DatabaseStorage) IsAvailable() bool {
	return d.db != nil
}

// BrokenStorage реализует Storage, всегда возвращает ошибку
// Используется, если DSN задан, но подключение к БД не удалось

type BrokenStorage struct{}

func (b *BrokenStorage) UpdateGauge(name string, value float64) {}
func (b *BrokenStorage) UpdateCounter(name string, value int64) {}
func (b *BrokenStorage) GetGauge(name string) (float64, bool)   { return 0, false }
func (b *BrokenStorage) GetCounter(name string) (int64, bool)   { return 0, false }
func (b *BrokenStorage) GetAllGauges() map[string]float64       { return map[string]float64{} }
func (b *BrokenStorage) GetAllCounters() map[string]int64       { return map[string]int64{} }
func (b *BrokenStorage) SaveToFile(filename string) error       { return fmt.Errorf("storage unavailable") }
func (b *BrokenStorage) LoadFromFile(filename string) error     { return fmt.Errorf("storage unavailable") }
func (b *BrokenStorage) Ping() error                            { return fmt.Errorf("storage unavailable") }
func (b *BrokenStorage) IsDatabase() bool                       { return true }
func (b *BrokenStorage) IsAvailable() bool                      { return false }

// UpdateBatch для BrokenStorage всегда возвращает ошибку
func (b *BrokenStorage) UpdateBatch(metrics []models.Metrics) error {
	return fmt.Errorf("storage unavailable")
}

// UpdateBatch обновляет множество метрик в одной транзакции
func (d *DatabaseStorage) UpdateBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Начинаем транзакцию
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем запросы для gauge и counter
	gaugeStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (name, type, value) 
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (name, type) 
		DO UPDATE SET value = $2, created_at = CURRENT_TIMESTAMP
		WHERE metrics.name = $1 AND metrics.type = 'gauge'
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare gauge statement: %w", err)
	}
	defer gaugeStmt.Close()

	counterStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (name, type, delta) 
		VALUES ($1, 'counter', $2)
		ON CONFLICT (name, type) 
		DO UPDATE SET delta = metrics.delta + $2, created_at = CURRENT_TIMESTAMP
		WHERE metrics.name = $1 AND metrics.type = 'counter'
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare counter statement: %w", err)
	}
	defer counterStmt.Close()

	// Выполняем все обновления в транзакции
	for _, m := range metrics {
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				return fmt.Errorf("gauge metric %s has nil value", m.ID)
			}
			_, err := gaugeStmt.ExecContext(ctx, m.ID, *m.Value)
			if err != nil {
				return fmt.Errorf("failed to update gauge metric %s: %w", m.ID, err)
			}
		case "counter":
			if m.Delta == nil {
				return fmt.Errorf("counter metric %s has nil delta", m.ID)
			}
			_, err := counterStmt.ExecContext(ctx, m.ID, *m.Delta)
			if err != nil {
				return fmt.Errorf("failed to update counter metric %s: %w", m.ID, err)
			}
		default:
			return fmt.Errorf("unknown metric type: %s", m.MType)
		}
	}

	return tx.Commit()
}
