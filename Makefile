.PHONY: build run docker-build docker-run docker-compose-up docker-compose-down clean test

APP_NAME = glitchy

build:
	go build -o $(APP_NAME) ./cmd/glitchy

run:
	go run ./cmd/glitchy

docker-build:
	docker build -t $(APP_NAME) .

docker-run:
	@if [ ! -f glitchy.pem ]; then \
		echo "Error: glitchy.pem not found in current directory"; \
		echo "Place your GitHub App private key in the current directory with filename 'glitchy.pem'"; \
		exit 1; \
	fi
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found in current directory"; \
		echo "Create a .env file with the required environment variables"; \
		exit 1; \
	fi
	docker run -p 8080:8080 --env-file .env -e GITHUB_APP_PRIVATE_KEY_PATH=/app/keys/glitchy.pem -v ./glitchy.pem:/app/keys/glitchy.pem:ro $(APP_NAME)

clean:
	rm -f $(APP_NAME)

test:
	go test -v ./...

deps:
	go mod tidy