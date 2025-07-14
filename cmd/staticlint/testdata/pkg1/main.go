package main

import "os"

func main() {
	// Проблемный код - прямой вызов os.Exit в main
	os.Exit(1) // want "прямой вызов os.Exit в функции main запрещен"
}

func otherFunction() {
	// Этот вызов os.Exit не должен вызывать предупреждение
	// так как он не в функции main
	os.Exit(1)
}
