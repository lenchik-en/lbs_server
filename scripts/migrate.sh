#!bin/bash

set -e

ecgho "Starting locate database migration..."
migrate -path migrations/locatedb -database "$DB_DSN" up

ecgho "Starting update database migration..."
migrate -path migrations/updatedb -database "$UDB_DSN" up

ecgho "Starting external database migration..."
migrate -path migrations/externaldb -database "$EDB_DSN" up

ecgho "Database migrations completed successfully."