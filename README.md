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

При отправке уже существующего url будет ответ:
```json
{
    "status": "Error",
    "error": "url already exists"
}
```

Получение оригинального URL
GET `/{alias}`
При успехе: HTTP 302 Redirect на оригинальный URL
При ошибке: JSON с описанием ошибки

Request:
non

Response:
html по указанному адресу


## Хранилища
1. PostgreSQL:
- Данные сохраняются на диске
- Таблица создается автоматически при первом запуске
- Уникальность обеспечивается на уровне БД

2. In-memory:
- Данные хранятся в оперативной памяти
- Потеря данных при перезапуске
- Потокобезопасность через `sync.RWMutex`
