# URL Shortener

Сервис для сокращения ссылок с возможностью выбора хранилища (PostgreSQL или in-memory).

## Функциональность

- Создание короткой ссылки (`POST /url`)
- Редирект по короткой ссылке на оригинальный URL (`GET /{alias}`)
- Поддержка двух типов хранилищ: PostgreSQL и in-memory
- Уникальность ссылок (один URL = одна короткая ссылка)
- Генерация коротких ссылок длиной 10 символов из набора: `a-z`, `A-Z`, `0-9`, `_`

## Технологии

- **Go 1.25+** - язык программирования
- **PostgreSQL** - основное хранилище
- **In-memory storage** - для тестирования и разработки
- **Chi** - HTTP роутер
- **Docker** - контейнеризация
- **Testify** - Unit-тесты

## Запуск

### Локально

```bash
go run cmd/url-shortener/main.go
```

Чтобы запустить с inmemory, надо в `config\local.yaml` поставить в storage_type: "memory",
а чтобы с postgres - "postgres"

Пример `local.yaml`:
```yaml
env: "prod"
storage_type: "postgres"  # "postgres" или "memory"
storage_path: "postgres://postgres:123@localhost:5432/urlshortener?sslmode=disable"
http_server:
  address: "localhost:8082"
  timeout: 4s
  idle_timeout: 60s
```

### Doccker

Запуск с PostgreSQL:

```bash
docker-compose up -d
```


Запуск только приложения с in-memory хранилищем
```bash
docker run -p 8082:8082 \
  -v $(pwd)/config:/root/config \
  -e CONFIG_PATH=/root/config/memory.yaml \
  url-shortener:latest
```

## API Endpoints

Создание короткой ссылки:
POST `/url`

Request:
```json
{
    "url": "https://example.com",
    "alias": "custom"  // опционально
}
```

Response:
```json
{
    "status": "OK",
    "alias": "custom"  // или сгенерированный
}
```

При отправке уже существующего url или alias будет ответ `409 Conflict`:
```json
{
    "status": "Error",
    "error": "url already exists"
}
```

Коды ответов `POST /url`:

| Код | Когда |
|-----|-------|
| `201 Created` | ссылка сохранена |
| `400 Bad Request` | невалидный JSON, нет поля `url` или это не URL |
| `409 Conflict` | такой `url` или `alias` уже есть |
| `500 Internal Server Error` | ошибка хранилища |

Получение оригинального URL
GET `/{alias}`

| Код | Когда |
|-----|-------|
| `302 Found` | редирект, адрес в заголовке `Location` |
| `404 Not Found` | алиас не найден, JSON `{"status":"Error","error":"not found"}` |
| `500 Internal Server Error` | ошибка хранилища |

## Тесты

Юнит-тесты и функциональные тесты (`tests/`) запускаются одной командой.
Функциональные тесты сами поднимают сервис с in-memory хранилищем и ходят в него по HTTP:

```bash
go test -race ./...
```

Прогнать функциональные тесты против уже запущенного сервиса (например, из docker-compose с PostgreSQL):

```bash
URL_SHORTENER_BASE_URL=http://localhost:8082 go test ./tests/...
```


## Хранилища
1. PostgreSQL:
- Данные сохраняются на диске
- Таблица создается автоматически при первом запуске
- Уникальность обеспечивается на уровне БД

2. In-memory:
- Данные хранятся в оперативной памяти
- Потеря данных при перезапуске
- Потокобезопасность через `sync.RWMutex`
