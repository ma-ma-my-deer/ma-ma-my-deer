# Makefile
MIGRATE = ./migrate -path ./migrations -database "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable"

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-status:
	$(MIGRATE) status
