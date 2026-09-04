-include .env
export

MIGRATE := go run -tags 'postgres file' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
COMPOSE := docker compose --env-file .env.docker

.PHONY: run build up down logs migrate-up migrate-down migrate-force migrate-create

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

up:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

migrate-up:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

migrate-force:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" force $(version)

migrate-create:
	$(MIGRATE) create -ext sql -dir migrations -seq $(name)
