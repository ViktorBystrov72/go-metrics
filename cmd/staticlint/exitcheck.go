package main

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ExitCheckAnalyzer запрещает прямой вызов os.Exit в функции main пакета main.
// Анализатор находит вызовы os.Exit в функции main и выдает предупреждения.
var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc: `Запрещает прямой вызов os.Exit в функции main пакета main.

Анализатор находит вызовы os.Exit в функции main и выдает предупреждения.
Прямой вызов os.Exit может нарушить корректное завершение программы,
включая отложенные функции defer и graceful shutdown.

Примеры проблемного кода:
	func main() {
		os.Exit(1) // вызовет предупреждение
	}

	func main() {
		if err != nil {
			os.Exit(1) // вызовет предупреждение
		}
	}

Рекомендуется использовать log.Fatal или возвращать ошибки из main.`,
	Run: run,
}

// run выполняет анализ кода на предмет вызовов os.Exit в функции main.
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// Проверяем, что файл принадлежит пакету main
		if file.Name.Name != "main" {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			// Ищем вызовы функций
			if callExpr, ok := node.(*ast.CallExpr); ok {
				// Проверяем, что это вызов os.Exit
				if isOsExitCall(callExpr) {
					// Проверяем, что вызов находится в функции main
					if isInMainFunction(pass, callExpr) {
						pass.Reportf(callExpr.Pos(), "прямой вызов os.Exit в функции main запрещен")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

// isOsExitCall проверяет, является ли вызов функцией os.Exit.
func isOsExitCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "os" && sel.Sel.Name == "Exit"
		}
	}
	return false
}

// isInMainFunction проверяет, находится ли вызов в функции main.
func isInMainFunction(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Получаем позицию вызова
	callPos := call.Pos()

	// Обходим все функции в файле
	for _, decl := range pass.Files[0].Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Проверяем, что это функция main
			if funcDecl.Name.Name == "main" {
				// Проверяем, что вызов находится внутри этой функции
				if callPos >= funcDecl.Pos() && callPos <= funcDecl.End() {
					return true
				}
			}
		}
	}
	return false
}
