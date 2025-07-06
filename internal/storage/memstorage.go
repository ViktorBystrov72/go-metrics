package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// MemStorage реализация хранилища в памяти
type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage создает новый экземпляр хранилища в памяти
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge обновляет значение gauge метрики
func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

// UpdateCounter обновляет значение counter метрики
func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

// GetGauge возвращает значение gauge метрики
func (s *MemStorage) GetGauge(name string) (float64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.gauges[name]
	return value, exists
}

// GetCounter возвращает значение counter метрики
func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.counters[name]
	return value, exists
}

// GetAllGauges возвращает все gauge метрики
func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]float64)
	for k, v := range s.gauges {
		result[k] = v
	}
	return result
}

// GetAllCounters возвращает все counter метрики
func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]int64)
	for k, v := range s.counters {
		result[k] = v
	}
	return result
}

type storageDump struct {
	Gauges   map[string]float64 `json:"gauges"`
	Counters map[string]int64   `json:"counters"`
}

func (s *MemStorage) SaveToFile(filename string) error {
	s.mu.Lock()
	gaugesCopy := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		gaugesCopy[k] = v
	}
	countersCopy := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		countersCopy[k] = v
	}
	s.mu.Unlock()

	dump := storageDump{
		Gauges:   gaugesCopy,
		Counters: countersCopy,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	if err := enc.Encode(dump); err != nil {
		return fmt.Errorf("failed to encode data to file %s: %w", filename, err)
	}
	return nil
}

func (s *MemStorage) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()
	var dump storageDump
	dec := json.NewDecoder(file)
	if err := dec.Decode(&dump); err != nil {
		return fmt.Errorf("failed to decode data from file %s: %w", filename, err)
	}

	s.mu.Lock()
	s.gauges = make(map[string]float64, len(dump.Gauges))
	for k, v := range dump.Gauges {
		s.gauges[k] = v
	}
	s.counters = make(map[string]int64, len(dump.Counters))
	for k, v := range dump.Counters {
		s.counters[k] = v
	}
	s.mu.Unlock()
	return nil
}

// Ping проверяет соединение с хранилищем (для совместимости с интерфейсом)
func (s *MemStorage) Ping() error {
	// Для хранилища в памяти всегда возвращаем nil
	return nil
}
