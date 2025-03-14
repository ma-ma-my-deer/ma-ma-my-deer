# Makefile
MIGRATE = ./migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
MIGRATE_TEST = ./migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-status:
	$(MIGRATE) status

migrate-test-up:
	$(MIGRATE_TEST) up

migrate-test-down:
	$(MIGRATE_TEST) down

build:
	go build -o main .

run: build
	./main

apitest: migrate-test-up
	env $(shell cat .env.test | xargs) go test -v ./test
	$(MIGRATE_TEST) down

