package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

// TestExitCheckAnalyzer тестирует собственный анализатор exitcheck.
func TestExitCheckAnalyzer(t *testing.T) {
	// analysistest.Run применяет тестируемый анализатор ExitCheckAnalyzer
	// к пакетам из папки testdata и проверяет ожидания
	analysistest.Run(t, analysistest.TestData(), ExitCheckAnalyzer, "pkg1", "pkg2")
}
