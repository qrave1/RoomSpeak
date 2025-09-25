# RoomSpeak

## Voice Chat Application

### Tech List

- [ ] Github CI/CD
- [ ] Prometheus Metrics
- [ ] Grafana + TG Alerts
- [x] TURN servers API

### Features List

- [x] Basic WebRTC Voice Chat
- [x] Frontend UI/UX Redesign
- [x] User Authentication
- [ ] RBAC
- [x] Channel Creation and Management
- [x] Mute / Unmute + channel notifying
- [ ] Voice activity detection via WebRTC Data Channels
- [ ] Channel messaging via WebRTC Data Channels
- [ ] Frontend for mobile

## Docker

### Сборка и запуск

```bash
# Сборка образа с timestamp тегом
task docker-build

# Сборка и пуш в registry
task docker-push
```

**Примечание:** Установите переменную `REGISTRY` в `Taskfile.yml` для указания вашего registry.