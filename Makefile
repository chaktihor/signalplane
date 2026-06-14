.PHONY: run test build stack-up stack-down stack-logs stack-reset demo-shop demo-traffic fmt clean

APP := signalplane
BIN := bin/$(APP)

run:
	go run ./cmd/signalplane

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/signalplane

stack-up:
	docker compose up --build

stack-down:
	docker compose down

stack-logs:
	docker compose logs -f

stack-reset:
	docker compose down -v

demo-shop:
	go run ./examples/demo-shop

demo-traffic:
	curl -s "http://127.0.0.1:8088/traffic?count=12&failEvery=4"

fmt:
	gofmt -w cmd internal examples/demo-shop

clean:
	rm -rf bin
