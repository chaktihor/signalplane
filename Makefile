.PHONY: run test build demo-shop demo-traffic fmt clean

APP := signalplane
BIN := bin/$(APP)

run:
	go run ./cmd/signalplane

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/signalplane

demo-shop:
	go run ./examples/demo-shop

demo-traffic:
	curl -s "http://127.0.0.1:8088/traffic?count=12&failEvery=4"

fmt:
	gofmt -w cmd internal examples/demo-shop

clean:
	rm -rf bin
