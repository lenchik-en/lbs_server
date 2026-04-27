SERVER_URL=http://localhost:8080

.PHONY: run all migrate-all migrate-external migrate-locate migrate-update psql-external psql-locate psql-update clean stop

run:
	docker compose up --build

locate_test: run
	curl -X POST $(SERVER_URL)/locate \
       -H "Content-Type: application/json" -d '{ "sessionUUID": "11111111-2222-3333-4444-555555555555", "cell": [ {"lte": {"mcc": 310,"mnc": 404,"tac": 1,"ci": 5632016 }}],"wifi": []}'

PSQL=psql
MIGRATIONS_DIR=./migrations

LOCATE_DB_URL=postgres://admin:admin@localhost:55432/locatedb?sslmode=disable
UPDATE_DB_URL=postgres://admin:admin@localhost:55433/updatedb?sslmode=disable
EXTERNAL_DB_URL=postgres://admin:admin@localhost:55434/externaldb?sslmode=disable

migrate-locate:
	@echo "Applying migrations to LocateDB..."
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locatedb/000001_create_cells_and_wifis.up.sql
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locatedb/000002_create_session_uuid.up.sql
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locatedb/000003_add_coordinates_to_wifi_ip.up.sql
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locatedb/000004_add_unique_wifi_ip.up.sql
	$(PSQL) "$(LOCATE_DB_URL)" -f $(MIGRATIONS_DIR)/locatedb/000005_create_api_keys.up.sql
	@echo "LocateDB migrations applied."

migrate-update:
	@echo "Applying migrations to UpdateDB..."
	$(PSQL) "$(UPDATE_DB_URL)" -f $(MIGRATIONS_DIR)/updatedb/000001_create_tables.up.sql
	$(PSQL) "$(UPDATE_DB_URL)" -f $(MIGRATIONS_DIR)/updatedb/000002_add_coordinates_to_wifi_ip.up.sql
	$(PSQL) "$(UPDATE_DB_URL)" -f $(MIGRATIONS_DIR)/updatedb/000003_add_type.up.sql
	@echo "UpdateDB migrations applied."

migrate-external:
	@echo "Applying migrations to ExternalDB..."
	$(PSQL) "$(EXTERNAL_DB_URL)" -f $(MIGRATIONS_DIR)/externaldb/000001_create_external_cell_db.up.sql
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