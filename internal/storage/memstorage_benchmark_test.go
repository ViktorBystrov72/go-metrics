package storage

import (
	"testing"
)

// BenchmarkMemStorage_ReadHeavy тестирует производительность при преобладании операций чтения
func BenchmarkMemStorage_ReadHeavy(b *testing.B) {
	storage := NewMemStorage()

	// Предварительно заполняем данными
	for i := 0; i < 1000; i++ {
		storage.UpdateGauge("gauge"+string(rune(i)), float64(i))
		storage.UpdateCounter("counter"+string(rune(i)), int64(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 90% операций чтения, 10% записи
			if i%10 == 0 {
				storage.UpdateGauge("gauge"+string(rune(i%1000)), float64(i))
				storage.UpdateCounter("counter"+string(rune(i%1000)), 1)
			} else {
				storage.GetGauge("gauge" + string(rune(i%1000)))
				storage.GetCounter("counter" + string(rune(i%1000)))
			}
			i++
		}
	})
}

// BenchmarkMemStorage_WriteHeavy тестирует производительность при преобладании операций записи
func BenchmarkMemStorage_WriteHeavy(b *testing.B) {
	storage := NewMemStorage()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 90% операций записи, 10% чтения
			if i%10 != 0 {
				storage.UpdateGauge("gauge"+string(rune(i%1000)), float64(i))
				storage.UpdateCounter("counter"+string(rune(i%1000)), 1)
			} else {
				storage.GetGauge("gauge" + string(rune(i%1000)))
				storage.GetCounter("counter" + string(rune(i%1000)))
			}
			i++
		}
	})
}

// BenchmarkMemStorage_Balanced тестирует производительность при сбалансированной нагрузке
func BenchmarkMemStorage_Balanced(b *testing.B) {
	storage := NewMemStorage()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 50% операций чтения, 50% записи
			if i%2 == 0 {
				storage.UpdateGauge("gauge"+string(rune(i%1000)), float64(i))
				storage.UpdateCounter("counter"+string(rune(i%1000)), 1)
			} else {
				storage.GetGauge("gauge" + string(rune(i%1000)))
				storage.GetCounter("counter" + string(rune(i%1000)))
			}
			i++
		}
	})
}

// BenchmarkMemStorage_GetAllOperations тестирует производительность операций получения всех метрик
func BenchmarkMemStorage_GetAllOperations(b *testing.B) {
	storage := NewMemStorage()

	// Предварительно заполняем данными
	for i := 0; i < 1000; i++ {
		storage.UpdateGauge("gauge"+string(rune(i)), float64(i))
		storage.UpdateCounter("counter"+string(rune(i)), int64(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			storage.GetAllGauges()
			storage.GetAllCounters()
		}
	})
}

// BenchmarkMemStorage_SaveLoad тестирует производительность операций сохранения и загрузки
func BenchmarkMemStorage_SaveLoad(b *testing.B) {
	storage := NewMemStorage()

	// Предварительно заполняем данными
	for i := 0; i < 1000; i++ {
		storage.UpdateGauge("gauge"+string(rune(i)), float64(i))
		storage.UpdateCounter("counter"+string(rune(i)), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := "benchmark_test.json"
		if err := storage.SaveToFile(filename); err != nil {
			b.Fatal(err)
		}
		if err := storage.LoadFromFile(filename); err != nil {
			b.Fatal(err)
		}
	}
}
