# RoomSpeak

## Voice Chat Application

### Tech List

- [ ] Github CI/CD

### Features List

- [x] Basic WebRTC Voice Chat
- [ ] User Authentication + RBAC
- [ ] Room Creation and Management
- [ ] Voice activity detection via WebRTC Data Channels
- [ ] Frontend UI/UX Redesign
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

## Kubernetes развертывание

### Развертывание

```bash
# Создание namespace
kubectl apply -f k8s/namespace.yaml

# Установка cert-manager issuers (выполнить один раз)
kubectl apply -f k8s/tls/cert-issuer.yaml
kubectl apply -f k8s/tls/cert.yaml

# Секреты (заполнить перед развертыванием)
kubectl apply -f k8s/turn-secret.yaml

# Развертывание приложения
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress/ingress.yaml
```