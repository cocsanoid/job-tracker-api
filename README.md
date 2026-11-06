# Job Tracker API

REST API для учёта откликов на вакансии: где откликнулся, на какой стадии
процесс, сколько отказов и офферов. Написан на Go, только стандартная библиотека.

Данные хранятся в JSON-файле (`data.json`), поэтому не пропадают при перезапуске.

## Запуск

```bash
go run .
```

Сервер стартует на `http://localhost:8080`. Настройки через переменные окружения:

| Переменная  | По умолчанию | Описание               |
|-------------|--------------|------------------------|
| `PORT`      | `8080`       | порт сервера           |
| `DATA_FILE` | `data.json`  | файл для хранения данных |

## Тесты

```bash
go test -race ./...
```

## Структура

```
main.go                    точка входа, graceful shutdown
internal/job/model.go      сущность, статусы, валидация
internal/job/store.go      хранилище: память + JSON-файл, sync.RWMutex
internal/job/handler.go    HTTP-обработчики и маршруты
```

## Статусы

`wishlist` → `applied` → `interview` → `offer` / `rejected`

Дата отклика (`applied_at`) проставляется автоматически, когда статус впервые
становится отличным от `wishlist`.

## API

| Метод  | Путь                            | Описание                                   |
|--------|---------------------------------|--------------------------------------------|
| GET    | `/health`                       | проверка работоспособности                 |
| GET    | `/stats`                        | общая статистика по статусам               |
| GET    | `/applications`                 | список откликов (`?status=applied` — фильтр) |
| POST   | `/applications`                 | добавить отклик                            |
| GET    | `/applications/{id}`            | получить отклик                            |
| PUT    | `/applications/{id}`            | обновить отклик                            |
| PATCH  | `/applications/{id}/status`     | сменить только статус                      |
| DELETE | `/applications/{id}`            | удалить отклик                             |

Коды ответов: `200`, `201`, `204`, `400` (неверные данные), `404` (не найдено), `500`.

### Примеры

```bash
# добавить вакансию
curl -X POST localhost:8080/applications -d '{
  "company": "Google",
  "position": "Intership",
  "city": "London, UK",
  "status": "applied",
  "notes": "wrote a letter"
}'

# позвали на собеседование
curl -X PATCH localhost:8080/applications/1/status -d '{"status":"interview"}'

# все отклики, ожидающие ответа
curl "localhost:8080/applications?status=applied"

# статистика
curl localhost:8080/stats
```

Ответ `/stats`:

```json
{
  "total": 3,
  "by_status": {
    "applied": 2,
    "interview": 1,
    "offer": 0,
    "rejected": 0,
    "wishlist": 0
  }
}
```

## Что можно улучшить

- поиск по компании и должности;
- напоминания: отклики без ответа дольше N дней;
- хранение в базе данных вместо JSON-файла;
- простой веб-интерфейс.
