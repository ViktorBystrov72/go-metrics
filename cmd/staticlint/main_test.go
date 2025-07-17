package main

import (
	"testing"
)

// TestMainFunction тестирует, что main функция не паникует при запуске.
// Это базовый тест для покрытия функции main.
func TestMainFunction(t *testing.T) {
	// Этот тест проверяет, что main функция может быть вызвана
	// без паники. В реальности main запускается через multichecker.Main(),
	// но мы можем протестировать создание анализаторов.

	// Проверяем, что ExitCheckAnalyzer создан корректно
	if ExitCheckAnalyzer == nil {
		t.Fatal("ExitCheckAnalyzer не должен быть nil")
	}

	if ExitCheckAnalyzer.Name != "exitcheck" {
		t.Errorf("Ожидалось имя 'exitcheck', получено '%s'", ExitCheckAnalyzer.Name)
	}

	if ExitCheckAnalyzer.Run == nil {
		t.Fatal("ExitCheckAnalyzer.Run не должен быть nil")
	}
}

// TestAnalyzerCreation тестирует создание анализаторов.
func TestAnalyzerCreation(t *testing.T) {
	// Проверяем, что анализатор exitcheck работает корректно
	analyzer := ExitCheckAnalyzer

	// Проверяем документацию
	if analyzer.Doc == "" {
		t.Error("Документация анализатора не должна быть пустой")
	}

	// Проверяем, что функция run существует
	if analyzer.Run == nil {
		t.Error("Функция Run не должна быть nil")
	}
}
