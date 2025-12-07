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