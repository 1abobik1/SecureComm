.PHONY: gen-keys build up up-rebuild

gen-keys:
	@bash scripts/generate_keys.sh

build: gen-keys
	docker-compose build

up: build
	docker-compose up

up-rebuild: build
	docker-compose up --build

create-docs:
	swag init -g cmd/server/main.go -o docs