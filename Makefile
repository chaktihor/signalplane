.PHONY: run test build stack-up stack-down stack-logs stack-agent-logs stack-reset demo-shop demo-traffic fmt clean

APP := signalplane
BIN := bin/$(APP)
CONTAINER_COMPOSE ?= podman compose

run:
	go run ./cmd/signalplane

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/signalplane

stack-up:
	$(CONTAINER_COMPOSE) up --build

stack-down:
	$(CONTAINER_COMPOSE) down

stack-logs:
	$(CONTAINER_COMPOSE) logs -f

stack-agent-logs:
	$(CONTAINER_COMPOSE) logs -f demo-log-writer signalplane-log-agent

stack-reset:
	$(CONTAINER_COMPOSE) down -v

demo-shop:
	go run ./examples/demo-shop

demo-traffic:
	curl -s "http://127.0.0.1:8088/traffic?count=12&failEvery=4"

fmt:
	gofmt -w cmd internal examples/demo-shop

clean:
	rm -rf bin
