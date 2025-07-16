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
		if file.Name.Name != "main" {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			// Проверка вызова функции
			callExpr, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Проверка не вызов os.Exit
			if !isOsExitCall(pass, callExpr) {
				return true
			}

			// Проверка вызов не в функции main
			if !isInMainFunction(file, callExpr) {
				return true
			}

			pass.Report(analysis.Diagnostic{
				Pos:     callExpr.Pos(),
				Message: "прямой вызов os.Exit в функции main запрещен",
			})
			return true
		})
	}
	return nil, nil
}

// isOsExitCall проверяет, является ли вызов функцией os.Exit.
func isOsExitCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Проверка на селектор
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// Если левая часть селектора не идентификатор
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	if pass.TypesInfo != nil && pass.TypesInfo.Uses != nil {
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if pkg := obj.Pkg(); pkg != nil {
				// Проверяем что это пакет "os" и метод "Exit"
				if pkg.Path() == "os" && sel.Sel.Name == "Exit" {
					return true
				}
			}
		}
	}

	return ident.Name == "os" && sel.Sel.Name == "Exit"
}

// isInMainFunction проверяет, находится ли вызов в функции main.
func isInMainFunction(file *ast.File, call *ast.CallExpr) bool {
	// Получаем позицию вызова
	callPos := call.Pos()

	// Обходим все функции в файле
	for _, decl := range file.Decls {
		// Если это не объявление функции
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Если это не функция main
		if funcDecl.Name.Name != "main" {
			continue
		}

		// Проверяем, что вызов находится внутри этой функции
		if callPos >= funcDecl.Pos() && callPos <= funcDecl.End() {
			return true
		}
	}
	return false
}
