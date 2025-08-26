# RoomSpeak - MVP аналога TeamSpeak

Голосовой сервер с архитектурой SFU (Selective Forwarding Unit) для командного общения в реальном времени.

## Технический стек

- **Бэкенд**: Go (Golang)
- **Сигналинг**: WebSocket (gorilla/websocket)
- **WebRTC SFU**: Pion WebRTC
- **База данных**: PostgreSQL (pgx)
- **Фронтенд**: HTML + CSS + JavaScript (нативный WebRTC API)

## Архитектура

### SFU (Selective Forwarding Unit)
- Один PeerConnection на клиента
- Сервер принимает аудиопотоки от клиентов
- Ретранслирует RTP-потоки всем участникам комнаты
- Без декодирования и микширования аудио

### Компоненты
1. **Сигналинг-сервер** - обработка WebSocket сообщений
2. **SFU-сервер** - управление WebRTC соединениями
3. **HTTP-сервер** - статические файлы и API
4. **База данных** - хранение данных о комнатах

## Функциональность

### Пользовательские возможности
- ✅ Создание комнаты с названием
- ✅ Присоединение к комнате по ID
- ✅ Просмотр списка участников
- ✅ Голосовое общение в реальном времени
- ✅ Включение/выключение микрофона (Mute/Unmute)
- ✅ Список доступных комнат

### WebSocket API

#### Клиент → Сервер
```json
{ "action": "create_room", "data": { "roomName": "Gamers" } }
{ "action": "join_room", "data": { "roomId": "abc123" } }
{ "action": "offer", "data": { "type": "offer", "sdp": "..." } }
{ "action": "ice_candidate", "data": { "candidate": "...", "sdpMid": "...", "sdpMLineIndex": 0 } }
{ "action": "toggle_mute", "data": { "isMuted": true } }
{ "action": "get_room_users", "data": {} }
```

#### Сервер → Клиент
```json
{ "action": "connected", "data": { "clientId": "uuid" } }
{ "action": "room_created", "data": { "room_id": "uuid", "name": "..." } }
{ "action": "joined_room", "data": { "roomId": "...", "users": [...] } }
{ "action": "user_joined", "data": { "userId": "...", "userName": "...", "isMuted": false } }
{ "action": "user_left", "data": { "userId": "...", "userName": "..." } }
{ "action": "answer", "data": { "type": "answer", "sdp": "..." } }
{ "action": "ice_candidate", "data": { "candidate": "...", "sdpMid": "...", "sdpMLineIndex": 0 } }
{ "action": "mute_changed", "data": { "userId": "...", "isMuted": true } }
{ "action": "error", "data": { "message": "Текст ошибки" } }
```

## Установка и настройка

### Предварительные требования

1. **Go 1.21+**
2. **PostgreSQL 13+**
3. **Git**

### 1. Клонирование проекта

```bash
git clone https://github.com/qrave1/RoomSpeak.git
cd RoomSpeak
```

### 2. Настройка PostgreSQL

#### Создание базы данных

```sql
-- Подключитесь к PostgreSQL как суперпользователь
createdb roomspeak
```

#### Настройка переменных окружения

```bash
# Скопируйте файл с примером переменных окружения
cp env.example .env

# Отредактируйте .env файл под ваши настройки
nano .env
```

Пример конфигурации:
```env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=roomspeak
DB_USER=postgres
DB_PASSWORD=your_password
DB_SSL_MODE=disable

SERVER_HOST=localhost
SERVER_PORT=8080
STATIC_DIR=./web/static
```

### 3. Установка зависимостей Go

```bash
go mod download
go mod tidy
```

### 4. Миграция базы данных

```bash
# Выполните миграцию вручную
psql -h localhost -U postgres -d roomspeak -f migrations/001_create_rooms_table.up.sql
```

Или используйте migrate CLI:

```bash
# Установка migrate (опционально)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Выполнение миграций
migrate -path migrations -database "postgres://postgres:password@localhost:5432/roomspeak?sslmode=disable" up
```

### 5. Сборка и запуск

```bash
# Сборка
go build -o bin/roomspeak ./cmd/server

# Запуск
./bin/roomspeak
```

Или запуск напрямую:

```bash
go run ./cmd/server/main.go
```

### 6. Открытие приложения

Откройте браузер и перейдите по адресу: http://localhost:8080

## API Endpoints

### HTTP API

- `GET /api/rooms` - получить список всех комнат
- `GET /` - статические файлы (HTML, CSS, JS)
- `WS /ws` - WebSocket соединение для сигналинга

## Структура проекта

```
RoomSpeak/
├── cmd/
│   └── server/           # Точка входа приложения
│       └── main.go
├── internal/
│   ├── config/          # Конфигурация
│   ├── database/        # Слой работы с БД
│   ├── models/          # Модели данных
│   ├── sfu/            # SFU логика (Pion WebRTC)
│   └── websocket/      # WebSocket сигналинг
├── migrations/          # SQL миграции
├── web/
│   └── static/         # Фронтенд файлы
│       ├── index.html
│       ├── style.css
│       └── app.js
├── go.mod
├── go.sum
├── env.example         # Пример переменных окружения
└── README.md
```

## Разработка

### Запуск в режиме разработки

```bash
# Установите air для hot reload (опционально)
go install github.com/cosmtrek/air@latest

# Запуск с hot reload
air
```

### Тестирование

```bash
# Запуск тестов
go test ./...

# Тесты с покрытием
go test -cover ./...
```

## Docker (опционально)

### Создание Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o roomspeak ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/roomspeak .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

CMD ["./roomspeak"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: roomspeak
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  roomspeak:
    build: .
    ports:
      - "8080:8080"
    environment:
      DB_HOST: postgres
      DB_PORT: 5432
      DB_NAME: roomspeak
      DB_USER: postgres
      DB_PASSWORD: postgres
    depends_on:
      - postgres

volumes:
  postgres_data:
```

## Troubleshooting

### Частые проблемы

1. **Ошибка подключения к PostgreSQL**
   - Проверьте, что PostgreSQL запущен
   - Убедитесь в правильности настроек в .env файле
   - Проверьте права доступа пользователя к базе данных

2. **WebRTC не работает**
   - Убедитесь, что сайт доступен по HTTPS (для продакшена)
   - Проверьте настройки брандмауэра
   - Убедитесь, что браузер поддерживает WebRTC

3. **Не работает микрофон**
   - Проверьте разрешения браузера на использование микрофона
   - Убедитесь, что микрофон не используется другим приложением

### Логи

Сервер выводит подробные логи в консоль. Основные события:
- Подключения WebSocket
- Создание/присоединение к комнатам
- WebRTC соединения
- Ошибки

## Contributing

1. Fork проекта
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Создайте Pull Request

## Лицензия

MIT License

## Авторы

- [@qrave1](https://github.com/qrave1)

## Связь

По вопросам и предложениям создавайте Issues в репозитории. 