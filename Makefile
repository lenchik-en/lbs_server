TARGET=cmd/server/main.go

.PHONY: test build all

all: test

build:
	mkdir -p bin
	go build -o bin/ $(TARGET)

test: build
	./bin/main
	curl -X POST http://localhost:8080/locate \
       -H "Content-Type: application/json" -d '{ "sessionUUID": "11111111-2222-3333-4444-555555555555", "cell": [ {"lte": {"mcc": 310,"mnc": 404,"tac": 1,"ci": 5632016 }}],"wifi": []}'