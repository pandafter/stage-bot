.PHONY: run build test clean

run:
	go run cmd/server/main.go

build:
	go build -o dist/bot cmd/server/main.go

test:
	go test ./... -v

clean:
	rm -rf dist/ data/*.db
