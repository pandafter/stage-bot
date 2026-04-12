.PHONY: run build test clean lint

run:
	go run ./cmd/bot/

build:
	go build -o dist/bot ./cmd/bot/

test:
	go test ./... -v

clean:
	rm -rf dist/ data/*.db

lint:
	go vet ./...
