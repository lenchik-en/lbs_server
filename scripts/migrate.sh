#!bin/bash

set -e

echo "Starting locate database migration..."
migrate -path ../migrations/locatedb -database postgres://admin:admin@localhost:55432/locatedb?sslmode=disable up

echo "Starting update database migration..."
migrate -path ../migrations/updatedb -database postgres://admin:admin@localhost:55433/updatedb?sslmode=disable up

echo "Starting external database migration..."
migrate -path ../migrations/externaldb -database postgres://admin:admin@localhost:55434/externaldb?sslmode=disable up

echo "Database migrations completed successfully."
