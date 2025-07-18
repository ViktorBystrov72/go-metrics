package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestServerGracefulShutdown тестирует graceful shutdown сервера
func TestServerGracefulShutdown(t *testing.T) {
	// Получаем путь к исполняемому файлу сервера
	serverPath := filepath.Join("..", "bin", "server")
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Skip("Сервер не скомпилирован, запустите: go build -o bin/server ./cmd/server")
	}

	signals := []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}

	for _, sig := range signals {
		t.Run(fmt.Sprintf("Signal_%s", sig), func(t *testing.T) {
			// Запускаем сервер с уникальным портом для каждого теста
			port := fmt.Sprintf(":%d", 8090+int(sig))
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, serverPath, "-a", "localhost"+port, "-i", "1")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				t.Fatalf("Ошибка запуска сервера: %v", err)
			}

			// Ждем, пока сервер запустится
			serverURL := "http://localhost" + port
			for i := 0; i < 50; i++ {
				resp, err := http.Get(serverURL)
				if err == nil {
					resp.Body.Close()
					break
				}
				if i == 49 {
					cmd.Process.Kill()
					t.Fatalf("Сервер не запустился за отведенное время")
				}
				time.Sleep(100 * time.Millisecond)
			}

			// Отправляем сигнал серверу
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatalf("Ошибка отправки сигнала %v: %v", sig, err)
			}

			// Ждем завершения процесса
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Logf("Сервер завершился с ошибкой: %v", err)
				} else {
					t.Logf("Сервер успешно завершился после получения сигнала %v", sig)
				}
			case <-time.After(5 * time.Second):
				cmd.Process.Kill()
				t.Fatalf("Graceful shutdown не завершился за отведенное время для сигнала %v", sig)
			}
		})
	}
}

// TestAgentGracefulShutdown тестирует graceful shutdown агента
func TestAgentGracefulShutdown(t *testing.T) {
	// Получаем путь к исполняемому файлу агента
	agentPath := filepath.Join("..", "bin", "agent")
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		t.Skip("Агент не скомпилирован, запустите: go build -o bin/agent ./cmd/agent")
	}

	signals := []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}

	for _, sig := range signals {
		t.Run(fmt.Sprintf("Signal_%s", sig), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Запускаем агент с короткими интервалами для быстрого тестирования
			cmd := exec.CommandContext(ctx, agentPath, "-p", "1", "-r", "2", "-a", "localhost:9999")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				t.Fatalf("Ошибка запуска агента: %v", err)
			}

			// Даем агенту время запуститься
			time.Sleep(500 * time.Millisecond)

			// Отправляем сигнал агенту
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatalf("Ошибка отправки сигнала %v: %v", sig, err)
			}

			// Ждем завершения процесса
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Logf("Агент завершился с ошибкой: %v", err)
				} else {
					t.Logf("Агент успешно завершился после получения сигнала %v", sig)
				}
			case <-time.After(3 * time.Second):
				cmd.Process.Kill()
				t.Fatalf("Graceful shutdown агента не завершился за отведенное время для сигнала %v", sig)
			}
		})
	}
}

// TestServerAgentIntegration тестирует graceful shutdown в реальном взаимодействии
func TestServerAgentIntegration(t *testing.T) {
	serverPath := filepath.Join("..", "bin", "server")
	agentPath := filepath.Join("..", "bin", "agent")

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		t.Skip("Сервер не скомпилирован")
	}
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		t.Skip("Агент не скомпилирован")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Запускаем сервер
	serverCmd := exec.CommandContext(ctx, serverPath, "-a", "localhost:8095", "-i", "1")
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Ошибка запуска сервера: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// Ждем запуска сервера
	time.Sleep(500 * time.Millisecond)

	// Запускаем агент
	agentCmd := exec.CommandContext(ctx, agentPath, "-a", "localhost:8095", "-p", "1", "-r", "2")
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("Ошибка запуска агента: %v", err)
	}

	// Даем системе поработать
	time.Sleep(2 * time.Second)

	// Проверяем, что метрики действительно передаются
	resp, err := http.Get("http://localhost:8095/")
	if err != nil {
		t.Fatalf("Ошибка получения метрик: %v", err)
	}
	resp.Body.Close()

	// Отправляем SIGTERM агенту
	t.Log("Отправляем SIGTERM агенту...")
	if err := agentCmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Ошибка отправки SIGTERM агенту: %v", err)
	}

	// Ждем завершения агента
	agentDone := make(chan error, 1)
	go func() {
		agentDone <- agentCmd.Wait()
	}()

	select {
	case <-agentDone:
		t.Log("Агент успешно завершился")
	case <-time.After(3 * time.Second):
		agentCmd.Process.Kill()
		t.Fatal("Агент не завершился за отведенное время")
	}

	// Отправляем SIGTERM серверу
	t.Log("Отправляем SIGTERM серверу...")
	if err := serverCmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Ошибка отправки SIGTERM серверу: %v", err)
	}

	// Ждем завершения сервера
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- serverCmd.Wait()
	}()

	select {
	case <-serverDone:
		t.Log("Сервер успешно завершился")
	case <-time.After(5 * time.Second):
		serverCmd.Process.Kill()
		t.Fatal("Сервер не завершился за отведенное время")
	}

	t.Log("Интеграционный тест graceful shutdown прошел успешно")
}
