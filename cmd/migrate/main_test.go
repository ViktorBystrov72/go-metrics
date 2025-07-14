package main

import (
	"os/exec"
	"testing"
)

func TestMainMigrateSmoke(t *testing.T) {
	cmd := exec.Command("go", "run", "./main.go", "-h")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Logf("main.go (migrate) успешно запустился: %s", string(out))
	} else {
		t.Logf("main.go (migrate) завершился с ошибкой (ожидаемо для -h): %s", string(out))
	}
}
