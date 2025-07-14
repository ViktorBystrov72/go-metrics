#!/bin/bash

# Скрипт для сбора профилей памяти с помощью pprof

echo "Запуск сервера для профилирования..."
./bin/server &
SERVER_PID=$!

sleep 3

echo "Сбор базового профиля памяти..."
curl -s http://localhost:8080/debug/pprof/heap > profiles/base.pprof

echo "Генерация нагрузки на сервер..."
for i in {1..1000}; do
    curl -s -X POST "http://localhost:8080/update/gauge/test_metric_$i/$i.0" > /dev/null
    curl -s -X POST "http://localhost:8080/update/counter/test_counter_$i/$i" > /dev/null
done

echo "Сбор профиля памяти после нагрузки..."
curl -s http://localhost:8080/debug/pprof/heap > profiles/result.pprof

echo "Остановка сервера..."
kill $SERVER_PID

echo "Профили сохранены в profiles/base.pprof и profiles/result.pprof"
echo "Для анализа используйте: pprof -top -diff_base=profiles/base.pprof profiles/result.pprof"
