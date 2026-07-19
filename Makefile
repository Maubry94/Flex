.PHONY: build dev down test

build:
	docker compose build

dev:
	docker compose -f compose.dev.yml up --build

down:
	docker compose -f compose.dev.yml down

test:
	npm test
	cd apps/server && go test ./...

