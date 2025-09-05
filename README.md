# RoomSpeak

## Voice Chat Application

### Tech List

- [ ] Github CI/CD
- [ ] Grafana
- [x] TURN servers API

### Features List

- [x] Basic WebRTC Voice Chat
- [x] Frontend UI/UX Redesign
- [ ] User Authentication + RBAC
- [ ] Room Creation and Management
- [ ] Voice activity detection via WebRTC Data Channels
- [ ] Room messaging via WebRTC Data Channels

## Docker

### Сборка и запуск

```bash
# Сборка образа с timestamp тегом
task docker-build

# Сборка и пуш в registry
task docker-push
```

**Примечание:** Установите переменную `REGISTRY` в `Taskfile.yml` для указания вашего registry.