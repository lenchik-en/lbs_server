SERVER_URL=http://localhost:8080

.PHONY: locate_test run all migrate-all migrate-external migrate-locate migrate-update psql-external psql-locate psql-update clean stop

all: locate_test

run:
	docker compose up --build

locate_test: run
	curl -X POST $(SERVER_URL)/locate \
       -H "Content-Type: application/json" -d '{ "sessionUUID": "11111111-2222-3333-4444-555555555555", "cell": [ {"lte": {"mcc": 310,"mnc": 404,"tac": 1,"ci": 5632016 }}],"wifi": []}'

# =========================
# Миграции и работа с бд
# =========================
PSQL=psql
MIGRATIONS_DIR=./migrations

LOCATE_DB_URL=postgres://admin:admin@localhost:55432/locatedb?sslmode=disable
UPDATE_DB_URL=postgres://admin:admin@localhost:55433/updatedb?sslmode=disable
EXTERNAL_DB_URL=postgres://admin:admin@localhost:55434/externaldb?sslmode=disable

migrate-locate:
	@echo "Applying migrations to LocateDB..."
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locate/001_init.sql
	@echo "LocateDB migrations applied."

migrate-update:
	@echo "Applying migrations to UpdateDB..."
	$(PSQL) "$(UPDATE_DB_URL)" -f $(MIGRATIONS_DIR)/update/001_init.sql
	@echo "UpdateDB migrations applied."

migrate-external:
	@echo "Applying migrations to ExternalDB..."
	$(PSQL) "$(EXTERNAL_DB_URL)" -f $(MIGRATIONS_DIR)/external/001_init.sql
	@echo "ExternalDB migrations applied."

migrate-all:
	make migrate-locate
	make migrate-update
	make migrate-external

psql-locate:
	$(PSQL) "$(LOCATE_DB_URL)"

psql-update:
	$(PSQL) "$(UPDATE_DB_URL)"

psql-external:
	$(PSQL) "$(EXTERNAL_DB_URL)"

clean:
	docker compose down -v

stop:
	docker compose down