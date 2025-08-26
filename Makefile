# RoomSpeak Makefile

.PHONY: help build run clean test docker-build docker-run docker-stop setup-db migrate-up migrate-down

# Переменные
BINARY_NAME=roomspeak
MAIN_PATH=./cmd/server
BUILD_DIR=./bin
DOCKER_IMAGE=roomspeak:latest

# По умолчанию показываем help
.DEFAULT_GOAL := help

help: ## Показать справку по командам
	@echo "RoomSpeak - MVP аналога TeamSpeak"
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup: ## Первоначальная настройка проекта
	@echo "🔧 Настройка проекта..."
	go mod download
	go mod tidy
	mkdir -p $(BUILD_DIR)
	@echo "✅ Проект настроен!"

build: ## Собрать приложение
	@echo "🔨 Сборка приложения..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Приложение собрано: $(BUILD_DIR)/$(BINARY_NAME)"

run: build ## Запустить приложение
	@echo "🚀 Запуск приложения..."
	$(BUILD_DIR)/$(BINARY_NAME)

dev: ## Запуск в режиме разработки (с hot reload)
	@echo "🔄 Запуск в режиме разработки..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "❌ Air не установлен. Установите: go install github.com/cosmtrek/air@latest"; \
		echo "🔄 Запуск без hot reload..."; \
		go run $(MAIN_PATH); \
	fi

test: ## Запустить тесты
	@echo "🧪 Запуск тестов..."
	go test -v ./...

test-coverage: ## Запустить тесты с покрытием
	@echo "🧪 Запуск тестов с покрытием..."
	go test -v -cover ./...

lint: ## Запустить линтер
	@echo "🔍 Проверка кода..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "❌ golangci-lint не установлен. Установите: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

clean: ## Очистить собранные файлы
	@echo "🧹 Очистка..."
	rm -rf $(BUILD_DIR)
	go clean
	@echo "✅ Очистка завершена"

# Docker команды
docker-build: ## Собрать Docker образ
	@echo "🐳 Сборка Docker образа..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker образ собран: $(DOCKER_IMAGE)"

docker-run: ## Запустить с помощью Docker Compose
	@echo "🐳 Запуск через Docker Compose..."
	docker-compose up --build -d
	@echo "✅ Сервисы запущены!"
	@echo "🌐 Приложение: http://localhost:8080"

docker-stop: ## Остановить Docker контейнеры
	@echo "🛑 Остановка Docker контейнеров..."
	docker-compose down
	@echo "✅ Контейнеры остановлены"

docker-logs: ## Показать логи Docker контейнеров
	@echo "📊 Логи контейнеров:"
	docker-compose logs -f

# База данных
setup-db: ## Настроить базу данных PostgreSQL
	@echo "🗄️  Настройка базы данных..."
	@echo "Убедитесь, что PostgreSQL запущен и создайте базу данных 'roomspeak'"
	@echo "Выполните: createdb roomspeak"

migrate-up: ## Применить миграции базы данных
	@echo "⬆️  Применение миграций..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/roomspeak?sslmode=disable" up; \
	else \
		echo "❌ migrate не установлен. Установите или выполните миграции вручную:"; \
		echo "psql -h localhost -U postgres -d roomspeak -f migrations/001_create_rooms_table.up.sql"; \
	fi

migrate-down: ## Откатить миграции базы данных
	@echo "⬇️  Откат миграций..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/roomspeak?sslmode=disable" down; \
	else \
		echo "❌ migrate не установлен. Выполните откат миграций вручную:"; \
		echo "psql -h localhost -U postgres -d roomspeak -f migrations/001_create_rooms_table.down.sql"; \
	fi

# Установка инструментов разработки
install-tools: ## Установить инструменты разработки
	@echo "🛠️  Установка инструментов разработки..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✅ Инструменты установлены!"

# Быстрый старт
quick-start: ## Быстрый старт с Docker
	@echo "🚀 Быстрый старт RoomSpeak..."
	@if [ -f "scripts/run.sh" ]; then \
		./scripts/run.sh; \
	else \
		make docker-run; \
	fi

# Проверка состояния
status: ## Показать статус сервисов
	@echo "📊 Статус сервисов:"
	@if command -v docker-compose > /dev/null; then \
		docker-compose ps; \
	else \
		echo "❌ docker-compose не найден"; \
	fi 