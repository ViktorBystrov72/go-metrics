package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"honnef.co/go/tools/staticcheck"
)

// Multichecker объединяет различные статические анализаторы в один инструмент.
// Включает стандартные анализаторы из golang.org/x/tools/go/analysis/passes,
// анализаторы класса SA из staticcheck.io, а также собственный анализатор.
func main() {
	// Создаем список анализаторов staticcheck класса SA
	var saAnalyzers []*analysis.Analyzer
	for _, analyzer := range staticcheck.Analyzers {
		// Добавляем только анализаторы класса SA (SA1xxx, SA2xxx, etc.)
		if len(analyzer.Analyzer.Name) >= 2 && analyzer.Analyzer.Name[:2] == "SA" {
			saAnalyzers = append(saAnalyzers, analyzer.Analyzer)
		}
	}

	// Создаем список всех анализаторов
	analyzers := []*analysis.Analyzer{
		// Собственный анализатор
		ExitCheckAnalyzer,

		// Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
		printf.Analyzer,      // Проверяет соответствие спецификаторов printf и аргументов
		shadow.Analyzer,      // Находит затененные переменные
		shift.Analyzer,       // Проверяет корректность операций побитового сдвига
		structtag.Analyzer,   // Проверяет корректность тегов структур
		unreachable.Analyzer, // Находит недостижимый код
	}

	// Добавляем анализаторы класса SA из staticcheck
	analyzers = append(analyzers, saAnalyzers...)

	// Добавляем несколько анализаторов других классов из staticcheck
	// S1xxx - упрощения кода
	// ST1xxx - стилистические проверки
	for _, analyzer := range staticcheck.Analyzers {
		if len(analyzer.Analyzer.Name) >= 2 {
			prefix := analyzer.Analyzer.Name[:2]
			if prefix == "S1" || prefix == "ST" {
				analyzers = append(analyzers, analyzer.Analyzer)
			}
		}
	}

	// Запускаем multichecker
	multichecker.Main(analyzers...)
}
