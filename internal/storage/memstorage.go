package storage

import (
	"encoding/json"
	"os"
	"sync"
)

// MemStorage реализация хранилища в памяти
type MemStorage struct {
	mu       sync.RWMutex
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.gauges[name]
	return value, exists
}

// GetCounter возвращает значение counter метрики
func (s *MemStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.counters[name]
	return value, exists
}

// GetAllGauges возвращает все gauge метрики
func (s *MemStorage) GetAllGauges() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]float64)
	for k, v := range s.gauges {
		result[k] = v
	}
	return result
}

// GetAllCounters возвращает все counter метрики
func (s *MemStorage) GetAllCounters() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	dump := storageDump{
		Gauges:   s.gauges,
		Counters: s.counters,
	}
	s.mu.RUnlock()
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(dump)
}

func (s *MemStorage) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	var dump storageDump
	dec := json.NewDecoder(file)
	if err := dec.Decode(&dump); err != nil {
		return err
	}
	s.mu.Lock()
	s.gauges = dump.Gauges
	s.counters = dump.Counters
	s.mu.Unlock()
	return nil
}
