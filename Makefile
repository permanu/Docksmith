.PHONY: build test lint vet check clean fmt

BIN := bin/docksmith

build:
	go build -o $(BIN) ./cmd/docksmith

test:
	go test -race -count=1 -coverprofile=cover.out ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w .

check: fmt vet test lint

cover: test
	go tool cover -html=cover.out -o cover.html

fuzz:
	go test -fuzz=Fuzz -fuzztime=30s ./...

clean:
	rm -rf bin/ cover.out cover.html
