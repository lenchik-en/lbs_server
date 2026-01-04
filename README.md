# LBS Server 

Прототип Location-Based Service (LBS) сервера.

Проект реализует сервер геолокации по данным сотовых сетей (GSM / WCDMA / LTE) и Wi‑Fi, IP, совместимый по входному формату с Яндекс Локатором, с использованием сторонней базы данных на этапе PoC.

TODO: - написать образец .env файла, инструкции к нему
---

## PoC

```
Client
  │
  │  JSON (cells / wifi / ip)
  ▼
LBS Server (Go)
  │
  │  Convert LocateRequest → OpenCellRequest
  ▼
UnwiredLabs / OpenCellID API
  │
  │  lat / lon / accuracy
  ▼
PostgreSQL
```

---

## Технологии

* Go
* PostgreSQL
* Docker / docker-compose
* UnwiredLabs (OpenCellID compatible API)

---

## Структура проекта

```
.
├── cmd/
│   └── server/            # точка входа
├── internal/
│   ├── app/               # HTTP handlers
│   ├── api/               # DTO + OpenCell client
│   └── db/                # DB, Logger
├── migrations/            # SQL миграции
├── docker-compose.yml
├── .env
└── README.md
```

---

## Основной API

### POST `/locate`

#### Пример запроса

```json
{
  "sessionUUID": "optional",
  "cell": [
    {
      "lte": {
        "mcc": 310,
        "mnc": 404,
        "tac": 1,
        "ci": 5632016
      }
    }
  ],
  "wifi": []
}
```

Если `sessionUUID` не передан — сервер генерирует его автоматически.

---

### Ответ

```json
{
  "sessionUUID": "generated-or-passed-uuid",
  "location": {
    "point": {
      "lat": 41.366559,
      "lon": -75.570403
    },
    "accuracy": 500
  }
}
```
---

## Запуск через Docker

```bash
docker-compose up -d
```

`.env` пример:

```
DATABASE_URL=postgres://admin:admin@localhost:5432/locatedb?sslmode=disable
OPENCELL_URL=https://eu1.unwiredlabs.com/v2/process.php
OPENCELL_API_KEY=your_api_key
```

Миграции применяются самостоятельно:

```bash
psql postgres://admin:admin@localhost:55432/locatedb?sslmode=disable \
  -f migrations/000001_create_sessions.up.sql

psql postgres://admin:admin@localhost:55432/locatedb?sslmode=disable \
  -f migrations/000002_create_session_points.up.sql
```

---

## Тестирование

```bash
curl -X POST http://localhost:8080/locate \
  -H "Content-Type: application/json" \
  -d '{
    "cell": [
      { "lte": { "mcc": 310, "mnc": 404, "tac": 1, "ci": 5632016 } }
    ]
  }'
```
--- 

#### Пример запроса

```json
    {
  "sessionUUID": "deea0dba-0000-41a1-a1b4-8b6fc342b07d",
  "cell": [
    {
      "lte": {
        "mcc": 310,
        "mnc": 404,
        "tac": 1,
        "ci": 5632016
      }
    }
  ]
}
```

#### Пример ответа

```json
    {
  "sessionUUID": "deea0dba-0000-41a1-a1b4-8b6fc342b07d",
  "location": {
    "point": {
      "lat": 41.366559,
      "lon": -75.570403
    },
    "accuracy": 500
  }
}
```


Для перекачки данных в externaldb используется команда
```postgresql
COPY cells (
radio,
mcc,
mnc,
area,
cell,
unit,
strength,
lat,
lon
)
FROM '/external_data/cell.csv'
DELIMITER ','
CSV;

```