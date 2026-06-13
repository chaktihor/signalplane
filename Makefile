.PHONY: run test build fmt clean

APP := signalplane
BIN := bin/$(APP)

run:
	go run ./cmd/signalplane

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/signalplane

fmt:
	gofmt -w cmd internal

clean:
	rm -rf bin

