# Система сбора телеметрии - Backend

## Быстрый старт

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f backend

# Проверка health endpoint
curl http://localhost:8080/health

# Остановка
docker-compose down