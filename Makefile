.PHONY: build run docker-build docker-run docker-compose-up docker-compose-down clean test

APP_NAME = glitchy

build:
	go build -o $(APP_NAME)

run:
	go run main.go

docker-build:
	docker build -t $(APP_NAME) .

docker-run:
	docker run -p 8080:8080 --env-file .env -v ./glitchy.pem:/app/keys/glitchy.pem:ro $(APP_NAME)

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

clean:
	rm -f $(APP_NAME)

test:
	go test -v ./...

deps:
	go mod tidy

init:
	cp -n .env.example .env || true