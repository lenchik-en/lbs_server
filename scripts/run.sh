#!/bin/bash
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN_NAME="lbs-server"

echo "==> Starting LBS Server bootstrap"
cd "$PROJECT_ROOT"

# 1. Проверка .env
if [ ! -f .env ]; then
  echo "ERROR: .env file not found"
  echo "Create it manually based on .env.example"
  exit 1
fi

# 2. Экспорт env
set -a
source .env
set +a

echo "==> Environment loaded"

# 3. Установка зависимостей
echo "==> Installing dependencies"
./scripts/install.sh

# 4. Сборка
echo "==> Building server"
./scripts/build.sh

# 5. Миграции
echo "==> Applying migrations"
./scripts/migrate.sh

# 6. Запуск
echo "==> Starting server"
exec ./bin/${BIN_NAME}
