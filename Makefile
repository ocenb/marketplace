fmt:
	go fmt ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

swag:
	swag init -g cmd/marketplace/main.go -o docs

test:
	docker compose -f docker-compose.test.yaml up -d --build
	go test ./tests || true
	docker compose -f docker-compose.test.yaml down -v

up:
	docker compose up -d

up-build:
	docker compose up -d --build

down:
	docker compose down